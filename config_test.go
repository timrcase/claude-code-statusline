package main

import (
	"os"
	"reflect"
	"testing"
)

// Drift guard: config.example.toml documents the defaults, so parsing it must
// yield exactly the embedded defaults. If a default changes, change the
// example too (and vice versa).
func TestExampleConfigMatchesEmbeddedDefaults(t *testing.T) {
	raw, err := os.ReadFile("config.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	c := parseConfig(string(raw), "config.example.toml")
	d := DefaultConfig()
	if !reflect.DeepEqual(c, d) {
		t.Errorf("example config drifted from defaults:\n got: %+v\nwant: %+v", c, d)
	}
}

func TestEmptyConfigIsAllDefaults(t *testing.T) {
	c := parseConfig("", "test")
	if !reflect.DeepEqual(c, DefaultConfig()) {
		t.Errorf("got %+v", c)
	}
}

func TestPartialSectionKeepsOtherDefaults(t *testing.T) {
	c := parseConfig("[context]\nok_color = \"123456\"\n", "test")
	if c.Context.OkColor != "123456" {
		t.Errorf("ok_color = %q", c.Context.OkColor)
	}
	if c.Context.WarnColor != "e6c800" {
		t.Errorf("warn_color should keep default, got %q", c.Context.WarnColor)
	}
	if c.Context.Width != 5 || !c.Context.Counts {
		t.Errorf("context non-color defaults disturbed: %+v", c.Context)
	}
	if c.Directory.Color != "2e9599" {
		t.Errorf("other sections disturbed: %+v", c.Directory)
	}
}

func TestDirectoryFlagsCanBeDisabled(t *testing.T) {
	c := parseConfig("[directory]\ndiff = false\n", "test")
	if c.Directory.Diff {
		t.Error("diff should be false")
	}
	if !c.Directory.Git || !c.Directory.Worktree {
		t.Errorf("unset flags should stay true: %+v", c.Directory)
	}
}

func TestLayoutLinesOrderAndSeparator(t *testing.T) {
	raw := `
[layout]
separator = " / "
line2 = ["limit_5h", "limit_7d"]
line1 = ["model", "context"]
`
	c := parseConfig(raw, "test")
	if c.Layout.Separator != " / " {
		t.Errorf("separator = %q", c.Layout.Separator)
	}
	want := [][]string{{"model", "context"}, {"limit_5h", "limit_7d"}}
	if !reflect.DeepEqual(c.Layout.Lines, want) {
		t.Errorf("lines = %v, want %v", c.Layout.Lines, want)
	}
}

func TestLayoutSeparatorOnlyKeepsDefaultLines(t *testing.T) {
	c := parseConfig("[layout]\nseparator = \" · \"\n", "test")
	if c.Layout.Separator != " · " {
		t.Errorf("separator = %q", c.Layout.Separator)
	}
	if !reflect.DeepEqual(c.Layout.Lines, DefaultConfig().Layout.Lines) {
		t.Errorf("lines should keep defaults, got %v", c.Layout.Lines)
	}
}

func TestUnknownLayoutEntryIsSkippedNotFatal(t *testing.T) {
	c := parseConfig("[layout]\nline1 = [\"model\", \"flux_capacitor\", \"effort\"]\n", "test")
	want := [][]string{{"model", "effort"}}
	if !reflect.DeepEqual(c.Layout.Lines, want) {
		t.Errorf("lines = %v, want %v", c.Layout.Lines, want)
	}
}

func TestCustomSegmentReferencedWithoutSectionIsSkipped(t *testing.T) {
	c := parseConfig("[layout]\nline1 = [\"model\", \"custom.missing\"]\n", "test")
	want := [][]string{{"model"}}
	if !reflect.DeepEqual(c.Layout.Lines, want) {
		t.Errorf("lines = %v, want %v", c.Layout.Lines, want)
	}
}

func TestCustomSectionParsesAndDefaultsTimeout(t *testing.T) {
	raw := `
[layout]
line1 = ["custom.clock"]

[custom.clock]
command = "date"
`
	c := parseConfig(raw, "test")
	want := CustomCfg{Command: "date", TimeoutMs: 300}
	if got := c.Custom["clock"]; got != want {
		t.Errorf("custom.clock = %+v, want %+v", got, want)
	}
	if !reflect.DeepEqual(c.Layout.Lines, [][]string{{"custom.clock"}}) {
		t.Errorf("lines = %v", c.Layout.Lines)
	}
}

func TestCustomWithoutCommandIsSkipped(t *testing.T) {
	c := parseConfig("[custom.broken]\ntimeout_ms = 100\n", "test")
	if _, ok := c.Custom["broken"]; ok {
		t.Error("custom section without command should be dropped")
	}
}

func TestUnknownBarStyleFallsBackToBlocks(t *testing.T) {
	c := parseConfig("[context]\nbar = \"lasers\"\n", "test")
	if c.Context.Bar != BarBlocks {
		t.Errorf("bar = %q, want blocks", c.Context.Bar)
	}
}

func TestBadSectionFallsBackToItsDefaults(t *testing.T) {
	// width must be an integer; the whole [context] section reverts.
	c := parseConfig("[context]\nwidth = \"wide\"\nok_color = \"123456\"\n", "test")
	if !reflect.DeepEqual(c.Context, DefaultConfig().Context) {
		t.Errorf("context should revert to defaults, got %+v", c.Context)
	}
}

func TestUnparseableFileFallsBackToDefaults(t *testing.T) {
	c := parseConfig("this is not toml [[[", "test")
	if !reflect.DeepEqual(c, DefaultConfig()) {
		t.Errorf("got %+v", c)
	}
}
