package tools

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	fspkg "github.com/heitor/remote-ai-connector/racx/internal/fs"
)

func Tree(s *fspkg.Sandbox, v *fspkg.Visibility, req TreeRequest) (*TreeResponse, *Error) {
	maxDepth := TreeDefaultDepth
	if req.MaxDepth != nil {
		maxDepth = *req.MaxDepth
	}
	if maxDepth < 1 {
		maxDepth = 1
	}
	if maxDepth > TreeMaxDepth {
		maxDepth = TreeMaxDepth
	}

	target := s.Root()
	if req.Path != "" {
		resolved, err := s.Resolve(req.Path)
		if err != nil {
			if errors.Is(err, fspkg.ErrOutsideRoot) {
				return nil, NewError(ErrDeniedByPolicy, "path escapes project root")
			}
			return nil, NewError(ErrIOError, err.Error())
		}
		target = resolved
	}

	local := v
	if req.IncludeHidden {
		local = fspkg.NewVisibility(v.Gitignore(), true)
	}

	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewError(ErrNotFound, "path not found")
		}
		return nil, NewError(ErrIOError, err.Error())
	}
	if !info.IsDir() {
		return nil, NewError(ErrInvalidArgument, "path is not a directory")
	}

	var sb strings.Builder
	sb.WriteString(filepath.Base(target))
	sb.WriteString("\n")
	total := 1
	truncated := false
	renderTree(&sb, s, local, target, "", 1, maxDepth, &total, &truncated)

	return &TreeResponse{
		Rendered:   sb.String(),
		TotalNodes: total,
		Truncated:  truncated,
	}, nil
}

func renderTree(
	sb *strings.Builder,
	s *fspkg.Sandbox,
	v *fspkg.Visibility,
	dir string,
	prefix string,
	depth int,
	maxDepth int,
	total *int,
	truncated *bool,
) {
	if depth > maxDepth {
		return
	}
	if *truncated {
		return
	}
	raw, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	entries := make([]os.DirEntry, 0, len(raw))
	for _, d := range raw {
		full := filepath.Join(dir, d.Name())
		rel, _ := filepath.Rel(s.Root(), full)
		if visible, _ := v.IsVisible(filepath.ToSlash(rel)); !visible {
			continue
		}
		entries = append(entries, d)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	for i, d := range entries {
		if *total >= TreeMaxNodes {
			*truncated = true
			return
		}
		isLast := i == len(entries)-1
		branch := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			branch = "└── "
			childPrefix = prefix + "    "
		}
		sb.WriteString(prefix)
		sb.WriteString(branch)
		sb.WriteString(d.Name())
		sb.WriteString("\n")
		*total++

		if d.IsDir() {
			renderTree(sb, s, v, filepath.Join(dir, d.Name()), childPrefix, depth+1, maxDepth, total, truncated)
		}
	}
}
