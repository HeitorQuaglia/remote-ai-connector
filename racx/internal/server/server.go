// Package server expõe as 4 tools via HTTP/JSON em um mux net/http.
package server

import (
	"net/http"

	"github.com/heitor/remote-ai-connector/racx/internal/audit"
	fspkg "github.com/heitor/remote-ai-connector/racx/internal/fs"
)

type Server struct {
	sandbox    *fspkg.Sandbox
	visibility *fspkg.Visibility
	log        audit.Logger
	mux        *http.ServeMux
}

func New(s *fspkg.Sandbox, v *fspkg.Visibility, log audit.Logger) *Server {
	srv := &Server{sandbox: s, visibility: v, log: log, mux: http.NewServeMux()}
	srv.registerRoutes()
	return srv
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/ping", methodGuard("GET", s.handlePing))
	s.mux.HandleFunc("/read", methodGuard("POST", s.handleRead))
	s.mux.HandleFunc("/grep", methodGuard("POST", s.handleGrep))
	s.mux.HandleFunc("/dir", methodGuard("POST", s.handleDir))
	s.mux.HandleFunc("/tree", methodGuard("POST", s.handleTree))
}

func methodGuard(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Allow", method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}
