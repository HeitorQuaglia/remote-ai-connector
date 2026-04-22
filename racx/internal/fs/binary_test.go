package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsBinaryDetectsNullByte(t *testing.T) {
	if !IsBinary([]byte{0x41, 0x00, 0x42}) {
		t.Error("expected null-byte content to be binary")
	}
}

func TestIsBinaryAllowsText(t *testing.T) {
	if IsBinary([]byte("hello world\nfoo\n")) {
		t.Error("expected ASCII text to be non-binary")
	}
}

func TestIsBinaryAllowsEmpty(t *testing.T) {
	if IsBinary(nil) {
		t.Error("empty content must not be reported as binary")
	}
}

func TestIsBinaryAllowsUTF8(t *testing.T) {
	if IsBinary([]byte("olá, mundo — ações")) {
		t.Error("UTF-8 text must not be reported as binary")
	}
}

func TestIsBinaryFileDetectsBinaryFile(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin.dat")
	if err := os.WriteFile(bin, []byte{0x7f, 0x45, 0x4c, 0x46, 0x00, 0x01}, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := IsBinaryFile(bin)
	if err != nil {
		t.Fatalf("IsBinaryFile: %v", err)
	}
	if !got {
		t.Error("expected ELF-like file to be detected as binary")
	}
}

func TestIsBinaryFileAllowsText(t *testing.T) {
	root := t.TempDir()
	txt := filepath.Join(root, "a.txt")
	if err := os.WriteFile(txt, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := IsBinaryFile(txt)
	if err != nil {
		t.Fatalf("IsBinaryFile: %v", err)
	}
	if got {
		t.Error("expected ASCII text file to not be binary")
	}
}
