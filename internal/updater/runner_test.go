package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name      string
		before    string
		after     string
		output    string
		updateErr error
		want      Status
	}{
		{name: "updated", before: "v1", after: "v2", want: Updated},
		{name: "latest", before: "v1", after: "v1", output: "Already up to date", want: UpToDate},
		{name: "restart", before: "v1", after: "v1", output: "Please restart agy", want: RestartRequired},
		{name: "failed", before: "v1", after: "v1", updateErr: errors.New("exit 1"), want: Failed},
		{name: "ambiguous", before: "v1", after: "v1", output: "done", want: Check},
		{name: "unverified", before: "unknown", after: "v1", want: Check},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := classify(test.before, test.after, nil, nil, test.updateErr, test.output)
			if got != test.want {
				t.Fatalf("classify() = %s, want %s", got, test.want)
			}
		})
	}
}

func TestRunStreamsOutputAndVerifiesVersionChange(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(dir, "updated")
	toolPath := filepath.Join(dir, "fake-codex")
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then
  if [ -f "` + state + `" ]; then echo 'codex-cli 2.0'; else echo 'codex-cli 1.0'; fi
else
  echo 'download started'
  echo 'install complete' >&2
  touch "` + state + `"
fi
`
	if err := os.WriteFile(toolPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	var events []Event
	Run(context.Background(), Tool{Name: "codex", Label: "Codex CLI", Bin: toolPath}, func(event Event) {
		events = append(events, event)
	})

	var result Event
	var logLines int
	for _, event := range events {
		if event.Kind == LogLine {
			logLines++
		}
		if event.Kind == Finished {
			result = event
		}
	}
	if result.Status != Updated || result.After != "codex-cli 2.0" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if logLines != 2 {
		t.Fatalf("received %d log lines, want stdout and stderr (2)", logLines)
	}
}
