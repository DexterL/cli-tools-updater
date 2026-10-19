package main

import (
	"strings"
	"testing"

	"github.com/dexterl/cli-tools-updater/internal/updater"
)

func TestSummaryHighlightsVersionsAndAction(t *testing.T) {
	states := []updater.State{
		{
			Tool:   updater.Tool{Label: "Codex CLI"},
			Status: updater.Updated,
			Before: "codex-cli 1.0",
			After:  "codex-cli 2.0",
		},
		{
			Tool:   updater.Tool{Label: "Claude Code"},
			Status: updater.UpToDate,
			Before: "2.1.0",
			After:  "2.1.0",
		},
		{
			Tool:   updater.Tool{Label: "agy CLI"},
			Status: updater.Failed,
			Before: "1.3.0",
			Error:  "network timeout",
		},
	}

	got := summaryView(states)
	for _, want := range []string{"Review required", "1 updated", "1 already current", "Codex CLI", "UPDATED", "codex-cli 1.0", "codex-cli 2.0", "UP TO DATE", "FAILED", "network timeout"} {
		if !strings.Contains(got, want) {
			t.Errorf("summary missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "RESULT|") {
		t.Errorf("human summary should not contain machine-oriented RESULT lines:\n%s", got)
	}
}
