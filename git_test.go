package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNumstatSumsAndSkipsBinary(t *testing.T) {
	s := "12\t4\tsrc/main.go\n3\t0\tREADME.md\n-\t-\tlogo.png\n"
	adds, dels := parseNumstat(s)
	if adds != 15 || dels != 4 {
		t.Errorf("got (%d, %d), want (15, 4)", adds, dels)
	}
	adds, dels = parseNumstat("")
	if adds != 0 || dels != 0 {
		t.Errorf("empty numstat: got (%d, %d)", adds, dels)
	}
}

func TestWorktreeNameExtraction(t *testing.T) {
	cases := []struct {
		gitDir string
		want   string
	}{
		{"/repo/.git/worktrees/feature-x", "feature-x"},
		{"/Users/t/.git/worktrees/wt-1/", "wt-1"},
		{"/repo/.git", ""},
		{".git", ""},
	}
	for _, c := range cases {
		if got := worktreeName(c.gitDir); got != c.want {
			t.Errorf("worktreeName(%q) = %q, want %q", c.gitDir, got, c.want)
		}
	}
}

func TestNonexistentDirYieldsNothing(t *testing.T) {
	info := collectGit("/nonexistent-path-for-sure", true, true)
	if info.Branch != "" || info.Adds != 0 || info.Dels != 0 || info.Worktree != "" {
		t.Errorf("expected empty GitInfo, got %+v", info)
	}
}

// End-to-end against a scratch repo: branch, dirty diff, and a linked
// worktree. Exercises the real git binary.
func TestRealRepoBranchDiffWorktree(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	wt := filepath.Join(base, "wt")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	git(repo, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(repo, "add", ".")
	git(repo, "commit", "-q", "-m", "init")

	// Dirty the tree: +1 line (three), -1 line (two).
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("one\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	info := collectGit(repo, true, true)
	if info.Branch != "main" {
		t.Errorf("branch = %q", info.Branch)
	}
	if info.Adds != 1 || info.Dels != 1 {
		t.Errorf("diff = (+%d -%d), want (+1 -1)", info.Adds, info.Dels)
	}
	if info.Worktree != "" {
		t.Errorf("main checkout is not a worktree, got %q", info.Worktree)
	}

	// Linked worktree detection.
	git(repo, "worktree", "add", "-q", wt, "-b", "side")
	wtInfo := collectGit(wt, false, true)
	if wtInfo.Branch != "side" {
		t.Errorf("worktree branch = %q", wtInfo.Branch)
	}
	if wtInfo.Worktree != "wt" {
		t.Errorf("worktree name = %q, want wt", wtInfo.Worktree)
	}
}
