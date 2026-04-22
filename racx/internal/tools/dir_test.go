package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirListsRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "x")
	_ = os.Mkdir(filepath.Join(root, "sub"), 0o755)
	s, v := makeSandbox(t, root)

	resp, err := Dir(s, v, DirRequest{})
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("Total = %d, want 2", resp.Total)
	}
	if resp.Entries[0].Name != "sub" || resp.Entries[0].Type != "dir" {
		t.Errorf("first entry should be directory 'sub', got %+v", resp.Entries[0])
	}
	if resp.Entries[1].Name != "a.txt" || resp.Entries[1].Type != "file" {
		t.Errorf("second entry should be file 'a.txt', got %+v", resp.Entries[1])
	}
}

func TestDirHidesDotfilesByDefault(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "visible.txt"), "x")
	writeFile(t, filepath.Join(root, ".hidden"), "x")
	s, v := makeSandbox(t, root)

	resp, _ := Dir(s, v, DirRequest{})
	for _, e := range resp.Entries {
		if e.Name == ".hidden" {
			t.Fatal(".hidden should be suppressed")
		}
	}
}

func TestDirDeniedOutsideRoot(t *testing.T) {
	root := t.TempDir()
	s, v := makeSandbox(t, root)

	_, err := Dir(s, v, DirRequest{Path: "../etc"})
	if err == nil || err.Code != ErrDeniedByPolicy {
		t.Fatalf("expected ErrDeniedByPolicy, got %v", err)
	}
}
