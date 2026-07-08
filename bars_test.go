package main

import (
	"testing"
	"unicode/utf8"
)

func blocksPlain(pct float64, width int) string {
	return stripANSI(blocks(pct, width, "00a000"))
}

func dotsPlain(pct float64, width int) string {
	return stripANSI(dots(pct, width, "00a000"))
}

func TestBlocksFillMath(t *testing.T) {
	cases := []struct {
		pct   float64
		width int
		want  string
	}{
		{0.0, 5, "░░░░░"},
		{50.0, 5, "██▌░░"}, // 20/40 eighths
		{56.0, 5, "██▊░░"}, // 22.4 -> 22 eighths
		{100.0, 5, "█████"},
		{2.0, 5, "▏░░░░"}, // 0.8 -> 1 eighth
		// Rounding at a cell boundary: 60% of 5 cells is exactly 3 cells.
		{60.0, 5, "███░░"},
		// 98% -> 39.2 -> 39 eighths: 4 full cells + 7/8 partial, no phantom 6th cell.
		{98.0, 5, "████▉"},
		// 99% -> 39.6 rounds to the full 40 eighths.
		{99.0, 5, "█████"},
	}
	for _, c := range cases {
		if got := blocksPlain(c.pct, c.width); got != c.want {
			t.Errorf("blocks(%v, %d) = %q, want %q", c.pct, c.width, got, c.want)
		}
	}
}

func TestBlocksAlwaysWidthCells(t *testing.T) {
	for _, pct := range []int{0, 1, 13, 49, 50, 51, 87, 99, 100} {
		for _, width := range []int{1, 3, 5, 10} {
			plain := blocksPlain(float64(pct), width)
			if got := utf8.RuneCountInString(plain); got != width {
				t.Errorf("pct=%d width=%d: %d cells (%q)", pct, width, got, plain)
			}
		}
	}
}

func TestDotsFillMath(t *testing.T) {
	cases := []struct {
		pct  float64
		want string
	}{
		{0.0, "○○○○○"},
		{50.0, "●●●○○"}, // 2.5 rounds up
		{56.0, "●●●○○"},
		{100.0, "●●●●●"},
	}
	for _, c := range cases {
		if got := dotsPlain(c.pct, 5); got != c.want {
			t.Errorf("dots(%v, 5) = %q, want %q", c.pct, got, c.want)
		}
	}
}

func TestNoneStyleHasNoBar(t *testing.T) {
	if _, ok := bar(BarNone, 56.0, 5, "00a000"); ok {
		t.Error("bar(none) should report no bar")
	}
}

func TestOutOfRangePctIsClamped(t *testing.T) {
	over, _ := bar(BarBlocks, 250.0, 5, "0")
	if got := stripANSI(over); got != "█████" {
		t.Errorf("pct 250 = %q", got)
	}
	under, _ := bar(BarBlocks, -3.0, 5, "0")
	if got := stripANSI(under); got != "░░░░░" {
		t.Errorf("pct -3 = %q", got)
	}
}

func TestTokenFormatting(t *testing.T) {
	cases := []struct {
		n    uint64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1_000, "1k"},
		{112_213, "112k"},
		{200_000, "200k"},
		{999_999, "999k"},
		{1_000_000, "1m"},
		{1_500_000, "1.5m"},
		{2_000_000, "2m"},
	}
	for _, c := range cases {
		if got := fmtTokens(c.n); got != c.want {
			t.Errorf("fmtTokens(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}
