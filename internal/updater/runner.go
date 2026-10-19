package updater

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

type Tool struct {
	Name  string
	Label string
	Bin   string
}

type Status string

const (
	Queued          Status = "QUEUED"
	Checking        Status = "CHECKING"
	Updating        Status = "UPDATING"
	Updated         Status = "UPDATED"
	UpToDate        Status = "UP_TO_DATE"
	RestartRequired Status = "RESTART_REQUIRED"
	Failed          Status = "FAILED"
	Check           Status = "CHECK"
	Cancelled       Status = "CANCELLED"
)

func (s Status) Done() bool {
	return s == Updated || s == UpToDate || s == RestartRequired || s == Failed || s == Check || s == Cancelled
}

type EventKind uint8

const (
	StatusChange EventKind = iota
	VersionFound
	LogLine
	Finished
)

type Event struct {
	Kind   EventKind
	Name   string
	Status Status
	Text   string
	After  string
	Error  string
}

type State struct {
	Tool   Tool
	Status Status
	Before string
	After  string
	Logs   []string
	Error  string
}

func DefaultTools() []Tool {
	return []Tool{
		{Name: "codex", Label: "Codex CLI", Bin: "codex"},
		{Name: "claude", Label: "Claude Code", Bin: "claude"},
		{Name: "agy", Label: "agy CLI", Bin: "agy"},
	}
}

var (
	restartPattern = regexp.MustCompile(`(?i)restart (codex|claude|agy)?|please restart|restart to use`)
	latestPattern  = regexp.MustCompile(`(?i)up to date|already (on the )?latest|no update available|already up-to-date`)
)

// Run executes the tool's updater and streams output as events. Callers may
// run multiple tools concurrently; each invocation owns its child process.
func Run(ctx context.Context, tool Tool, emit func(Event)) {
	send := func(event Event) {
		event.Name = tool.Name
		emit(event)
	}
	send(Event{Kind: StatusChange, Status: Checking})
	path, err := exec.LookPath(tool.Bin)
	if err != nil {
		send(Event{Kind: Finished, Status: Failed, Error: fmt.Sprintf("%s not found in PATH", tool.Bin)})
		return
	}

	before, versionErr := readVersion(ctx, path)
	if versionErr != nil {
		before = "unknown"
	}
	send(Event{Kind: VersionFound, Text: before})
	send(Event{Kind: StatusChange, Status: Updating})

	var output strings.Builder
	var outputMu sync.Mutex
	updateErr := runCommand(ctx, path, []string{"update"}, func(line string) {
		outputMu.Lock()
		output.WriteString(line)
		output.WriteByte('\n')
		outputMu.Unlock()
		send(Event{Kind: LogLine, Text: line})
	})

	after, afterErr := readVersion(ctx, path)
	status := classify(before, after, versionErr, afterErr, updateErr, output.String())
	if errors.Is(updateErr, context.Canceled) {
		status = Cancelled
	}
	errorText := ""
	if updateErr != nil {
		errorText = updateErr.Error()
	} else if afterErr != nil {
		errorText = "could not verify installed version: " + afterErr.Error()
	} else if status == Check {
		errorText = "updater output did not confirm the result"
	}
	send(Event{Kind: Finished, Status: status, After: after, Error: errorText})
}

func readVersion(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, path, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	lines := strings.Fields(strings.TrimSpace(string(output)))
	if len(lines) == 0 {
		return "", errors.New("empty version output")
	}
	return strings.Join(lines, " "), nil
}

func runCommand(ctx context.Context, path string, args []string, onLine func(string)) error {
	cmd := exec.CommandContext(ctx, path, args...)
	writer := &lineWriter{onLine: onLine}
	cmd.Stdout = writer
	cmd.Stderr = writer
	err := cmd.Run()
	writer.Flush()
	return err
}

type lineWriter struct {
	mu      sync.Mutex
	pending string
	onLine  func(string)
}

func (w *lineWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	w.pending += string(data)
	parts := strings.Split(w.pending, "\n")
	w.pending = parts[len(parts)-1]
	w.mu.Unlock()
	for _, line := range parts[:len(parts)-1] {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line != "" {
			w.onLine(line)
		}
	}
	return len(data), nil
}

func (w *lineWriter) Flush() {
	w.mu.Lock()
	line := strings.TrimSpace(w.pending)
	w.pending = ""
	w.mu.Unlock()
	if line != "" {
		w.onLine(line)
	}
}

func classify(before, after string, beforeErr, afterErr, updateErr error, output string) Status {
	if updateErr != nil {
		return Failed
	}
	if beforeErr != nil || afterErr != nil || before == "unknown" || after == "" {
		return Check
	}
	if before != after {
		return Updated
	}
	if restartPattern.MatchString(output) {
		return RestartRequired
	}
	if latestPattern.MatchString(output) {
		return UpToDate
	}
	return Check
}
