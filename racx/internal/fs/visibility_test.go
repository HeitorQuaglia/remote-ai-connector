package fs

import (
	"path/filepath"
	"testing"
)

func TestVisibilityDenylistAlwaysWins(t *testing.T) {
	root := t.TempDir()
	gi, _ := LoadGitignore(root)
	v := NewVisibility(gi, true) // include hidden

	visible, reason := v.IsVisible(".env")
	if visible {
		t.Fatalf("expected .env to be hidden by denylist, got visible")
	}
	if reason != ReasonDenylist {
		t.Errorf("reason = %q, want %q", reason, ReasonDenylist)
	}
}

func TestVisibilityDotfileHiddenByDefault(t *testing.T) {
	root := t.TempDir()
	gi, _ := LoadGitignore(root)
	v := NewVisibility(gi, false)

	visible, reason := v.IsVisible(".github/workflows/ci.yml")
	if visible {
		t.Fatalf("expected dotfile to be hidden, got visible")
	}
	if reason != ReasonDotfile {
		t.Errorf("reason = %q, want %q", reason, ReasonDotfile)
	}
}

func TestVisibilityDotfileShownWhenIncluded(t *testing.T) {
	root := t.TempDir()
	gi, _ := LoadGitignore(root)
	v := NewVisibility(gi, true)

	visible, _ := v.IsVisible(".github/workflows/ci.yml")
	if !visible {
		t.Fatal("expected dotfile to be visible when include_hidden=true")
	}
}

func TestVisibilityRespectsGitignore(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".gitignore"), "node_modules/\n")
	gi, _ := LoadGitignore(root)
	v := NewVisibility(gi, false)

	visible, reason := v.IsVisible("node_modules/foo.js")
	if visible {
		t.Fatal("expected gitignored path to be hidden")
	}
	if reason != ReasonGitignore {
		t.Errorf("reason = %q, want %q", reason, ReasonGitignore)
	}
}

func TestVisibilityPlainFileVisible(t *testing.T) {
	root := t.TempDir()
	gi, _ := LoadGitignore(root)
	v := NewVisibility(gi, false)

	visible, _ := v.IsVisible("src/main.py")
	if !visible {
		t.Fatal("expected plain file to be visible")
	}
}
