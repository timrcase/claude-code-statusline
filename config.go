// TOML config: $XDG_CONFIG_HOME/claude-code-statusline/config.toml, falling
// back to ~/.config/claude-code-statusline/config.toml. Missing file →
// embedded defaults. Whole-file parse failure → warn to stderr, use defaults.
// A malformed section or unknown layout entry is skipped with a warning
// instead of failing the file.
//
// Starship-style schema: each segment is a named section that only configures
// appearance; the [layout] stanza owns presence, order, line placement, and
// the separator. A segment renders iff it appears in a layout line.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Layout    Layout
	Model     ModelCfg
	Directory DirectoryCfg
	Context   ContextCfg
	Effort    EffortCfg
	Limit5h   LimitCfg
	Limit7d   LimitCfg
	Custom    map[string]CustomCfg
	Fields    map[string]FieldCfg
}

type Layout struct {
	// Separator is rendered dim between segments.
	Separator string
	// Lines holds segment names per statusline row, from line1, line2, …
	Lines [][]string
}

type ModelCfg struct {
	Color string `toml:"color"`
}

type DirectoryCfg struct {
	Git           bool   `toml:"git"`
	Diff          bool   `toml:"diff"`
	Worktree      bool   `toml:"worktree"`
	Color         string `toml:"color"`
	BranchColor   string `toml:"branch_color"`
	WorktreeColor string `toml:"worktree_color"`
	AddsColor     string `toml:"adds_color"`
	DelsColor     string `toml:"dels_color"`
}

// Thresholds are the usage-gradient colors shared by bar-bearing sections;
// each section carries its own overridable copy.
type Thresholds struct {
	OkColor       string `toml:"ok_color"`
	WarnColor     string `toml:"warn_color"`
	HotColor      string `toml:"hot_color"`
	CriticalColor string `toml:"critical_color"`
}

type ContextCfg struct {
	Bar         BarStyle `toml:"bar"`
	Width       int      `toml:"width"`
	Counts      bool     `toml:"counts"`
	Percent     bool     `toml:"percent"`
	CountsColor string   `toml:"counts_color"`
	Thresholds
}

type EffortCfg struct {
	LabelColor  string `toml:"label_color"`
	MediumColor string `toml:"medium_color"`
	HighColor   string `toml:"high_color"`
	XhighColor  string `toml:"xhigh_color"`
	MaxColor    string `toml:"max_color"`
}

type LimitCfg struct {
	Bar        BarStyle `toml:"bar"`
	Width      int      `toml:"width"`
	Reset      bool     `toml:"reset"`
	LabelColor string   `toml:"label_color"`
	Thresholds
}

// CustomCfg is the escape hatch: user-provided script output as a segment,
// declared as [custom.name] and referenced in layout as "custom.name".
type CustomCfg struct {
	Command   string `toml:"command"`
	TimeoutMs uint64 `toml:"timeout_ms"`
}

// FieldCfg is a declarative segment reading any dotted path out of the stdin
// JSON, declared as [field.name] and referenced in layout as "field.name". A
// scalar format (usd/tokens/duration/percent/epoch or raw) renders text; a viz
// format (bar/dots/pie) renders a gauge colored by the usage gradient unless an
// explicit color is set.
type FieldCfg struct {
	Path    string `toml:"path"`
	Symbol  string `toml:"symbol"`
	Format  string `toml:"format"`
	Color   string `toml:"color"`
	Suffix  string `toml:"suffix"`
	Width   int    `toml:"width"`   // cells for bar/dots
	Percent bool   `toml:"percent"` // append "NN%" after a viz gauge
	Thresholds
}

const defaultTimeoutMs = 300

func defaultThresholds() Thresholds {
	return Thresholds{
		OkColor:       "00a000",
		WarnColor:     "e6c800",
		HotColor:      "ffb055",
		CriticalColor: "ff5555",
	}
}

func defaultLimit() LimitCfg {
	return LimitCfg{
		Bar:        BarBlocks,
		Width:      5,
		Reset:      true,
		LabelColor: "dcdcdc",
		Thresholds: defaultThresholds(),
	}
}

