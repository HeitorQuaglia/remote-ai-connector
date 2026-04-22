package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/heitor/remote-ai-connector/racx/internal/audit"
	fspkg "github.com/heitor/remote-ai-connector/racx/internal/fs"
	"github.com/heitor/remote-ai-connector/racx/internal/tools"
)

func newTestServer(t *testing.T, root string) *httptest.Server {
	t.Helper()
	s, err := fspkg.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	gi, _ := fspkg.LoadGitignore(root)
	v := fspkg.NewVisibility(gi, false)
	logger := audit.NewTextLogger(io.Discard, false, time.Now)

	srv := New(s, v, logger)
	return httptest.NewServer(srv.Handler())
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPingOK(t *testing.T) {
	root := t.TempDir()
	ts := newTestServer(t, root)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d", resp.StatusCode)
	}
	var body map[string]bool
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if !body["ok"] {
		t.Errorf("ok = false, body = %+v", body)
	}
}

func TestReadRouteHappy(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "a.txt"), "hello\n")
	ts := newTestServer(t, root)
	defer ts.Close()

	body, _ := json.Marshal(tools.ReadRequest{Path: "a.txt"})
	resp, err := http.Post(ts.URL+"/read", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		out, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d, body=%s", resp.StatusCode, out)
	}
	var r tools.ReadResponse
	_ = json.NewDecoder(resp.Body).Decode(&r)
	if r.Content != "hello\n" {
		t.Errorf("content = %q", r.Content)
	}
}

func TestReadRouteDenied(t *testing.T) {
	root := t.TempDir()
	ts := newTestServer(t, root)
	defer ts.Close()

	body, _ := json.Marshal(tools.ReadRequest{Path: "../etc/passwd"})
	resp, err := http.Post(ts.URL+"/read", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Errorf("status = %d, want 403", resp.StatusCode)
	}
	var envelope struct {
		Error tools.Error `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&envelope)
	if envelope.Error.Code != tools.ErrDeniedByPolicy {
		t.Errorf("code = %q", envelope.Error.Code)
	}
}

func TestBadJSONReturns400(t *testing.T) {
	root := t.TempDir()
	ts := newTestServer(t, root)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/read", "application/json", bytes.NewReader([]byte("{not json")))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	ts := newTestServer(t, root)
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/read", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 405 {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
}
