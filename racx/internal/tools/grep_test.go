package tools

import (
    "path/filepath"
    "testing"
)

func TestGrepBasicMatch(t *testing.T) {
    root := t.TempDir()
    writeFile(t, filepath.Join(root, "a.py"), "class User:\n    pass\nclass Admin(User):\n    pass\n")
    s, v := makeSandbox(t, root)

    resp, err := Grep(s, v, GrepRequest{Pattern: "class \\w+"})
    if err != nil {
        t.Fatalf("Grep: %v", err)
    }
    if resp.Total < 2 {
        t.Errorf("expected >=2 matches, got %d", resp.Total)
    }
    if resp.Matches[0].File != "a.py" {
        t.Errorf("first match file = %q", resp.Matches[0].File)
    }
    if resp.Matches[0].Line != 1 {
        t.Errorf("first match line = %d", resp.Matches[0].Line)
    }
}

func TestGrepInvalidRegex(t *testing.T) {
    root := t.TempDir()
    s, v := makeSandbox(t, root)

    _, err := Grep(s, v, GrepRequest{Pattern: "[unclosed"})
    if err == nil || err.Code != ErrInvalidRegex {
        t.Fatalf("expected ErrInvalidRegex, got %v", err)
    }
}

func TestGrepRespectsGitignore(t *testing.T) {
    root := t.TempDir()
    writeFile(t, filepath.Join(root, ".gitignore"), "ignored/\n")
    writeFile(t, filepath.Join(root, "ignored", "a.py"), "needle\n")
    writeFile(t, filepath.Join(root, "kept", "b.py"), "needle\n")
    s, v := makeSandbox(t, root)

    resp, err := Grep(s, v, GrepRequest{Pattern: "needle"})
    if err != nil {
        t.Fatalf("Grep: %v", err)
    }
    if resp.Total != 1 {
        t.Fatalf("expected 1 match, got %d", resp.Total)
    }
    if resp.Matches[0].File != filepath.ToSlash(filepath.Join("kept", "b.py")) {
        t.Errorf("unexpected file: %q", resp.Matches[0].File)
    }
}

func TestGrepRespectsDenylist(t *testing.T) {
    root := t.TempDir()
    writeFile(t, filepath.Join(root, ".env"), "PASSWORD=needle\n")
    s, v := makeSandbox(t, root)

    resp, err := Grep(s, v, GrepRequest{Pattern: "needle", IncludeHidden: true})
    if err != nil {
        t.Fatalf("Grep: %v", err)
    }
    if resp.Total != 0 {
        t.Fatalf("denylist must block .env even with include_hidden: got %d matches", resp.Total)
    }
}

func TestGrepIncludeGlob(t *testing.T) {
    root := t.TempDir()
    writeFile(t, filepath.Join(root, "a.py"), "needle\n")
    writeFile(t, filepath.Join(root, "a.go"), "needle\n")
    s, v := makeSandbox(t, root)

    resp, err := Grep(s, v, GrepRequest{Pattern: "needle", Include: "**/*.py"})
    if err != nil {
        t.Fatalf("Grep: %v", err)
    }
    if resp.Total != 1 || resp.Matches[0].File != "a.py" {
        t.Errorf("expected only a.py, got %+v", resp.Matches)
    }
}

func TestGrepContextLines(t *testing.T) {
    root := t.TempDir()
    writeFile(t, filepath.Join(root, "a.txt"), "pre1\npre2\nhit\npost1\npost2\n")
    s, v := makeSandbox(t, root)

    ctx := 1
    resp, err := Grep(s, v, GrepRequest{Pattern: "hit", ContextLines: &ctx})
    if err != nil {
        t.Fatalf("Grep: %v", err)
    }
    m := resp.Matches[0]
    if len(m.Before) != 1 || m.Before[0] != "pre2" {
        t.Errorf("Before = %v", m.Before)
    }
    if len(m.After) != 1 || m.After[0] != "post1" {
        t.Errorf("After = %v", m.After)
    }
}

func TestGrepTruncatesAtLimit(t *testing.T) {
    root := t.TempDir()
    var content []byte
    for i := 0; i < GrepMaxMatches+10; i++ {
        content = append(content, []byte("needle\n")...)
    }
    writeFile(t, filepath.Join(root, "a.txt"), string(content))
    s, v := makeSandbox(t, root)

    resp, err := Grep(s, v, GrepRequest{Pattern: "needle"})
    if err != nil {
        t.Fatalf("Grep: %v", err)
    }
    if len(resp.Matches) != GrepMaxMatches {
        t.Errorf("len(Matches) = %d, want %d", len(resp.Matches), GrepMaxMatches)
    }
    if !resp.Truncated {
        t.Error("Truncated should be true")
    }
}
