// Git info via shelling out (git -C <cwd>). All errors are swallowed — parts
// just come back absent and the segment renders without them.

package main

import (
	"os/exec"
	"strconv"
	"strings"
)

type GitInfo struct {
	Branch   string
	Adds     uint64
	Dels     uint64
	Worktree string
}

// collectGit gates the extra git invocations on wantDiff / wantWorktree so
// disabled options cost nothing (numstat can be slow in huge repos).
func collectGit(cwd string, wantDiff, wantWorktree bool) GitInfo {
	var info GitInfo
	branch, ok := runGit(cwd, "rev-parse", "--abbrev-ref", "HEAD")
	if !ok {
		return info
	}
	info.Branch = branch
	if wantDiff {
		if numstat, ok := runGit(cwd, "diff", "--numstat"); ok {
			info.Adds, info.Dels = parseNumstat(numstat)
		}
	}
	if wantWorktree {
		if gitDir, ok := runGit(cwd, "rev-parse", "--git-dir"); ok {
			info.Worktree = worktreeName(gitDir)
		}
	}
	return info
}

func runGit(cwd string, args ...string) (string, bool) {
	cmd := exec.Command("git", append([]string{"-C", cwd}, args...)...)
	out, err := cmd.Output() // stdin and stderr both discarded
	if err != nil {
		return "", false
	}
	s := strings.TrimSpace(string(out))
	return s, s != ""
}

// parseNumstat sums the added/deleted columns of git diff --numstat. Binary
// files report "-\t-\tpath"; those columns fail to parse and count as 0.
func parseNumstat(numstat string) (adds, dels uint64) {
	for _, line := range strings.Split(numstat, "\n") {
		cols := strings.Fields(line)
		if len(cols) >= 2 {
			a, _ := strconv.ParseUint(cols[0], 10, 64)
			d, _ := strconv.ParseUint(cols[1], 10, 64)
			adds += a
			dels += d
		}
	}
	return adds, dels
}

// worktreeName extracts the linked-worktree name from a git-dir like
// "<repo>/.git/worktrees/<name>": the component right after "/worktrees/".
func worktreeName(gitDir string) string {
	const marker = "/worktrees/"
	idx := strings.Index(gitDir, marker)
	if idx < 0 {
		return ""
	}
	name, _, _ := strings.Cut(gitDir[idx+len(marker):], "/")
	return name
}
