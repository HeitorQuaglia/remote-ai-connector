// Package tools implementa as 4 ferramentas read-only expostas por racx:
// Read, Grep, Dir, Tree. Cada tool é uma função pura sobre um Sandbox e
// uma Visibility; o transporte (HTTP/JSON) vive em internal/server.
package tools

// ErrorCode é o enum estável de códigos de erro que tools retornam.
// Valores são também expostos nas descriptions do manifest MCP/OpenAPI.
type ErrorCode string

const (
	ErrNotFound        ErrorCode = "not_found"
	ErrIsDirectory     ErrorCode = "is_directory"
	ErrBinaryFile      ErrorCode = "binary_file"
	ErrDeniedByPolicy  ErrorCode = "denied_by_policy"
	ErrFileTooLarge    ErrorCode = "file_too_large"
	ErrIOError         ErrorCode = "io_error"
	ErrInvalidRegex    ErrorCode = "invalid_regex"
	ErrInvalidArgument ErrorCode = "invalid_argument"
)

type Error struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string { return string(e.Code) + ": " + e.Message }

func NewError(code ErrorCode, msg string) *Error {
	return &Error{Code: code, Message: msg}
}

// ----- Read -----

type ReadRequest struct {
	Path   string `json:"path"`
	Offset *int   `json:"offset,omitempty"` // 1-based first line, default 1
	Limit  *int   `json:"limit,omitempty"`  // max lines to return, default 2000
}

type ReadRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type ReadResponse struct {
	Content       string    `json:"content"`
	TotalLines    int       `json:"total_lines"`
	Truncated     bool      `json:"truncated"`
	ReturnedRange ReadRange `json:"returned_range"`
}

// ----- Grep -----

type GrepRequest struct {
	Pattern       string `json:"pattern"`
	Path          string `json:"path,omitempty"`
	Include       string `json:"include,omitempty"`
	ContextLines  *int   `json:"context_lines,omitempty"`
	IncludeHidden bool   `json:"include_hidden,omitempty"`
}

type GrepMatch struct {
	File   string   `json:"file"`
	Line   int      `json:"line"`
	Column int      `json:"column"`
	Text   string   `json:"text"`
	Before []string `json:"before,omitempty"`
	After  []string `json:"after,omitempty"`
}

type GrepResponse struct {
	Matches   []GrepMatch `json:"matches"`
	Total     int         `json:"total"`
	Truncated bool        `json:"truncated"`
}

// ----- Dir -----

type DirRequest struct {
	Path          string `json:"path,omitempty"`
	IncludeHidden bool   `json:"include_hidden,omitempty"`
}

type DirEntry struct {
	Name          string `json:"name"`
	Type          string `json:"type"` // "file" | "dir" | "symlink"
	Size          *int64 `json:"size,omitempty"`
	SymlinkTarget string `json:"symlink_target,omitempty"`
}

type DirResponse struct {
	Entries   []DirEntry `json:"entries"`
	Total     int        `json:"total"`
	Truncated bool       `json:"truncated"`
}

// ----- Tree -----

type TreeRequest struct {
	Path          string `json:"path,omitempty"`
	MaxDepth      *int   `json:"max_depth,omitempty"`
	IncludeHidden bool   `json:"include_hidden,omitempty"`
}

type TreeResponse struct {
	Rendered   string `json:"rendered"`
	TotalNodes int    `json:"total_nodes"`
	Truncated  bool   `json:"truncated"`
}

// ----- Limits (spec) -----

const (
	ReadDefaultLineLimit = 2000
	ReadMaxBytes         = 256 * 1024              // 256 KB per response
	ReadHardMaxBytes     = 10 * 1024 * 1024       // 10 MB absolute cap
	GrepMaxMatches       = 200
	GrepMaxContext       = 5
	DirMaxEntries        = 500
	TreeDefaultDepth     = 3
	TreeMaxDepth         = 10
	TreeMaxNodes         = 5000
)
