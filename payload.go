// Payload capture + --print-payload. The statusline only receives its JSON on a
// Claude Code refresh (stdin), so we stash the last one on disk; --print-payload
// then shows it, letting users discover exact paths for [field.*] segments.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// statePath is where the last-seen payload is cached — env-only, mirroring
// configPath (config.go).
func statePath() (string, bool) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return "", false
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "claude-code-statusline", "last-payload.json"), true
}

// capturePayload writes the raw stdin bytes to the state file atomically (temp
// file + rename), so a concurrent --print-payload never reads a torn file.
// Best-effort: every error is swallowed so capture can't break rendering.
func capturePayload(raw []byte) {
	path, ok := statePath()
	if !ok {
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, "payload-*.tmp")
	if err != nil {
		return
	}
	name := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		os.Remove(name)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
	}
}

// printPayload pretty-prints a payload for troubleshooting: piped stdin when
// present, otherwise the last captured file (noting its capture time on stderr
// so stdout stays valid JSON for piping into jq).
func printPayload(stdin io.Reader, piped bool, stdout, stderr io.Writer) int {
	if piped {
		raw, _ := io.ReadAll(stdin)
		return prettyPrint(raw, stdout)
	}
	path, ok := statePath()
	if !ok {
		fmt.Fprintln(stderr, "claude-code-statusline: no $XDG_STATE_HOME or $HOME to read a captured payload from")
		return 1
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "claude-code-statusline: no captured payload yet at %s\n", path)
		fmt.Fprintln(stderr, "Trigger a Claude Code statusline refresh first, or pipe JSON: claude-code-statusline --print-payload < payload.json")
		return 1
	}
	if fi, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, "# captured %s from %s\n", fi.ModTime().Local().Format("2006-01-02 15:04:05"), path)
	}
	return prettyPrint(raw, stdout)
}

// prettyPrint indents valid JSON; falls back to the raw bytes otherwise so a
// malformed payload is still inspectable.
func prettyPrint(raw []byte, stdout io.Writer) int {
	var buf bytes.Buffer
	if json.Indent(&buf, raw, "", "  ") == nil {
		buf.WriteByte('\n')
		stdout.Write(buf.Bytes())
		return 0
	}
	stdout.Write(raw)
	if n := len(raw); n == 0 || raw[n-1] != '\n' {
		io.WriteString(stdout, "\n")
	}
	return 0
}
