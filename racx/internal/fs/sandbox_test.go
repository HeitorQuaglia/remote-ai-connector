package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSandboxResolveWithinRoot(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.txt"), "hello")

	s, err := NewSandbox(root)
	if err != nil {
		t.Fatalf("NewSandbox: %v", err)
	}

	got, err := s.Resolve("a.txt")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := filepath.Join(root, "a.txt")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSandboxResolveNonexistentPath(t *testing.T) {
	root := t.TempDir()
	s, _ := NewSandbox(root)

	got, err := s.Resolve("does-not-exist.txt")
	if err != nil {
		t.Fatalf("Resolve should succeed for nonexistent: %v", err)
	}
	want := filepath.Join(root, "does-not-exist.txt")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSandboxResolveDotDotEscape(t *testing.T) {
	root := t.TempDir()
	s, _ := NewSandbox(root)

	if _, err := s.Resolve("../etc/passwd"); err == nil {
		t.Fatal("expected error for .. escape, got nil")
	}
}

func TestSandboxResolveAbsoluteOutsideRoot(t *testing.T) {
	root := t.TempDir()
	s, _ := NewSandbox(root)

	if _, err := s.Resolve("/etc/passwd"); err == nil {
		t.Fatal("expected error for absolute path outside root, got nil")
	}
}

func TestSandboxResolveSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	mustWrite(t, filepath.Join(outside, "secret.txt"), "top secret")

	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "link")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	s, _ := NewSandbox(root)
	if _, err := s.Resolve("link"); err == nil {
		t.Fatal("expected error for symlink escape, got nil")
	}
}

func TestSandboxRoot(t *testing.T) {
	root := t.TempDir()
	s, _ := NewSandbox(root)
	if s.Root() != root {
		t.Errorf("Root()=%q, want %q", s.Root(), root)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
