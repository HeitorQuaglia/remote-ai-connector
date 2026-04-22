// Package audit fornece um Logger simples para stderr no formato humano
// descrito no spec. Um logger JSON em arquivo pode ser adicionado depois.
package audit

import (
	"fmt"
	"io"
	"time"
)

type Logger interface {
	TunnelUp(endpoint string)
	TunnelDown(reason string)
	ToolSuccess(tool, args, outcome string)
	ToolFailure(tool, args, errCode string)
}

type textLogger struct {
	w     io.Writer
	quiet bool
	now   func() time.Time
}

func NewTextLogger(w io.Writer, quiet bool, now func() time.Time) Logger {
	if now == nil {
		now = time.Now
	}
	return &textLogger{w: w, quiet: quiet, now: now}
}

func (l *textLogger) stamp() string {
	return l.now().Format("15:04:05")
}

func (l *textLogger) writef(format string, args ...any) {
	if l.quiet {
		return
	}
	fmt.Fprintf(l.w, format, args...)
}

func (l *textLogger) TunnelUp(endpoint string) {
	l.writef("[%s] ✓ tunnel up  %s\n", l.stamp(), endpoint)
}

func (l *textLogger) TunnelDown(reason string) {
	l.writef("[%s] ✗ tunnel down  %s\n", l.stamp(), reason)
}

func (l *textLogger) ToolSuccess(tool, args, outcome string) {
	l.writef("[%s] ✓ %s  %s  (%s)\n", l.stamp(), tool, args, outcome)
}

func (l *textLogger) ToolFailure(tool, args, errCode string) {
	l.writef("[%s] ✗ %s  %s  (%s)\n", l.stamp(), tool, args, errCode)
}
