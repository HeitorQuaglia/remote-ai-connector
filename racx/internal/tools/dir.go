package tools

import (
	"errors"
	"os"
	"path/filepath"
	"sort"

	fspkg "github.com/heitor/remote-ai-connector/racx/internal/fs"
)

func Dir(s *fspkg.Sandbox, v *fspkg.Visibility, req DirRequest) (*DirResponse, *Error) {
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

	raw, err := os.ReadDir(target)
	if err != nil {
		return nil, NewError(ErrIOError, err.Error())
	}

	entries := make([]DirEntry, 0, len(raw))
	for _, d := range raw {
		full := filepath.Join(target, d.Name())
		rel, _ := filepath.Rel(s.Root(), full)
		relSlash := filepath.ToSlash(rel)
		if visible, _ := local.IsVisible(relSlash); !visible {
			continue
		}
		entries = append(entries, makeDirEntry(full, d))
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type != entries[j].Type {
			return entries[i].Type == "dir" // dirs first
		}
		return entries[i].Name < entries[j].Name
	})

	total := len(entries)
	truncated := false
	if total > DirMaxEntries {
		entries = entries[:DirMaxEntries]
		truncated = true
	}

	return &DirResponse{
		Entries:   entries,
		Total:     total,
		Truncated: truncated,
	}, nil
}

func makeDirEntry(full string, d os.DirEntry) DirEntry {
	e := DirEntry{Name: d.Name()}
	info, err := d.Info()
	if err != nil {
		e.Type = "file"
		return e
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		e.Type = "symlink"
		if tgt, err := os.Readlink(full); err == nil {
			e.SymlinkTarget = tgt
		}
	case info.IsDir():
		e.Type = "dir"
	default:
		e.Type = "file"
		size := info.Size()
		e.Size = &size
	}
	return e
}