func DefaultConfig() Config {
	return Config{
		Layout: Layout{
			Separator: " | ",
			Lines: [][]string{
				{"model", "directory", "context", "effort", "limit_5h", "limit_7d"},
			},
		},
		Model: ModelCfg{Color: "0099ff"},
		Directory: DirectoryCfg{
			Git:           true,
			Diff:          true,
			Worktree:      true,
			Color:         "2e9599",
			BranchColor:   "00a000",
			WorktreeColor: "ffb055",
			AddsColor:     "00a000",
			DelsColor:     "ff5555",
		},
		Context: ContextCfg{
			Bar:         BarBlocks,
			Width:       5,
			Counts:      true,
			Percent:     true,
			CountsColor: "ffb055",
			Thresholds:  defaultThresholds(),
		},
		Effort: EffortCfg{
			LabelColor:  "dcdcdc",
			MediumColor: "ffb055",
			HighColor:   "00a000",
			XhighColor:  "a78bfa",
			MaxColor:    "ff5555",
		},
		Limit5h: defaultLimit(),
		Limit7d: defaultLimit(),
		Custom:  map[string]CustomCfg{},
		Fields:  map[string]FieldCfg{},
	}
}

func warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "claude-code-statusline: "+format+"\n", args...)
}

func loadConfig() Config {
	path, ok := configPath()
	if !ok {
		return DefaultConfig()
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig()
	}
	return parseConfig(string(raw), path)
}

// configPath uses env vars only — no home-dir library.
func configPath() (string, bool) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return "", false
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "claude-code-statusline", "config.toml"), true
}

// fileConfig defers each section so a bad one can fall back to its defaults
// without failing the whole file.
type fileConfig struct {
	Layout    toml.Primitive            `toml:"layout"`
	Model     toml.Primitive            `toml:"model"`
	Directory toml.Primitive            `toml:"directory"`
	Context   toml.Primitive            `toml:"context"`
	Effort    toml.Primitive            `toml:"effort"`
	Limit5h   toml.Primitive            `toml:"limit_5h"`
	Limit7d   toml.Primitive            `toml:"limit_7d"`
	Custom    map[string]toml.Primitive `toml:"custom"`
	Field     map[string]toml.Primitive `toml:"field"`
}

func parseConfig(raw, origin string) Config {
	cfg := DefaultConfig()
	var fc fileConfig
	md, err := toml.Decode(raw, &fc)
	if err != nil {
		warnf("bad config %s: %v; using defaults", origin, err)
		return cfg
	}

	decodeSection(md, origin, "model", fc.Model, &cfg.Model)
	decodeSection(md, origin, "directory", fc.Directory, &cfg.Directory)
	decodeSection(md, origin, "context", fc.Context, &cfg.Context)
	decodeSection(md, origin, "effort", fc.Effort, &cfg.Effort)
	decodeSection(md, origin, "limit_5h", fc.Limit5h, &cfg.Limit5h)
	decodeSection(md, origin, "limit_7d", fc.Limit7d, &cfg.Limit7d)

	for name, prim := range fc.Custom {
		cc := CustomCfg{TimeoutMs: defaultTimeoutMs}
		if err := md.PrimitiveDecode(prim, &cc); err != nil {
			warnf("bad [custom.%s] in %s: %v; skipping", name, origin, err)
			continue
		}
		if cc.Command == "" {
			warnf("[custom.%s] in %s has no command; skipping", name, origin)
			continue
		}
		cfg.Custom[name] = cc
	}

	for name, prim := range fc.Field {
		f := FieldCfg{Width: 5, Thresholds: defaultThresholds()}
		if err := md.PrimitiveDecode(prim, &f); err != nil {
			warnf("bad [field.%s] in %s: %v; skipping", name, origin, err)
			continue
		}
		if f.Path == "" {
			warnf("[field.%s] in %s has no path; skipping", name, origin)
			continue
		}
		cfg.Fields[name] = f
	}

	if md.IsDefined("layout") {
		var layoutRaw map[string]any
		if err := md.PrimitiveDecode(fc.Layout, &layoutRaw); err != nil {
			warnf("bad [layout] in %s: %v; using default layout", origin, err)
		} else {
			applyLayout(&cfg.Layout, layoutRaw, origin)
		}
	}

	cfg.normalize(origin)
	return cfg
}

