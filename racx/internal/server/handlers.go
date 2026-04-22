package server

import (
	"encoding/json"
	"net/http"

	"github.com/HeitorQuaglia/remote-ai-connector/racx/internal/tools"
)

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleRead(w http.ResponseWriter, r *http.Request) {
	var req tools.ReadRequest
	if !decode(w, r, &req) {
		return
	}
	resp, toolErr := tools.Read(s.sandbox, s.visibility, req)
	s.respond(w, "read", req.Path, resp, toolErr)
}

func (s *Server) handleGrep(w http.ResponseWriter, r *http.Request) {
	var req tools.GrepRequest
	if !decode(w, r, &req) {
		return
	}
	resp, toolErr := tools.Grep(s.sandbox, s.visibility, req)
	s.respond(w, "grep", req.Pattern, resp, toolErr)
}

func (s *Server) handleDir(w http.ResponseWriter, r *http.Request) {
	var req tools.DirRequest
	if !decode(w, r, &req) {
		return
	}
	resp, toolErr := tools.Dir(s.sandbox, s.visibility, req)
	s.respond(w, "dir", req.Path, resp, toolErr)
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	var req tools.TreeRequest
	if !decode(w, r, &req) {
		return
	}
	resp, toolErr := tools.Tree(s.sandbox, s.visibility, req)
	s.respond(w, "tree", req.Path, resp, toolErr)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, &tools.Error{
			Code:    tools.ErrInvalidArgument,
			Message: "invalid JSON: " + err.Error(),
		})
		return false
	}
	return true
}

func (s *Server) respond(w http.ResponseWriter, tool, args string, resp any, toolErr *tools.Error) {
	if toolErr != nil {
		s.log.ToolFailure(tool, args, string(toolErr.Code))
		writeError(w, statusFor(toolErr.Code), toolErr)
		return
	}
	s.log.ToolSuccess(tool, args, "ok")
	writeJSON(w, http.StatusOK, resp)
}

func statusFor(code tools.ErrorCode) int {
	switch code {
	case tools.ErrNotFound:
		return http.StatusNotFound
	case tools.ErrIsDirectory, tools.ErrBinaryFile, tools.ErrInvalidArgument, tools.ErrInvalidRegex, tools.ErrFileTooLarge:
		return http.StatusBadRequest
	case tools.ErrDeniedByPolicy:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, e *tools.Error) {
	writeJSON(w, status, map[string]any{"error": e})
}
