package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig points XDG_CONFIG_HOME at a temp dir and drops a config.toml in it.
func writeConfig(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := filepath.Join(dir, "claude-code-statusline")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckConfigReportsWarningsAndExits1(t *testing.T) {
	writeConfig(t, `
[layout]
line1 = ["model", "cost", "field.ghost", "field.rl5h"]
[field.rl5h]
path = "rate_limits.five_hour.used_percentage"
format = "pie"
`)
	var out bytes.Buffer
	code := checkConfig(&out)
	s := out.String()

	if code != 1 {
		t.Fatalf("exit %d, want 1 (warnings present)", code)
	}
	if !strings.Contains(s, "(loaded)") {
		t.Errorf("should report the config as loaded:\n%s", s)
	}
	for _, want := range []string{
		`unknown segment "cost"`,
		`field.ghost`,
		"warnings (2)",
		"line1: model, field.rl5h", // effective layout drops the bad entries
		"field sections:  rl5h",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
	// The warning prefix should be stripped in the report.
	if strings.Contains(s, "claude-code-statusline:") {
		t.Errorf("warning prefix should be stripped:\n%s", s)
	}
	// warnOut must be restored, not left pointing at the buffer.
	if warnOut == &out {
		t.Error("warnOut left dangling after checkConfig")
	}
}

func TestCheckConfigCleanExits0(t *testing.T) {
	writeConfig(t, "[layout]\nline1 = [\"model\", \"effort\"]\n")
	var out bytes.Buffer
	if code := checkConfig(&out); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), "warnings: none") {
		t.Errorf("expected no warnings:\n%s", out.String())
	}
}

func TestCheckConfigNoFileUsesDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // dir exists, file does not
	var out bytes.Buffer
	code := checkConfig(&out)
	s := out.String()
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(s, "not found — using built-in defaults") {
		t.Errorf("expected not-found status:\n%s", s)
	}
	if !strings.Contains(s, "limit_5h") {
		t.Errorf("expected the default layout to be shown:\n%s", s)
	}
}
