package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/shahadulhaider/restless/internal/model"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0ms"},
		{500 * time.Nanosecond, "500ns"},
		{742 * time.Microsecond, "742µs"},
		{5 * time.Millisecond, "5ms"},
		{1500 * time.Millisecond, "1.5s"},
	}
	for _, c := range cases {
		if got := formatDuration(c.d); got != c.want {
			t.Errorf("formatDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestTimingViewSubMillisecond(t *testing.T) {
	resp := &model.Response{
		Timing: model.ResponseTiming{
			TTFB:  700 * time.Microsecond,
			Total: 742 * time.Microsecond,
		},
	}
	out := stripANSI(timingView(resp))
	if strings.Contains(out, "(no timing data)") {
		t.Fatalf("sub-millisecond timing must not fall back to (no timing data): %q", out)
	}
	if !strings.Contains(out, "742µs") {
		t.Errorf("expected microsecond total in output, got %q", out)
	}
}

func TestTimingViewNoData(t *testing.T) {
	out := stripANSI(timingView(&model.Response{}))
	if !strings.Contains(out, "(no timing data)") {
		t.Errorf("zero timing should show (no timing data), got %q", out)
	}
}
