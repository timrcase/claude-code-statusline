package main

import (
	"encoding/json"
	"os"
	"testing"
)

func loadFixture(t *testing.T, name string) Payload {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return p
}

func TestParsesFullFixture(t *testing.T) {
	p := loadFixture(t, "payload.json")
	if p.Model.DisplayName != "Opus 4.8" {
		t.Errorf("display_name = %q", p.Model.DisplayName)
	}
	if p.Cwd != "/home/user/projects/claude-statusline" {
		t.Errorf("cwd = %q", p.Cwd)
	}
	if p.Effort.Level != "high" {
		t.Errorf("effort = %q", p.Effort.Level)
	}
	// 5910 input + 2019 cache_creation + 18559 cache_read
	if got := p.ContextWindow.CurrentTotal(); got != 26_488 {
		t.Errorf("current_total = %d", got)
	}
	if got := p.ContextWindow.Size(); got != 1_000_000 {
		t.Errorf("size = %d", got)
	}
	if got := p.ContextWindow.PctUsed(); got != 2 {
		t.Errorf("pct_used = %d", got)
	}
	fh := p.RateLimits.FiveHour
	if fh == nil || fh.UsedPercentage == nil || *fh.UsedPercentage != 39.0 {
		t.Fatalf("five_hour = %+v", fh)
	}
	if epoch, ok := fh.ResetsAt.ToEpoch(); !ok || epoch != 1_783_404_600 {
		t.Errorf("five_hour resets_at = %d, %v", epoch, ok)
	}
	sd := p.RateLimits.SevenDay
	if sd == nil || sd.UsedPercentage == nil || *sd.UsedPercentage != 3.0 {
		t.Fatalf("seven_day = %+v", sd)
	}
}

func TestEmptyObjectIsFine(t *testing.T) {
	var p Payload
	if err := json.Unmarshal([]byte("{}"), &p); err != nil {
		t.Fatal(err)
	}
	if got := p.ContextWindow.Size(); got != 200_000 {
		t.Errorf("size = %d", got)
	}
	if got := p.ContextWindow.PctUsed(); got != 0 {
		t.Errorf("pct_used = %d", got)
	}
	if p.RateLimits.FiveHour != nil {
		t.Error("five_hour should be absent")
	}
}

func TestISOResetsAtParses(t *testing.T) {
	var r ResetsAt
	if err := json.Unmarshal([]byte(`"2026-07-06T13:00:00Z"`), &r); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.ToEpoch(); !ok {
		t.Error("ISO resets_at should yield an epoch")
	}
}

func TestZeroEpochResetsAtIsAbsent(t *testing.T) {
	var r ResetsAt
	if err := json.Unmarshal([]byte(`0`), &r); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.ToEpoch(); ok {
		t.Error("epoch 0 should not yield a reset time")
	}
}

func TestModelNameCleanup(t *testing.T) {
	if got := cleanModelName("Sonnet 4.6 (1M context)"); got != "Sonnet 4.6 1M" {
		t.Errorf("got %q", got)
	}
	if got := cleanModelName("Opus 4.6"); got != "Opus 4.6" {
		t.Errorf("got %q", got)
	}
}
