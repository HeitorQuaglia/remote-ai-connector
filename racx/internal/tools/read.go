package tools

import (
    "bufio"
    "errors"
    "io"
    "os"
    "path/filepath"
    "strings"

    fspkg "github.com/HeitorQuaglia/remote-ai-connector/racx/internal/fs"
)

// Read implements the `read` tool.
func Read(s *fspkg.Sandbox, v *fspkg.Visibility, req ReadRequest) (*ReadResponse, *Error) {
    if req.Path == "" {
        return nil, NewError(ErrInvalidArgument, "path is required")
    }

    resolved, err := s.Resolve(req.Path)
    if err != nil {
        if errors.Is(err, fspkg.ErrOutsideRoot) {
            return nil, NewError(ErrDeniedByPolicy, "path escapes project root")
        }
        return nil, NewError(ErrIOError, err.Error())
    }

    rel, _ := filepath.Rel(s.Root(), resolved)
    relSlash := filepath.ToSlash(rel)
    if visible, reason := v.IsVisible(relSlash); !visible {
        return nil, NewError(ErrDeniedByPolicy, "hidden by policy: "+string(reason))
    }

    info, statErr := os.Stat(resolved)
    if statErr != nil {
        if os.IsNotExist(statErr) {
            return nil, NewError(ErrNotFound, "file not found")
        }
        return nil, NewError(ErrIOError, statErr.Error())
    }
    if info.IsDir() {
        return nil, NewError(ErrIsDirectory, "path is a directory")
    }
    if info.Size() > ReadHardMaxBytes {
        return nil, NewError(ErrFileTooLarge, "file exceeds 10MB cap")
    }

    binary, binErr := fspkg.IsBinaryFile(resolved)
    if binErr != nil {
        return nil, NewError(ErrIOError, binErr.Error())
    }
    if binary {
        return nil, NewError(ErrBinaryFile, "file is binary")
    }

    return readText(resolved, req)
}

func readText(path string, req ReadRequest) (*ReadResponse, *Error) {
    f, openErr := os.Open(path)
    if openErr != nil {
        return nil, NewError(ErrIOError, openErr.Error())
    }
    defer f.Close()

    offset := 1
    if req.Offset != nil && *req.Offset >= 1 {
        offset = *req.Offset
    }
    limit := ReadDefaultLineLimit
    if req.Limit != nil && *req.Limit > 0 {
        limit = *req.Limit
    }

    scanner := bufio.NewScanner(f)
    scanner.Buffer(make([]byte, 64*1024), 1024*1024)

    var out strings.Builder
    collected := 0
    total := 0
    emittedStart := 0
    emittedEnd := 0
    byteLimitHit := false

    for scanner.Scan() {
        total++
        if total < offset {
            continue
        }
        if collected >= limit {
            continue
        }
        line := scanner.Text()
        if out.Len()+len(line)+1 > ReadMaxBytes {
            byteLimitHit = true
            continue
        }
        if emittedStart == 0 {
            emittedStart = total
        }
        emittedEnd = total
        out.WriteString(line)
        out.WriteString("\n")
        collected++
    }
    if err := scanner.Err(); err != nil && err != io.EOF {
        return nil, NewError(ErrIOError, err.Error())
    }

    truncated := byteLimitHit || collected >= limit && total > emittedEnd || total > emittedEnd
    if emittedStart == 0 {
        emittedStart = offset
        emittedEnd = offset - 1
    }

    return &ReadResponse{
        Content:       out.String(),
        TotalLines:    total,
        Truncated:     truncated,
        ReturnedRange: ReadRange{Start: emittedStart, End: emittedEnd},
    }, nil
}
