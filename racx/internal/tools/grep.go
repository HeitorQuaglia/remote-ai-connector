package tools

import (
    "bufio"
    "errors"
    "io/fs"
    "os"
    "path/filepath"
    "regexp"

    "github.com/bmatcuk/doublestar/v4"

    fspkg "github.com/heitor/remote-ai-connector/racx/internal/fs"
)

func Grep(s *fspkg.Sandbox, v *fspkg.Visibility, req GrepRequest) (*GrepResponse, *Error) {
    if req.Pattern == "" {
        return nil, NewError(ErrInvalidArgument, "pattern is required")
    }
    re, err := regexp.Compile(req.Pattern)
    if err != nil {
        return nil, NewError(ErrInvalidRegex, err.Error())
    }

    context := 0
    if req.ContextLines != nil {
        context = *req.ContextLines
    }
    if context < 0 {
        context = 0
    }
    if context > GrepMaxContext {
        context = GrepMaxContext
    }

    // Visibility local respects IncludeHidden from the request.
    local := v
    if req.IncludeHidden {
        local = fspkg.NewVisibility(v.Gitignore(), true)
    }

    searchRoot := s.Root()
    if req.Path != "" {
        resolved, resErr := s.Resolve(req.Path)
        if resErr != nil {
            if errors.Is(resErr, fspkg.ErrOutsideRoot) {
                return nil, NewError(ErrDeniedByPolicy, "path escapes project root")
            }
            return nil, NewError(ErrIOError, resErr.Error())
        }
        searchRoot = resolved
    }

    resp := &GrepResponse{Matches: []GrepMatch{}}

    walkErr := filepath.WalkDir(searchRoot, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return nil // best-effort: skip unreadable entries
        }
        rel, relErr := filepath.Rel(s.Root(), path)
        if relErr != nil {
            return nil
        }
        relSlash := filepath.ToSlash(rel)

        if d.IsDir() {
            if rel == "." {
                return nil
            }
            if visible, _ := local.IsVisible(relSlash); !visible {
                return filepath.SkipDir
            }
            return nil
        }

        if visible, _ := local.IsVisible(relSlash); !visible {
            return nil
        }
        if req.Include != "" {
            matched, _ := doublestar.PathMatch(req.Include, relSlash)
            if !matched {
                return nil
            }
        }
        if binary, _ := fspkg.IsBinaryFile(path); binary {
            return nil
        }

        fileMatches(path, relSlash, re, context, resp)
        return nil
    })
    if walkErr != nil {
        return nil, NewError(ErrIOError, walkErr.Error())
    }

    resp.Total = len(resp.Matches)
    if len(resp.Matches) > GrepMaxMatches {
        resp.Matches = resp.Matches[:GrepMaxMatches]
        resp.Truncated = true
    }
    return resp, nil
}

func fileMatches(path, relSlash string, re *regexp.Regexp, context int, resp *GrepResponse) {
    f, err := os.Open(path)
    if err != nil {
        return
    }
    defer f.Close()

    var lines []string
    scanner := bufio.NewScanner(f)
    scanner.Buffer(make([]byte, 64*1024), 1024*1024)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }

    for i, line := range lines {
        loc := re.FindStringIndex(line)
        if loc == nil {
            continue
        }
        before := sliceLines(lines, i-context, i)
        after := sliceLines(lines, i+1, i+1+context)
        resp.Matches = append(resp.Matches, GrepMatch{
            File:   relSlash,
            Line:   i + 1,
            Column: loc[0] + 1,
            Text:   line,
            Before: before,
            After:  after,
        })
    }
}

func sliceLines(lines []string, start, end int) []string {
    if start < 0 {
        start = 0
    }
    if end > len(lines) {
        end = len(lines)
    }
    if start >= end {
        return nil
    }
    out := make([]string, end-start)
    copy(out, lines[start:end])
    return out
}
