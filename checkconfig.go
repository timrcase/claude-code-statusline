// --check-config: load config exactly as a render would, but report the
// resolved path, any parse warnings (which otherwise vanish to stderr under
// Claude Code), the effective layout, and the declared field/custom sections.
// Exit 1 if the config produced warnings, so it doubles as a lint.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func checkConfig(stdout io.Writer) int {
	path, hasPath := configPath()

	var (
		status   string
		cfg      Config
		warnings bytes.Buffer
	)
	switch {
	case !hasPath:
		status = "no $XDG_CONFIG_HOME or $HOME set — using built-in defaults"
		cfg = DefaultConfig()
	default:
		raw, err := os.ReadFile(path)
		switch {
		case err == nil:
			status = path + " (loaded)"
			prev := warnOut
			warnOut = &warnings // capture warnings emitted during parse
			cfg = parseConfig(string(raw), path)
			warnOut = prev
		case os.IsNotExist(err):
			status = path + " (not found — using built-in defaults)"
			cfg = DefaultConfig()
		default:
			status = fmt.Sprintf("%s (unreadable: %v — using built-in defaults)", path, err)
			cfg = DefaultConfig()
		}
	}

	fmt.Fprintf(stdout, "config: %s\n\n", status)

	warnLines := nonEmptyLines(warnings.String())
	if len(warnLines) == 0 {
		fmt.Fprintln(stdout, "warnings: none")
	} else {
		fmt.Fprintf(stdout, "warnings (%d):\n", len(warnLines))
		for _, w := range warnLines {
			fmt.Fprintf(stdout, "  - %s\n", strings.TrimPrefix(w, "claude-code-statusline: "))
		}
	}

	fmt.Fprintf(stdout, "\nlayout (separator %q):\n", cfg.Layout.Separator)
	if len(cfg.Layout.Lines) == 0 {
		fmt.Fprintln(stdout, "  (no segments)")
	}
	for i, line := range cfg.Layout.Lines {
		if len(line) == 0 {
			continue
		}
		fmt.Fprintf(stdout, "  line%d: %s\n", i+1, strings.Join(line, ", "))
	}

	if names := sortedKeys(cfg.Fields); len(names) > 0 {
		fmt.Fprintf(stdout, "\nfield sections:  %s\n", strings.Join(names, ", "))
	}
	if names := sortedKeys(cfg.Custom); len(names) > 0 {
		fmt.Fprintf(stdout, "custom sections: %s\n", strings.Join(names, ", "))
	}

	if len(warnLines) > 0 {
		return 1
	}
	return 0
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
