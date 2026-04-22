package audit

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func frozenClock(tt time.Time) func() time.Time {
	return func() time.Time { return tt }
}

func TestStderrLoggerSuccess(t *testing.T) {
	var buf bytes.Buffer
	clock := frozenClock(time.Date(2026, 4, 22, 14, 32, 4, 0, time.UTC))
	l := NewTextLogger(&buf, false, clock)

	l.ToolSuccess("read", "src/app.py", "1.2KB, 42 lines")

	got := buf.String()
	if !strings.Contains(got, "[14:32:04]") {
		t.Errorf("missing timestamp in %q", got)
	}
	if !strings.Contains(got, "✓") {
		t.Errorf("missing success glyph in %q", got)
	}
	if !strings.Contains(got, "read") {
		t.Errorf("missing tool name in %q", got)
	}
	if !strings.Contains(got, "src/app.py") {
		t.Errorf("missing args in %q", got)
	}
}

func TestStderrLoggerFailure(t *testing.T) {
	var buf bytes.Buffer
	clock := frozenClock(time.Date(2026, 4, 22, 14, 32, 40, 0, time.UTC))
	l := NewTextLogger(&buf, false, clock)

	l.ToolFailure("read", "/etc/passwd", "denied_by_policy")

	got := buf.String()
	if !strings.Contains(got, "✗") {
		t.Errorf("missing failure glyph in %q", got)
	}
	if !strings.Contains(got, "denied_by_policy") {
		t.Errorf("missing error code in %q", got)
	}
}

func TestStderrLoggerQuietSuppressesOutput(t *testing.T) {
	var buf bytes.Buffer
	l := NewTextLogger(&buf, true, time.Now)

	l.ToolSuccess("read", "a", "ok")

	if buf.Len() != 0 {
		t.Errorf("quiet logger should produce no output, got %q", buf.String())
	}
}

func TestTunnelEvents(t *testing.T) {
	var buf bytes.Buffer
	l := NewTextLogger(&buf, false, time.Now)

	l.TunnelUp("racd.exemplo.com:2222")
	l.TunnelDown("connection refused")

	s := buf.String()
	if !strings.Contains(s, "tunnel up") || !strings.Contains(s, "racd.exemplo.com:2222") {
		t.Errorf("missing tunnel up info: %q", s)
	}
	if !strings.Contains(s, "tunnel down") || !strings.Contains(s, "connection refused") {
		t.Errorf("missing tunnel down info: %q", s)
	}
}
