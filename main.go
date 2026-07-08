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

func main() {
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
