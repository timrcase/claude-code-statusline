package main

import (
	"strings"
	"testing"
)

// stripANSI drops ANSI escape sequences; shared test helper.
func stripANSI(s string) string {
	var out strings.Builder
	inEscape := false
	for _, c := range s {
		switch {
		case inEscape:
			if c == 'm' {
				inEscape = false
			}
		case c == '\x1b':
			inEscape = true
		default:
			out.WriteRune(c)
		}
	}
	return out.String()
}

func TestHexToANSI(t *testing.T) {
	if got := fg("0099ff"); got != "\x1b[38;2;0;153;255m" {
		t.Errorf("fg(0099ff) = %q", got)
	}
	if got := fg("#ff5555"); got != "\x1b[38;2;255;85;85m" {
		t.Errorf("fg(#ff5555) = %q", got)
	}
	// Garbage falls back to the default text grey instead of failing.
	if got := fg("nope"); got != "\x1b[38;2;220;220;220m" {
		t.Errorf("fg(nope) = %q", got)
	}
}

func TestThresholdColors(t *testing.T) {
	th := defaultThresholds()
	cases := []struct {
		pct  float64
		want string
	}{
		{0.0, th.OkColor},
		{49.9, th.OkColor},
		{50.0, th.WarnColor},
		{70.0, th.HotColor},
		{89.9, th.HotColor},
		{90.0, th.CriticalColor},
		{100.0, th.CriticalColor},
	}
	for _, c := range cases {
		if got := usageColor(th, c.pct); got != c.want {
			t.Errorf("usageColor(%v) = %q, want %q", c.pct, got, c.want)
		}
	}
}
