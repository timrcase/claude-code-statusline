package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Deterministic snapshot: legacy fixture has no rate limits / effort / reset
// times (no local-timezone dependence) and a cwd outside any repo.
func TestLegacyFixtureDefaultConfigSnapshot(t *testing.T) {
	p := loadFixture(t, "legacy.json")
	cfg := DefaultConfig()
	line := stripANSI(render(&p, &cfg))
	want := "Claude 3.5 Sonnet | demo | ██▊░░ 112k/200k (56%) | effort: med | 5h - | 7d -"
	if line != want {
		t.Errorf("\n got: %q\nwant: %q", line, want)
	}
}

func TestFullFixtureRendersAllSegments(t *testing.T) {
	p := loadFixture(t, "payload.json")
	// Disable git so the assertion doesn't depend on this repo's dirty state.
	cfg := DefaultConfig()
	cfg.Directory.Git = false
	cfg.Directory.Diff = false
	cfg.Directory.Worktree = false
	line := stripANSI(render(&p, &cfg))
	if !strings.HasPrefix(line, "Opus 4.8 | claude-statusline | ") {
		t.Errorf("prefix: %q", line)
	}
	for _, want := range []string{
		"▏░░░░ 26k/1m (2%)", // 2% -> 0.8 -> 1 of 40 eighths
		"effort: high",
		"5h ", "39% @",
		"7d ", "3% @",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("missing %q in %q", want, line)
		}
	}
}

func TestMinimalFixtureRendersPlaceholders(t *testing.T) {
	p := loadFixture(t, "minimal.json")
	cfg := DefaultConfig()
	line := stripANSI(render(&p, &cfg))
	want := "Sonnet 4.6 1M | ░░░░░ 0/200k (0%) | effort: med | 5h - | 7d -"
	if line != want {
		t.Errorf("\n got: %q\nwant: %q", line, want)
	}
}

func TestEmptyPayloadNeverDanglesSeparators(t *testing.T) {
	cfg := DefaultConfig()
	line := stripANSI(render(&Payload{}, &cfg))
	want := "Claude | ░░░░░ 0/200k (0%) | effort: med | 5h - | 7d -"
	if line != want {
		t.Errorf("\n got: %q\nwant: %q", line, want)
	}
}

func TestNoneBarStyleDropsGraphicBars(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Context.Bar = BarNone
	cfg.Limit5h.Bar = BarNone
	cfg.Limit7d.Bar = BarNone
	line := stripANSI(render(&Payload{}, &cfg))
	want := "Claude | 0/200k (0%) | effort: med | 5h - | 7d -"
	if line != want {
		t.Errorf("\n got: %q\nwant: %q", line, want)
	}
}

func TestContextSectionOverrides(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Layout.Lines = [][]string{{"context"}}
	cfg.Context.Bar = BarDots
	cfg.Context.Width = 10
	cfg.Context.Counts = false
	cfg.Context.Percent = false
	p := loadFixture(t, "legacy.json") // 56%
	if got := stripANSI(render(&p, &cfg)); got != "●●●●●●○○○○" {
		t.Errorf("got %q", got)
	}
}

func TestMultiLineLayout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Layout.Lines = [][]string{
		{"model", "effort"},
		{"limit_5h", "limit_7d"},
	}
	got := stripANSI(render(&Payload{}, &cfg))
	want := "Claude | effort: med\n5h - | 7d -"
	if got != want {
		t.Errorf("\n got: %q\nwant: %q", got, want)
	}
}

func TestEmptyLineIsDroppedEntirely(t *testing.T) {
	cfg := DefaultConfig()
	// directory is the only segment on line 2 and the payload has no cwd, so
	// line 2 must vanish without a blank row or dangling newline.
	cfg.Layout.Lines = [][]string{{"model"}, {"directory"}}
	got := stripANSI(render(&Payload{}, &cfg))
	if got != "Claude" {
		t.Errorf("got %q, want %q", got, "Claude")
	}
}

func TestCustomSegmentRunsAndTimesOut(t *testing.T) {
	if got, ok := customSeg("echo hi", 1000); !ok || got != "hi" {
		t.Errorf("echo: %q, %v", got, ok)
	}
	if got, ok := customSeg("printf 'a\\nb\\n'", 1000); !ok || got != "a" {
		t.Errorf("multiline: %q, %v", got, ok)
	}
	if _, ok := customSeg("exit 3", 1000); ok {
		t.Error("non-zero exit should skip the segment")
	}
	if _, ok := customSeg("true", 1000); ok {
		t.Error("empty output should skip the segment")
	}
	start := time.Now()
	if _, ok := customSeg("sleep 5", 60); ok {
		t.Error("timeout should skip the segment")
	}
	if elapsed := time.Since(start); elapsed > 1500*time.Millisecond {
		t.Errorf("timeout must kill the child, took %v", elapsed)
	}
}

func TestEffortLevelsAndUnknowns(t *testing.T) {
	cfg := DefaultConfig()
	cases := []struct{ level, shown string }{
		{"low", "low"},
		{"medium", "med"},
		{"high", "high"},
		{"xhigh", "xhigh"},
		{"max", "max"},
		{"turbo", "turbo"},
	}
	for _, c := range cases {
		var p Payload
		if err := json.Unmarshal([]byte(fmt.Sprintf(`{"effort":{"level":%q}}`, c.level)), &p); err != nil {
			t.Fatal(err)
		}
		if got := stripANSI(effortSeg(&p, &cfg)); got != "effort: "+c.shown {
			t.Errorf("level %q: got %q", c.level, got)
		}
	}
}

func TestLimitWithoutResetTimeOrData(t *testing.T) {
	cfg := DefaultConfig()
	var l Limit
	if err := json.Unmarshal([]byte(`{"used_percentage": 91.4}`), &l); err != nil {
		t.Fatal(err)
	}
	// 91.4% of 40 eighths = 36.56 -> 37 -> 4 full + 5/8
	if got := stripANSI(limitSeg(&cfg.Limit5h, "5h", &l, "15:04")); got != "5h ████▋ 91%" {
		t.Errorf("got %q", got)
	}
	if got := stripANSI(limitSeg(&cfg.Limit7d, "7d", nil, "15:04")); got != "7d -" {
		t.Errorf("got %q", got)
	}
}
