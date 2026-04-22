package fs

import (
	"path/filepath"
	"strings"
)

// Os padrões abaixo são verificados antes de qualquer outro filtro e
// não podem ser desabilitados nem mesmo com include_hidden=true.
var denylistPatterns = []string{
	".env",
	".env.*",
	"*.pem",
	"*.key",
	"*.crt",
	"*_rsa",
	"*_rsa.pub",
	"*_ed25519",
	"*_ed25519.pub",
	"*_ecdsa",
	"*_ecdsa.pub",
	"*_dsa",
	"*_dsa.pub",
}

var denylistDirs = []string{
	".ssh",
	".gnupg",
}

var denylistPrefixPaths = []string{
	".git/objects/",
	".git/refs/",
	".aws/credentials",
}

var denylistExactPaths = []string{
	".git/HEAD",
}

// IsDenied reports whether relPath matches any hardcoded secret pattern.
// relPath is expected to be relative to the project root, using forward
// slashes (we normalise with filepath.ToSlash internally).
func IsDenied(relPath string) bool {
	p := filepath.ToSlash(relPath)
	base := filepath.Base(p)

	for _, pat := range denylistPatterns {
		if matched, _ := filepath.Match(pat, base); matched {
			return true
		}
	}

	segments := strings.Split(p, "/")
	for _, seg := range segments {
		for _, dir := range denylistDirs {
			if seg == dir {
				return true
			}
		}
	}

	for _, pref := range denylistPrefixPaths {
		if strings.HasPrefix(p, pref) {
			return true
		}
	}

	for _, exact := range denylistExactPaths {
		if p == exact {
			return true
		}
	}

	return false
}
