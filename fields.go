// Generic [field.*] segments: resolve an arbitrary dotted path out of the raw
// stdin JSON and format its scalar value. Visualization formats (bar/dots/pie)
// are handled in fieldSeg (render.go) since they also need width/color.

package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// resolvePath walks a dotted path (e.g. "cost.total_cost_usd") through nested
// JSON objects. Any missing key, non-object midway, or null leaf → (nil, false).
func resolvePath(raw map[string]any, path string) (any, bool) {
	if raw == nil || path == "" {
		return nil, false
	}
	var cur any = raw
	for _, p := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := m[p]
		if !ok {
			return nil, false
		}
		cur = v
	}
	if cur == nil {
		return nil, false
	}
	return cur, true
}

// vizFormats render the value as a graphic gauge rather than text.
func isVizFormat(format string) bool {
	switch format {
	case "bar", "dots", "pie":
		return true
	}
	return false
}

// formatScalar turns a resolved value into display text for the non-viz formats.
// A false second return means "nothing worth showing" (wrong type / empty).
func formatScalar(v any, format string) (string, bool) {
	switch format {
	case "usd":
		f, ok := toFloat(v)
		if !ok {
			return "", false
		}
		return strconv.FormatFloat(f, 'f', 2, 64), true
	case "tokens":
		f, ok := toFloat(v)
		if !ok || f < 0 {
			return "", false
		}
		return fmtTokens(uint64(f)), true
	case "duration":
		f, ok := toFloat(v)
		if !ok {
			return "", false
		}
		return fmtDuration(f), true
	case "percent":
		f, ok := toFloat(v)
		if !ok {
			return "", false
		}
		return strconv.FormatInt(int64(math.Round(f)), 10) + "%", true
	case "epoch":
		f, ok := toFloat(v)
		if !ok {
			return "", false
		}
		return time.Unix(int64(f), 0).Local().Format("Jan 2 15:04"), true
	default: // "" — raw value, lightly cleaned up
		return rawString(v)
	}
}

// rawString renders a JSON scalar with no special formatting. Whole floats lose
// their ".0"; empty strings report false so the segment drops out.
func rawString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, t != ""
	case bool:
		return strconv.FormatBool(t), true
	case float64:
		if t == math.Trunc(t) && !math.IsInf(t, 0) {
			return strconv.FormatInt(int64(t), 10), true
		}
		return strconv.FormatFloat(t, 'f', -1, 64), true
	default:
		return "", false // arrays/objects unsupported
	}
}

// toFloat accepts the numeric types JSON decoding and tests produce.
func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	default:
		return 0, false
	}
}

// fmtDuration humanizes a millisecond count, showing the two largest nonzero
// units: 45000 -> "45s", 90000 -> "1m30s", 3_600_000 -> "1h".
func fmtDuration(ms float64) string {
	s := int(ms / 1000)
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	s %= 60
	if m < 60 {
		if s > 0 {
			return fmt.Sprintf("%dm%ds", m, s)
		}
		return fmt.Sprintf("%dm", m)
	}
	h := m / 60
	m %= 60
	if h < 24 {
		if m > 0 {
			return fmt.Sprintf("%dh%dm", h, m)
		}
		return fmt.Sprintf("%dh", h)
	}
	d := h / 24
	h %= 24
	if h > 0 {
		return fmt.Sprintf("%dd%dh", d, h)
	}
	return fmt.Sprintf("%dd", d)
}
