// Bar renderers (blocks / dots / none) and number formatting.

package main

import (
	"math"
	"strconv"
	"strings"
)

type BarStyle string

const (
	BarBlocks BarStyle = "blocks"
	BarDots   BarStyle = "dots"
	BarNone   BarStyle = "none"
)

// Partial-block glyphs, 1/8 through 8/8 of a cell.
var eighthGlyphs = []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

// bar renders a usage bar. "none" has no graphic bar — the caller shows the
// number — so it reports false.
func bar(style BarStyle, pct float64, width int, colorHex string) (string, bool) {
	pct = math.Min(math.Max(pct, 0), 100)
	switch style {
	case BarNone:
		return "", false
	case BarDots:
		return dots(pct, width, colorHex), true
	default:
		return blocks(pct, width, colorHex), true
	}
}

// Sub-cell resolution: a width-cell bar has width*8 gradations.
func blocks(pct float64, width int, colorHex string) string {
	eighths := int(math.Round(pct / 100 * float64(width*8)))
	full := eighths / 8
	rem := eighths % 8
	var b strings.Builder
	b.WriteString(fg(colorHex))
	for range full {
		b.WriteRune('█')
	}
	if rem > 0 {
		b.WriteRune(eighthGlyphs[rem-1])
	}
	b.WriteString(reset)
	empty := width - full
	if rem > 0 {
		empty--
	}
	if empty > 0 {
		b.WriteString(dim)
		for range empty {
			b.WriteRune('░')
		}
		b.WriteString(reset)
	}
	return b.String()
}

// Quarter-circle pie glyphs, empty → full, for a one-cell compact gauge.
var pieGlyphs = []rune{'○', '◔', '◑', '◕', '●'}

// pie renders a percentage as a single quarter-resolution pie glyph. It needs
// no bar width and no Nerd Font — the glyphs are standard geometric shapes.
func pie(pct float64, colorHex string) string {
	pct = math.Min(math.Max(pct, 0), 100)
	i := int(math.Round(pct / 25))
	if i > 4 {
		i = 4
	}
	return fg(colorHex) + string(pieGlyphs[i]) + reset
}

func dots(pct float64, width int, colorHex string) string {
	filled := int(math.Round(pct / 100 * float64(width)))
	var b strings.Builder
	b.WriteString(fg(colorHex))
	for range filled {
		b.WriteRune('●')
	}
	b.WriteString(reset)
	if filled < width {
		b.WriteString(dim)
		for range width - filled {
			b.WriteRune('○')
		}
		b.WriteString(reset)
	}
	return b.String()
}

// >=1M -> "1.5m"/"2m" (trim .0), >=1k -> "112k", else raw.
func fmtTokens(n uint64) string {
	switch {
	case n >= 1_000_000:
		s := strconv.FormatFloat(float64(n)/1_000_000, 'f', 1, 64)
		return strings.TrimSuffix(s, ".0") + "m"
	case n >= 1_000:
		return strconv.FormatUint(n/1_000, 10) + "k"
	default:
		return strconv.FormatUint(n, 10)
	}
}
