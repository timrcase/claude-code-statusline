package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCapturePayloadWritesAtomically(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	raw := []byte(`{"model":{"display_name":"X"}}`)
	capturePayload(raw)

	got, err := os.ReadFile(filepath.Join(dir, "claude-code-statusline", "last-payload.json"))
	if err != nil {
		t.Fatalf("captured file missing: %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Errorf("captured %q, want %q", got, raw)
	}
	// No leftover temp files from the atomic write.
	entries, _ := os.ReadDir(filepath.Join(dir, "claude-code-statusline"))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("leftover temp file %s", e.Name())
		}
	}
}

func TestPrintPayloadFromCapturedFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	capturePayload([]byte(`{"a":{"b":1}}`))

	var out, errb bytes.Buffer
	// piped=false -> read the captured file.
	if code := printPayload(strings.NewReader(""), false, &out, &errb); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "\"a\": {") {
		t.Errorf("expected pretty JSON, got %q", out.String())
	}
	if !strings.Contains(errb.String(), "captured") {
		t.Errorf("expected capture note on stderr, got %q", errb.String())
	}
}

func TestPrintPayloadNoCapture(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var out, errb bytes.Buffer
	if code := printPayload(strings.NewReader(""), false, &out, &errb); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if out.Len() != 0 || !strings.Contains(errb.String(), "no captured payload") {
		t.Errorf("out=%q err=%q", out.String(), errb.String())
	}
}

func TestPrintPayloadInvalidJSONPrintsRaw(t *testing.T) {
	var out, errb bytes.Buffer
	if code := printPayload(strings.NewReader("not json"), true, &out, &errb); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(out.String()) != "not json" {
		t.Errorf("raw passthrough failed: %q", out.String())
	}
}