// decodeSection overlays a present section onto its defaults; on error the
// section keeps all defaults.
func decodeSection[T any](md toml.MetaData, origin, name string, prim toml.Primitive, into *T) {
	if !md.IsDefined(name) {
		return
	}
	tmp := *into
	if err := md.PrimitiveDecode(prim, &tmp); err != nil {
		warnf("bad [%s] in %s: %v; using defaults for it", name, origin, err)
		return
	}
	*into = tmp
}

var lineKeyRe = regexp.MustCompile(`^line(\d+)$`)

// applyLayout merges a [layout] table: separator if present, and lineN keys
// (in ascending numeric order) replacing the default lines when any exist.
func applyLayout(l *Layout, raw map[string]any, origin string) {
	if v, ok := raw["separator"]; ok {
		if s, ok := v.(string); ok {
			l.Separator = s
		} else {
			warnf("layout separator in %s must be a string; keeping %q", origin, l.Separator)
		}
	}
	type numbered struct {
		n    int
		segs []string
	}
	var lines []numbered
	for key, val := range raw {
		if key == "separator" {
			continue
		}
		m := lineKeyRe.FindStringSubmatch(key)
		if m == nil {
			warnf("unknown [layout] key %q in %s; ignoring", key, origin)
			continue
		}
		items, ok := val.([]any)
		if !ok {
			warnf("layout %s in %s must be an array of segment names; ignoring", key, origin)
			continue
		}
		segs := []string{}
		for _, it := range items {
			s, ok := it.(string)
			if !ok {
				warnf("layout %s in %s contains a non-string entry; skipping it", key, origin)
				continue
			}
			segs = append(segs, s)
		}
		n, _ := strconv.Atoi(m[1])
		lines = append(lines, numbered{n, segs})
	}
	if len(lines) == 0 {
		return
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].n < lines[j].n })
	l.Lines = make([][]string, 0, len(lines))
	for _, ln := range lines {
		l.Lines = append(l.Lines, ln.segs)
	}
}

var knownSegments = map[string]bool{
	"model":     true,
	"directory": true,
	"context":   true,
	"effort":    true,
	"limit_5h":  true,
	"limit_7d":  true,
}

// normalize validates bar styles and drops unrenderable layout entries.
func (cfg *Config) normalize(origin string) {
	fixBar := func(section string, b *BarStyle) {
		switch *b {
		case BarBlocks, BarDots, BarNone:
		default:
			warnf("unknown bar style %q in [%s] of %s; using blocks", string(*b), section, origin)
			*b = BarBlocks
		}
	}
	fixBar("context", &cfg.Context.Bar)
	fixBar("limit_5h", &cfg.Limit5h.Bar)
	fixBar("limit_7d", &cfg.Limit7d.Bar)

	validFormats := map[string]bool{
		"": true, "usd": true, "tokens": true, "duration": true,
		"percent": true, "epoch": true, "bar": true, "dots": true, "pie": true,
	}
	for name, f := range cfg.Fields {
		changed := false
		if !validFormats[f.Format] {
			warnf("unknown format %q in [field.%s] of %s; showing raw value", f.Format, name, origin)
			f.Format, changed = "", true
		}
		if f.Width <= 0 {
			f.Width, changed = 5, true
		}
		if changed {
			cfg.Fields[name] = f
		}
	}

	for i, line := range cfg.Layout.Lines {
		kept := make([]string, 0, len(line))
		for _, name := range line {
			switch {
			case knownSegments[name]:
			case strings.HasPrefix(name, "custom."):
				key := strings.TrimPrefix(name, "custom.")
				if _, exists := cfg.Custom[key]; !exists {
					warnf("layout references %q but no [custom.%s] section exists in %s; skipping", name, key, origin)
					continue
				}
			case strings.HasPrefix(name, "field."):
				key := strings.TrimPrefix(name, "field.")
				if _, exists := cfg.Fields[key]; !exists {
					warnf("layout references %q but no [field.%s] section exists in %s; skipping", name, key, origin)
					continue
				}
			default:
				warnf("unknown segment %q in layout of %s; skipping", name, origin)
				continue
			}
			kept = append(kept, name)
		}
		cfg.Layout.Lines[i] = kept
	}
}
