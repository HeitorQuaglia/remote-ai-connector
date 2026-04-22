package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitignoreMatchesRootPatterns(t *testing.T) {
	root := t.TempDir()
	gitignore := "node_modules/\n*.pyc\n!keep.pyc\nbuild/\n"
	mustWrite(t, filepath.Join(root, ".gitignore"), gitignore)

	m, err := LoadGitignore(root)
	if err != nil {
		t.Fatalf("LoadGitignore: %v", err)
	}

	cases := []struct {
		path string
		want bool
	}{
		{"node_modules/foo/bar.js", true},
		{"app.pyc", true},
		{"keep.pyc", false},
		{"build/output", true},
		{"src/main.py", false},
		{".gitignore", false},
	}
	for _, tc := range cases {
		if got := m.IsIgnored(tc.path); got != tc.want {
			t.Errorf("IsIgnored(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestGitignoreAbsentReturnsNoopMatcher(t *testing.T) {
	root := t.TempDir() // no .gitignore here
	m, err := LoadGitignore(root)
	if err != nil {
		t.Fatalf("LoadGitignore (no file): %v", err)
	}
	if m.IsIgnored("anything") {
		t.Error("empty matcher must not ignore anything")
	}
}

func TestGitignoreMissingRootErrors(t *testing.T) {
	if _, err := LoadGitignore(filepath.Join(os.TempDir(), "does-not-exist-racx")); err == nil {
		t.Fatal("expected error for missing root directory")
	}
}
