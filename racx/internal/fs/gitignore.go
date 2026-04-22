package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	ignore "github.com/sabhiram/go-gitignore"
)

type GitignoreMatcher struct {
	ig *ignore.GitIgnore
}

// LoadGitignore loads the root .gitignore file (if present). Nested
// .gitignore files are not supported in V1.
func LoadGitignore(root string) (*GitignoreMatcher, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root is not a directory")
	}

	path := filepath.Join(root, ".gitignore")
	_, err = os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return &GitignoreMatcher{ig: ignore.CompileIgnoreLines()}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat .gitignore: %w", err)
	}

	ig, err := ignore.CompileIgnoreFile(path)
	if err != nil {
		return nil, fmt.Errorf("compile .gitignore: %w", err)
	}
	return &GitignoreMatcher{ig: ig}, nil
}

// IsIgnored takes a path relative to the project root.
func (m *GitignoreMatcher) IsIgnored(relPath string) bool {
	if m == nil || m.ig == nil {
		return false
	}
	return m.ig.MatchesPath(filepath.ToSlash(relPath))
}
