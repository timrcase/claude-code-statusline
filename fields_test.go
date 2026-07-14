package main

import (
	"strings"
	"testing"
)

func TestResolvePath(t *testing.T) {
	raw := map[string]any{
		"cost":   map[string]any{"total_cost_usd": 0.06, "total_lines_added": float64(156)},
		"vim":    map[string]any{"mode": "NORMAL"},
		"nulled": nil,
	}
	cases := []struct {
		path string
		want any
		ok   bool
	}{
		{"cost.total_cost_usd", 0.06, true},
		{"vim.mode", "NORMAL", true},
		{"cost.missing", nil, false},
		{"cost.total_cost_usd.deeper", nil, false}, // scalar can't be descended
		{"nulled", nil, false},                     // null leaf -> absent
		{"", nil, false},
		{"nope", nil, false},
	}
	for _, c := range cases {
		got, ok := resolvePath(raw, c.path)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("resolvePath(%q) = (%v, %v), want (%v, %v)", c.path, got, ok, c.want, c.ok)
		}
	}
	if _, ok := resolvePath(nil, "x"); ok {
		t.Error("nil map should resolve nothing")
	}
}

func TestFmtDuration(t *testing.T) {
	cases := []struct {
		ms   float64
		want string
	}{
		{0, "0s"},
		{30340, "30s"},
		{45000, "45s"},
		{90000, "1m30s"},
		{120000, "2m"},
		{3_600_000, "1h"},
		{5_400_000, "1h30m"},
		{90_000_000, "1d1h"},
	}
	for _, c := range cases {
		if got := fmtDuration(c.ms); got != c.want {
			t.Errorf("fmtDuration(%v) = %q, want %q", c.ms, got, c.want)
		}
	}
}

func TestFormatScalar(t *testing.T) {
	cases := []struct {
		v      any
		format string
		want   string
		ok     bool
	}{
		{0.0610695, "usd", "0.06", true},
		{float64(15500), "tokens", "15k", true}, // fmtTokens truncates in the k-range
		{float64(1_500_000), "tokens", "1.5m", true},
		{float64(45000), "duration", "45s", true},
		{23.5, "percent", "24%", true},
		{"NORMAL", "", "NORMAL", true},
		{float64(156), "", "156", true}, // whole float -> int
		{1.5, "", "1.5", true},          // fractional float kept
		{"", "", "", false},             // empty string drops
		{[]any{1, 2}, "", "", false},    // arrays unsupported
		{"text", "usd", "", false},      // non-numeric for numeric format
	}
	for _, c := range cases {
		got, ok := formatScalar(c.v, c.format)
		if ok != c.ok || got != c.want {
			t.Errorf("formatScalar(%v, %q) = (%q, %v), want (%q, %v)", c.v, c.format, got, ok, c.want, c.ok)
		}
	}
}

func TestFieldSegScalar(t *testing.T) {
	raw := map[string]any{"cost": map[string]any{"total_cost_usd": 0.0610695}}
	fc := FieldCfg{Path: "cost.total_cost_usd", Symbol: "$", Format: "usd", Color: "00a000"}
	got, ok := fieldSeg(&fc, raw)
	if !ok || stripANSI(got) != "$0.06" {
		t.Errorf("scalar field = %q (%v)", stripANSI(got), ok)
	}
}

func TestFieldSegAbsentIsBlank(t *testing.T) {
	fc := FieldCfg{Path: "pr.number", Symbol: "#"}
	if _, ok := fieldSeg(&fc, map[string]any{}); ok {
		t.Error("absent path should render nothing")
	}
}

func TestFieldSegBoolFlag(t *testing.T) {
	fc := FieldCfg{Path: "thinking.enabled", Symbol: "🧠"}
	on := map[string]any{"thinking": map[string]any{"enabled": true}}
	off := map[string]any{"thinking": map[string]any{"enabled": false}}
	if got, ok := fieldSeg(&fc, on); !ok || stripANSI(got) != "🧠" {
		t.Errorf("flag true = %q (%v)", stripANSI(got), ok)
	}
	if _, ok := fieldSeg(&fc, off); ok {
		t.Error("flag false should render nothing")
	}
}

func TestFieldSegVizPiePercent(t *testing.T) {
	fc := FieldCfg{Path: "rl.pct", Format: "pie", Percent: true, Width: 5, Thresholds: defaultThresholds()}
	raw := map[string]any{"rl": map[string]any{"pct": float64(39)}}
	got, ok := fieldSeg(&fc, raw)
	if !ok || stripANSI(got) != "◑ 39%" {
		t.Errorf("pie+percent = %q (%v)", stripANSI(got), ok)
	}
}

func TestFieldSegVizExplicitColorOverridesGradient(t *testing.T) {
	raw := map[string]any{"rl": map[string]any{"pct": float64(95)}}
	// 95% would be critical (ff5555) by gradient; explicit color must win.
	fc := FieldCfg{Path: "rl.pct", Format: "bar", Width: 5, Color: "0000ff", Thresholds: defaultThresholds()}
	got, _ := fieldSeg(&fc, raw)
	if !strings.Contains(got, fg("0000ff")) {
		t.Errorf("explicit color not used: %q", got)
	}
	if strings.Contains(got, fg("ff5555")) {
		t.Errorf("gradient color leaked in: %q", got)
	}
}

func TestFieldSegVizNonNumericIsBlank(t *testing.T) {
	fc := FieldCfg{Path: "vim.mode", Format: "bar", Width: 5, Thresholds: defaultThresholds()}
	raw := map[string]any{"vim": map[string]any{"mode": "NORMAL"}}
	if _, ok := fieldSeg(&fc, raw); ok {
		t.Error("viz format on a non-numeric value should render nothing")
	}
}
