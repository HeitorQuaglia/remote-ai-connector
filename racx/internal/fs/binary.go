package fs

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

const binarySniffBytes = 8 * 1024

// IsBinary returns true if content contains a NUL byte within the first
// binarySniffBytes bytes. NUL is never present in UTF-8 text.
func IsBinary(content []byte) bool {
	limit := len(content)
	if limit > binarySniffBytes {
		limit = binarySniffBytes
	}
	return bytes.IndexByte(content[:limit], 0) >= 0
}

func IsBinaryFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	buf := make([]byte, binarySniffBytes)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	return IsBinary(buf[:n]), nil
}
