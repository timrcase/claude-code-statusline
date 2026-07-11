package main

import (
	"bytes"
	"strings"
	"testing"
)

// failReader fails the test if anything tries to read from it. The arg-handling
// paths (version/help/unknown) must return before ever touching stdin — a
// regression there previously left `--versioon` blocking on a terminal read.
type failReader struct{ t *testing.T }

func (r failReader) Read([]byte) (int, error) {
	r.t.Fatal("stdin was read; arg handling should have returned first")
	return 0, nil
}

func TestRunVersion(t *testing.T) {
	version = "v1.2.3"
	for _, arg := range []string{"--version", "-v", "version"} {
		var out, errb bytes.Buffer
		if code := run([]string{arg}, failReader{t}, &out, &errb); code != 0 {
			t.Fatalf("%s: exit %d, want 0", arg, code)
		}
		if got := strings.TrimSpace(out.String()); got != "v1.2.3" {
			t.Fatalf("%s: stdout %q, want v1.2.3", arg, got)
		}
	}
}

func TestRunHelp(t *testing.T) {
	for _, arg := range []string{"--help", "-h", "help"} {
		var out, errb bytes.Buffer
		if code := run([]string{arg}, failReader{t}, &out, &errb); code != 0 {
			t.Fatalf("%s: exit %d, want 0", arg, code)
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Fatalf("%s: help output missing usage", arg)
		}
	}
}

func TestRunUnknownOption(t *testing.T) {
	var out, errb bytes.Buffer
	// failReader asserts stdin is never read on this path.
	code := run([]string{"--versioon"}, failReader{t}, &out, &errb)
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout should be empty, got %q", out.String())
	}
	if !strings.Contains(errb.String(), "unknown option") || !strings.Contains(errb.String(), "--versioon") {
		t.Fatalf("stderr missing error for the bad option: %q", errb.String())
	}
	if !strings.Contains(errb.String(), "Usage:") {
		t.Fatalf("stderr should include usage: %q", errb.String())
	}
}

func TestRunEmptyStdin(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(nil, strings.NewReader("  \n"), &out, &errb); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if got := strings.TrimSpace(out.String()); got != "Claude" {
		t.Fatalf("stdout %q, want Claude", got)
	}
}
