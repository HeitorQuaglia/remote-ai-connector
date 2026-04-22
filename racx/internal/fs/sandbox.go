// Package fs fornece sandbox, filtros de visibilidade e detecção de arquivos binários.
package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Sandbox struct {
	root string
}

func NewSandbox(root string) (*Sandbox, error) {
	if !filepath.IsAbs(root) {
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("resolve absolute root: %w", err)
		}
		root = abs
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("eval symlinks on root: %w", err)
	}
	return &Sandbox{root: resolved}, nil
}

func (s *Sandbox) Root() string { return s.root }

var ErrOutsideRoot = errors.New("path escapes project root")

// Resolve returns the absolute, symlink-resolved path for input, validating
// it stays within the sandbox root. If the path does not exist, it returns
// the lexically-cleaned absolute path (without following symlinks) so callers
// can surface not_found errors instead of escape errors.
func (s *Sandbox) Resolve(input string) (string, error) {
	if filepath.IsAbs(input) {
		return "", ErrOutsideRoot
	}

	joined := filepath.Join(s.root, input)
	cleaned := filepath.Clean(joined)
	if !withinRoot(cleaned, s.root) {
		return "", ErrOutsideRoot
	}

	evaluated, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			return cleaned, nil
		}
		return "", fmt.Errorf("eval symlinks: %w", err)
	}
	if !withinRoot(evaluated, s.root) {
		return "", ErrOutsideRoot
	}
	return evaluated, nil
}

func withinRoot(p, root string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..")
}
