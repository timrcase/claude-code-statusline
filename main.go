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

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Println(version)
			return
		}
	}
	input, _ := io.ReadAll(os.Stdin)
	if strings.TrimSpace(string(input)) == "" {
		fmt.Println("Claude")
		return
	}
	var payload Payload
	if err := json.Unmarshal(input, &payload); err != nil {
		payload = Payload{}
	}
	cfg := loadConfig()
	fmt.Println(render(&payload, &cfg))
}
