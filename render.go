// Segment dispatch, separators, line assembly.

package main

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// render draws each layout line by joining its non-empty segments with a dim
// separator, then joins lines with newlines, dropping lines that came up
// entirely empty.
func render(p *Payload, cfg *Config) string {
	sep := dim + cfg.Layout.Separator + reset
	var out []string
	for _, line := range cfg.Layout.Lines {
		var parts []string
		for _, name := range line {
			if s, ok := segment(name, p, cfg); ok {
				parts = append(parts, s)
			}
		}
		if len(parts) > 0 {
			out = append(out, strings.Join(parts, sep))
		}
	}
	return strings.Join(out, "\n")
}

func segment(name string, p *Payload, cfg *Config) (string, bool) {
	switch name {
	case "model":
		return modelSeg(p, cfg), true
	case "directory":
		return directorySeg(p, cfg)
	case "context":
		return contextSeg(p, cfg)
	case "effort":
		return effortSeg(p, cfg), true
	case "limit_5h":
		return limitSeg(&cfg.Limit5h, "5h", p.RateLimits.FiveHour, "15:04"), true
	case "limit_7d":
		return limitSeg(&cfg.Limit7d, "7d", p.RateLimits.SevenDay, "Mon Jan 2, 15:04"), true
	default:
		key := strings.TrimPrefix(name, "custom.")
		if cc, ok := cfg.Custom[key]; ok && key != name {
			return customSeg(cc.Command, cc.TimeoutMs)
		}
		return "", false // config normalization already warned
	}
}

func modelSeg(p *Payload, cfg *Config) string {
	name := "Claude"
	if p.Model.DisplayName != "" {
		name = cleanModelName(p.Model.DisplayName)
	}
	return paint(cfg.Model.Color, name)
}

// directorySeg renders "dir@branch [wt-name] (+adds -dels)", each git part
// absent on any failure.
func directorySeg(p *Payload, cfg *Config) (string, bool) {
	d := &cfg.Directory
	if p.Cwd == "" {
		return "", false
	}
	trimmed := strings.TrimRight(p.Cwd, "/")
	base := trimmed
	if i := strings.LastIndex(trimmed, "/"); i >= 0 {
		base = trimmed[i+1:]
	}
	if base == "" {
		return "", false
	}
	var b strings.Builder
	b.WriteString(paint(d.Color, base))
	if d.Git {
		info := collectGit(p.Cwd, d.Diff, d.Worktree)
		if info.Branch != "" {
			b.WriteString(dim + "@" + reset + paint(d.BranchColor, info.Branch))
		}
		if info.Worktree != "" {
			b.WriteString(" " + dim + "[" + reset + paint(d.WorktreeColor, info.Worktree) + dim + "]" + reset)
		}
		if info.Adds+info.Dels > 0 {
			b.WriteString(" " + dim + "(" + reset +
				paint(d.AddsColor, fmt.Sprintf("+%d", info.Adds)) + " " +
				paint(d.DelsColor, fmt.Sprintf("-%d", info.Dels)) +
				dim + ")" + reset)
		}
	}
	return b.String(), true
}

func contextSeg(p *Payload, cfg *Config) (string, bool) {
	c := &cfg.Context
	cw := &p.ContextWindow
	pct := cw.PctUsed()
	color := usageColor(c.Thresholds, float64(pct))
	var parts []string
	if b, ok := bar(c.Bar, float64(pct), c.Width, color); ok {
		parts = append(parts, b)
	}
	if c.Counts {
		parts = append(parts, paint(c.CountsColor, fmtTokens(cw.CurrentTotal())+"/"+fmtTokens(cw.Size())))
	}
	if c.Percent {
		parts = append(parts, dim+"("+reset+paint(color, strconv.Itoa(pct)+"%")+dim+")"+reset)
	}
	if len(parts) == 0 {
		return "", false
	}
	return strings.Join(parts, " "), true
}

// effortSeg uses only stdin effort.level — no settings.json/env fallbacks.
// Missing → medium.
func effortSeg(p *Payload, cfg *Config) string {
	e := &cfg.Effort
	level := p.Effort.Level
	if level == "" {
		level = "medium"
	}
	var value string
	switch level {
	case "low":
		value = dim + "low" + reset
	case "medium":
		value = paint(e.MediumColor, "med")
	case "high":
		value = paint(e.HighColor, "high")
	case "xhigh":
		value = paint(e.XhighColor, "xhigh")
	case "max":
		value = paint(e.MaxColor, "max")
	default:
		value = paint(e.HighColor, level)
	}
	return fg(e.LabelColor) + "effort:" + reset + " " + value
}

func limitSeg(lc *LimitCfg, label string, l *Limit, timeLayout string) string {
	if l == nil || l.UsedPercentage == nil {
		// No fabricated 0% — dim placeholder.
		return dim + label + " -" + reset
	}
	pct := *l.UsedPercentage
	color := usageColor(lc.Thresholds, pct)
	out := paint(lc.LabelColor, label)
	if b, ok := bar(lc.Bar, pct, lc.Width, color); ok {
		out += " " + b
	}
	out += " " + paint(color, strconv.FormatInt(int64(math.Round(pct)), 10)+"%")
	if lc.Reset && l.ResetsAt != nil {
		if epoch, ok := l.ResetsAt.ToEpoch(); ok {
			t := time.Unix(epoch, 0).Local()
			out += " " + dim + "@" + t.Format(timeLayout) + reset
		}
	}
	return out
}

// customSeg runs the user's command through `sh -c`; the process is killed
// and the segment skipped when it outlives its timeout. Output is the first
// line, rendered verbatim (the script may emit its own ANSI colors).
func customSeg(command string, timeoutMs uint64) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.WaitDelay = 100 * time.Millisecond
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	first, _, _ := strings.Cut(string(out), "\n")
	first = strings.TrimSpace(first)
	return first, first != ""
}
