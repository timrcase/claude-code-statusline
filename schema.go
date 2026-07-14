// Stdin JSON contract from Claude Code.
//
// Every field is optional / defaulted. Claude Code's statusline payload has
// evolved across CLI versions (rate_limits is newer, effort newer still), so
// unknown fields are ignored and missing fields render as blank segments
// instead of failing the whole line.

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Payload struct {
	Model         Model         `json:"model"`
	Cwd           string        `json:"cwd"`
	Effort        Effort        `json:"effort"`
	ContextWindow ContextWindow `json:"context_window"`
	RateLimits    RateLimits    `json:"rate_limits"`

	// raw is the same stdin JSON decoded generically, so [field.*] segments can
	// resolve arbitrary dotted paths (cost.*, pr.*, …) the typed view omits.
	// Populated in run(); nil in unit tests that don't exercise field segments.
	raw map[string]any
}

type Model struct {
	DisplayName string `json:"display_name"`
}

type Effort struct {
	Level string `json:"level"`
}

type ContextWindow struct {
	ContextWindowSize uint64       `json:"context_window_size"`
	CurrentUsage      CurrentUsage `json:"current_usage"`
}

type CurrentUsage struct {
	InputTokens              uint64 `json:"input_tokens"`
	CacheCreationInputTokens uint64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     uint64 `json:"cache_read_input_tokens"`
}

// CurrentTotal is the tokens currently occupying the context window.
func (c *ContextWindow) CurrentTotal() uint64 {
	return c.CurrentUsage.InputTokens +
		c.CurrentUsage.CacheCreationInputTokens +
		c.CurrentUsage.CacheReadInputTokens
}

// Size is the window size, defaulting to 200k like the original script.
func (c *ContextWindow) Size() uint64 {
	if c.ContextWindowSize == 0 {
		return 200_000
	}
	return c.ContextWindowSize
}

func (c *ContextWindow) PctUsed() int {
	return int(min(c.CurrentTotal()*100/c.Size(), 100))
}

type RateLimits struct {
	FiveHour *Limit `json:"five_hour"`
	SevenDay *Limit `json:"seven_day"`
}

type Limit struct {
	UsedPercentage *float64  `json:"used_percentage"`
	ResetsAt       *ResetsAt `json:"resets_at"`
}

// ResetsAt arrives as a Unix epoch integer in current CLI builds, but the
// API-shaped variant of this data uses ISO 8601 strings. Accept both.
type ResetsAt struct {
	epoch   int64
	iso     string
	isEpoch bool
}

func (r *ResetsAt) UnmarshalJSON(b []byte) error {
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		r.epoch, r.isEpoch = n, true
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		r.iso = s
		return nil
	}
	return fmt.Errorf("resets_at: want epoch integer or ISO-8601 string, got %s", b)
}

func (r *ResetsAt) ToEpoch() (int64, bool) {
	if r.isEpoch {
		return r.epoch, r.epoch > 0
	}
	t, err := time.Parse(time.RFC3339, r.iso)
	if err != nil {
		return 0, false
	}
	return t.Unix(), true
}

// cleanModelName normalizes model display names:
// "Opus 4.6 (1M context)" -> "Opus 4.6 1M".
func cleanModelName(name string) string {
	open := strings.Index(name, " (")
	closing := strings.Index(name, " context)")
	if open >= 0 && closing > open {
		inner := name[open+2 : closing]
		if len(inner) > 0 && len(inner) <= 8 {
			return name[:open] + " " + inner
		}
	}
	return name
}
