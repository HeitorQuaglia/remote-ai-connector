package fs

import "testing"

func TestDenylistBlocksSecrets(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		// blocked
		{".env", true},
		{".env.production", true},
		{"server.pem", true},
		{"certs/server.key", true},
		{"some/nested/path/cert.crt", true},
		{"id_rsa", true},
		{"id_rsa.pub", true},
		{".ssh/id_ed25519", true},
		{".ssh/id_ed25519.pub", true},
		{".ssh/config", true},
		{".aws/credentials", true},
		{".gnupg/secring.gpg", true},
		{".git/objects/ab/cdef1234", true},
		{".git/refs/heads/main", true},
		{".git/HEAD", true},

		// allowed
		{"src/app.py", false},
		{"README.md", false},
		{".github/workflows/ci.yml", false},
		{".git/config", false},
		{".envoy.yaml", false}, // starts with .env but not .env or .env.*
		{"certificate.md", false},
	}
	for _, tc := range cases {
		if got := IsDenied(tc.path); got != tc.want {
			t.Errorf("IsDenied(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
