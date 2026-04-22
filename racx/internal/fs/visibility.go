package fs

import (
	"path/filepath"
	"strings"
)

type HideReason string

const (
	ReasonNone      HideReason = ""
	ReasonDenylist  HideReason = "denylist"
	ReasonDotfile   HideReason = "dotfile"
	ReasonGitignore HideReason = "gitignore"
)

type Visibility struct {
	gitignore     *GitignoreMatcher
	includeHidden bool
}

func NewVisibility(gi *GitignoreMatcher, includeHidden bool) *Visibility {
	return &Visibility{gitignore: gi, includeHidden: includeHidden}
}

// IsVisible decides whether relPath should be exposed through the tools.
// The order of checks matters:
//   1. denylist (hardcoded, always wins)
//   2. dotfile (suppressed unless includeHidden)
//   3. gitignore
func (v *Visibility) IsVisible(relPath string) (bool, HideReason) {
	if IsDenied(relPath) {
		return false, ReasonDenylist
	}
	if !v.includeHidden && hasDotSegment(relPath) {
		return false, ReasonDotfile
	}
	if v.gitignore != nil && v.gitignore.IsIgnored(relPath) {
		return false, ReasonGitignore
	}
	return true, ReasonNone
}

func hasDotSegment(relPath string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(relPath), "/") {
		if seg != "" && seg != "." && strings.HasPrefix(seg, ".") {
			return true
		}
	}
	return false
}

// Gitignore exposes the underlying matcher so callers can build a new
// Visibility with different flags while reusing the compiled patterns.
func (v *Visibility) Gitignore() *GitignoreMatcher {
	return v.gitignore
}
