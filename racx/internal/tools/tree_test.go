package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTreeBasic(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "app"), 0o755)
	writeFile(t, filepath.Join(root, "app", "main.py"), "x")
	writeFile(t, filepath.Join(root, "README.md"), "x")
	s, v := makeSandbox(t, root)

	resp, err := Tree(s, v, TreeRequest{})
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if !strings.Contains(resp.Rendered, "app") {
		t.Error("rendered should contain 'app'")
	}
	if !strings.Contains(resp.Rendered, "main.py") {
		t.Error("rendered should contain 'main.py'")
	}
	if !strings.Contains(resp.Rendered, "README.md") {
		t.Error("rendered should contain 'README.md'")
	}
	if resp.TotalNodes < 3 {
		t.Errorf("TotalNodes = %d, want >= 3", resp.TotalNodes)
	}
}

func TestTreeRespectsDepth(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "a", "b", "c"), 0o755)
	writeFile(t, filepath.Join(root, "a", "b", "c", "deep.txt"), "x")
	s, v := makeSandbox(t, root)

	depth := 1
	resp, _ := Tree(s, v, TreeRequest{MaxDepth: &depth})
	if strings.Contains(resp.Rendered, "deep.txt") {
		t.Error("deep.txt should be beyond depth=1")
	}
	if !strings.Contains(resp.Rendered, "a") {
		t.Error("'a' should be visible at depth=1")
	}
}

func TestTreeBranchCharacters(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "one.txt"), "x")
	writeFile(t, filepath.Join(root, "two.txt"), "x")
	s, v := makeSandbox(t, root)

	resp, _ := Tree(s, v, TreeRequest{})
	if !strings.Contains(resp.Rendered, "├──") {
		t.Error("rendered should contain '├──'")
	}
	if !strings.Contains(resp.Rendered, "└──") {
		t.Error("rendered should contain '└──'")
	}
}

func TestTreeDeniedSecretsNotShown(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".env"), "SECRET=1")
	s, v := makeSandbox(t, root)

	resp, _ := Tree(s, v, TreeRequest{IncludeHidden: true})
	if strings.Contains(resp.Rendered, ".env") {
		t.Error(".env must never appear even with include_hidden")
	}
}
