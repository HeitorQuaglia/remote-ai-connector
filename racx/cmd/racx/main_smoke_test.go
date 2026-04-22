package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSmokeServerPing(t *testing.T) {
	if os.Getenv("RACX_SMOKE") == "" {
		t.Skip("RACX_SMOKE not set; skipping long-running smoke test")
	}

	projectDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(projectDir, "a.txt"), []byte("hello"), 0o644)

	bin, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	out := filepath.Join(tmp, "racx")
	build := exec.Command(bin, "build", "-o", out, "./cmd/racx")
	build.Dir = repoRoot(t)
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, b)
	}

	cmd := exec.Command(out, "--listen", "127.0.0.1:0", "--print-port")
	cmd.Dir = projectDir
	cmd.Stderr = os.Stderr
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	buf := make([]byte, 64)
	n, _ := stdout.Read(buf)
	port := strings.TrimSpace(string(buf[:n]))
	if port == "" {
		t.Fatal("expected port on stdout")
	}

	// give the server a moment
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://" + port + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	var m map[string]bool
	_ = json.Unmarshal(body, &m)
	if !m["ok"] {
		t.Fatalf("bad body: %s", body)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	// wd is racx/cmd/racx; go up twice
	return filepath.Join(wd, "..", "..")
}
