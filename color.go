// Hex → truecolor ANSI, and usage-threshold color resolution.

package main

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	dim   = "\x1b[2m"
	reset = "\x1b[0m"
)

// fg converts "0099ff" (leading '#' tolerated) to a truecolor foreground escape.
func fg(hex string) string {
	v, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	if err != nil {
		v = 0x00dcdcdc
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", (v>>16)&0xff, (v>>8)&0xff, v&0xff)
}

func paint(hex, s string) string {
	return fg(hex) + s + reset
}

// Usage color thresholds: >=90 critical, >=70 hot, >=50 warn, else ok.
func usageColor(t Thresholds, pct float64) string {
	switch {
	case pct >= 90:
		return t.CriticalColor
	case pct >= 70:
		return t.HotColor
	case pct >= 50:
		return t.WarnColor
	default:
		return t.OkColor
	}
}
