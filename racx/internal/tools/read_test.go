package tools

import (
    "os"
    "path/filepath"
    "strings"
    "testing"

    fspkg "github.com/heitor/remote-ai-connector/racx/internal/fs"
)

func makeSandbox(t *testing.T, root string) (*fspkg.Sandbox, *fspkg.Visibility) {
    t.Helper()
    s, err := fspkg.NewSandbox(root)
    if err != nil {
        t.Fatal(err)
    }
    gi, err := fspkg.LoadGitignore(root)
    if err != nil {
        t.Fatal(err)
    }
    v := fspkg.NewVisibility(gi, false)
    return s, v
}

func writeFile(t *testing.T, path, content string) {
    t.Helper()
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
        t.Fatal(err)
    }
}

func TestReadHappyPath(t *testing.T) {
    root := t.TempDir()
    writeFile(t, filepath.Join(root, "a.txt"), "line1\nline2\nline3\n")
    s, v := makeSandbox(t, root)

    resp, err := Read(s, v, ReadRequest{Path: "a.txt"})
    if err != nil {
        t.Fatalf("Read: %v", err)
    }
    if resp.Content != "line1\nline2\nline3\n" {
        t.Errorf("content = %q", resp.Content)
    }
    if resp.TotalLines != 3 {
        t.Errorf("TotalLines = %d, want 3", resp.TotalLines)
    }
    if resp.Truncated {
        t.Error("Truncated should be false")
    }
}

func TestReadNotFound(t *testing.T) {
    root := t.TempDir()
    s, v := makeSandbox(t, root)

    _, err := Read(s, v, ReadRequest{Path: "nope.txt"})
    if err == nil || err.Code != ErrNotFound {
        t.Fatalf("expected ErrNotFound, got %v", err)
    }
}

func TestReadIsDirectory(t *testing.T) {
    root := t.TempDir()
    _ = os.Mkdir(filepath.Join(root, "sub"), 0o755)
    s, v := makeSandbox(t, root)

    _, err := Read(s, v, ReadRequest{Path: "sub"})
    if err == nil || err.Code != ErrIsDirectory {
        t.Fatalf("expected ErrIsDirectory, got %v", err)
    }
}

func TestReadBinaryRejected(t *testing.T) {
    root := t.TempDir()
    writeFile(t, filepath.Join(root, "bin"), "abc\x00def")
    s, v := makeSandbox(t, root)

    _, err := Read(s, v, ReadRequest{Path: "bin"})
    if err == nil || err.Code != ErrBinaryFile {
        t.Fatalf("expected ErrBinaryFile, got %v", err)
    }
}

func TestReadDeniedByPolicy(t *testing.T) {
    root := t.TempDir()
    writeFile(t, filepath.Join(root, ".env"), "SECRET=1\n")
    s, v := makeSandbox(t, root)

    _, err := Read(s, v, ReadRequest{Path: ".env"})
    if err == nil || err.Code != ErrDeniedByPolicy {
        t.Fatalf("expected ErrDeniedByPolicy, got %v", err)
    }
}

func TestReadOutsideRoot(t *testing.T) {
    root := t.TempDir()
    s, v := makeSandbox(t, root)

    _, err := Read(s, v, ReadRequest{Path: "../../etc/passwd"})
    if err == nil || err.Code != ErrDeniedByPolicy {
        t.Fatalf("expected ErrDeniedByPolicy, got %v", err)
    }
}

func TestReadPagination(t *testing.T) {
    root := t.TempDir()
    var b strings.Builder
    for i := 1; i <= 10; i++ {
        b.WriteString("L")
        b.WriteString(numStr(i))
        b.WriteString("\n")
    }
    writeFile(t, filepath.Join(root, "big.txt"), b.String())
    s, v := makeSandbox(t, root)

    offset := 3
    limit := 2
    resp, err := Read(s, v, ReadRequest{Path: "big.txt", Offset: &offset, Limit: &limit})
    if err != nil {
        t.Fatalf("Read: %v", err)
    }
    if resp.Content != "L3\nL4\n" {
        t.Errorf("content = %q", resp.Content)
    }
    if resp.ReturnedRange.Start != 3 || resp.ReturnedRange.End != 4 {
        t.Errorf("range = %+v, want {3,4}", resp.ReturnedRange)
    }
    if resp.TotalLines != 10 {
        t.Errorf("TotalLines = %d", resp.TotalLines)
    }
    if !resp.Truncated {
        t.Error("Truncated should be true")
    }
}

func TestReadFileTooLarge(t *testing.T) {
    root := t.TempDir()
    big := make([]byte, ReadHardMaxBytes+1)
    for i := range big {
        big[i] = 'a'
    }
    writeFile(t, filepath.Join(root, "huge.txt"), string(big))
    s, v := makeSandbox(t, root)

    _, err := Read(s, v, ReadRequest{Path: "huge.txt"})
    if err == nil || err.Code != ErrFileTooLarge {
        t.Fatalf("expected ErrFileTooLarge, got %v", err)
    }
}

func numStr(i int) string {
    if i < 10 {
        return string(rune('0' + i))
    }
    return string(rune('0'+i/10)) + string(rune('0'+i%10))
}
