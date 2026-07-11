// claude-code-statusline: read one Claude Code JSON payload from stdin, print
// one ANSI-colored statusline to stdout, exit. No network, no credentials.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// version is stamped at build time via -ldflags "-X main.version=...".
// It stays "dev" for plain `go build` / `go run`.
var version = "dev"

const usage = `claude-code-statusline — a statusline for Claude Code.

Reads one Claude Code JSON payload on stdin and prints an ANSI-colored
statusline on stdout. Claude Code normally invokes it for you via the
statusLine setting, with no arguments.

Usage:
  claude-code-statusline            read a JSON payload from stdin, print the statusline
  claude-code-statusline --version  print the version and exit
  claude-code-statusline --help     show this help and exit

Config: $XDG_CONFIG_HOME/claude-code-statusline/config.toml
    (falls back to ~/.config/claude-code-statusline/config.toml)
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is the testable entry point: it parses args and renders, returning the
// process exit code. Any argument means a manual invocation — only the known
// flags are accepted, and an unknown one errors out instead of falling through
// to a stdin read that would block forever on a terminal.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "--version", "-v", "version":
			fmt.Fprintln(stdout, version)
			return 0
		case "--help", "-h", "help":
			fmt.Fprint(stdout, usage)
			return 0
		default:
			fmt.Fprintf(stderr, "claude-code-statusline: unknown option %q\n\n", args[0])
			fmt.Fprint(stderr, usage)
			return 2
		}
	}

	input, _ := io.ReadAll(stdin)
	if strings.TrimSpace(string(input)) == "" {
		fmt.Fprintln(stdout, "Claude")
		return 0
	}
	var payload Payload
	if err := json.Unmarshal(input, &payload); err != nil {
		payload = Payload{}
	}
	cfg := loadConfig()
	fmt.Fprintln(stdout, render(&payload, &cfg))
	return 0
}
