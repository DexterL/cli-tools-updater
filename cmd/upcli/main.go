package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dexterl/cli-tools-updater/internal/updater"
)

var (
	version = "dev"

	cyan   = lipgloss.Color("86")
	green  = lipgloss.Color("42")
	yellow = lipgloss.Color("214")
	red    = lipgloss.Color("196")
	muted  = lipgloss.Color("241")
	border = lipgloss.Color("238")

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(cyan)
	mutedStyle = lipgloss.NewStyle().Foreground(muted)
	cardStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(0, 1)
	summaryOut = lipgloss.NewRenderer(os.Stdout)
)

type eventMsg updater.Event
type allDoneMsg struct{}
type spinnerMsg struct{}

type model struct {
	states   []updater.State
	events   <-chan updater.Event
	cancel   context.CancelFunc
	spin     int
	width    int
	finished bool
}

var spinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "--help" || os.Args[1] == "-h" {
			fmt.Println("upcli — update Codex CLI, Claude Code CLI, and agy concurrently")
			fmt.Println("Usage: upcli [--version | --help]")
			return
		}
		if os.Args[1] == "--version" || os.Args[1] == "-v" {
			fmt.Printf("upcli %s\n", version)
			return
		}
		fmt.Fprintf(os.Stderr, "unknown argument: %s\nUsage: upcli [--version | --help]\n", os.Args[1])
		os.Exit(2)
	}

	tools := updater.DefaultTools()
	if !isTerminal(os.Stdout) {
		runPlain(tools)
		return
	}
	runTUI(tools)
}

func runTUI(tools []updater.Tool) {
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithCancel(signalCtx)
	defer cancel()

	events := make(chan updater.Event, 256)
	initial := make([]updater.State, len(tools))
	for i, tool := range tools {
		initial[i] = updater.State{Tool: tool, Status: updater.Queued}
	}
	// Keep Bubble Tea in inline mode so updates stay in the invoking shell's
	// scrollback instead of switching to a separate alternate screen.
	p := tea.NewProgram(model{states: initial, events: events, cancel: cancel, width: 80}, tea.WithContext(signalCtx))

	go func() {
		var wg sync.WaitGroup
		for _, tool := range tools {
			wg.Add(1)
			go func(tool updater.Tool) {
				defer wg.Done()
				updater.Run(ctx, tool, func(event updater.Event) {
					select {
					case events <- event:
					case <-ctx.Done():
					}
				})
			}(tool)
		}
		wg.Wait()
		close(events)
	}()

	final, err := p.Run()
	cancel()
	result, ok := final.(model)
	if ok {
		for event := range events {
			for i := range result.states {
				if result.states[i].Tool.Name == event.Name {
					applyEvent(&result.states[i], event)
					break
				}
			}
		}
		for i := range result.states {
			if !result.states[i].Status.Done() {
				result.states[i].Status = updater.Cancelled
				result.states[i].Error = "cancelled by user"
			}
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nUI error: %v\n", err)
		os.Exit(1)
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "could not read update results")
		os.Exit(1)
	}
	printSummary(result.states)
}

func runPlain(tools []updater.Tool) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	events := make(chan updater.Event, 256)
	var wg sync.WaitGroup
	for _, tool := range tools {
		wg.Add(1)
		go func(tool updater.Tool) {
			defer wg.Done()
			updater.Run(ctx, tool, func(event updater.Event) { events <- event })
		}(tool)
	}
	go func() { wg.Wait(); close(events) }()

	states := make(map[string]updater.State, len(tools))
	for _, tool := range tools {
		states[tool.Name] = updater.State{Tool: tool, Status: updater.Queued}
	}
	fmt.Println("UPCLI  •  Concurrent CLI updates")
	fmt.Println("────────────────────────────────────────────────────────────")
	for event := range events {
		state := states[event.Name]
		applyEvent(&state, event)
		states[event.Name] = state
		if event.Kind == updater.LogLine {
			fmt.Printf("[%s] %s\n", event.Name, event.Text)
		}
	}
	ordered := make([]updater.State, 0, len(tools))
	for _, tool := range tools {
		ordered = append(ordered, states[tool.Name])
	}
	printSummary(ordered)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(waitEvent(m.events), tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg { return spinnerMsg{} }))
}

func waitEvent(events <-chan updater.Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return allDoneMsg{}
		}
		return eventMsg(event)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.finished = false
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
	case spinnerMsg:
		m.spin = (m.spin + 1) % len(spinner)
		return m, tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg { return spinnerMsg{} })
	case eventMsg:
		event := updater.Event(msg)
		for i := range m.states {
			if m.states[i].Tool.Name == event.Name {
				applyEvent(&m.states[i], event)
				break
			}
		}
		return m, waitEvent(m.events)
	case allDoneMsg:
		m.finished = true
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	w := m.width - 4
	if w < 44 {
		w = 44
	}
	var b strings.Builder
	b.WriteString("\n  ")
	b.WriteString(titleStyle.Render("UPCLI"))
	b.WriteString(mutedStyle.Render("  /  concurrent CLI updates"))
	b.WriteString("\n  ")
	b.WriteString(mutedStyle.Render("Codex  ·  Claude Code  ·  agy    |    Ctrl+C to cancel"))
	b.WriteString("\n\n")

	completed := 0
	for _, state := range m.states {
		if state.Status.Done() {
			completed++
		}
		b.WriteString(renderCard(state, w, m.spin))
		b.WriteString("\n")
	}
	footer := fmt.Sprintf("  %d/%d complete", completed, len(m.states))
	if m.finished {
		footer += "  ·  preparing summary"
	}
	b.WriteString(mutedStyle.Render(footer))
	return b.String()
}

func renderCard(state updater.State, width, spin int) string {
	name := lipgloss.NewStyle().Bold(true).Render(state.Tool.Label)
	status := statusStyle(state.Status).Render(statusText(state.Status, spin))
	line := fmt.Sprintf("%s%s", name, strings.Repeat(" ", max(1, width-lipgloss.Width(name)-lipgloss.Width(status)-4))+status)
	version := "Version: " + fallback(state.Before, "checking…")
	if state.After != "" && state.After != state.Before {
		version += "  →  " + state.After
	}
	versionLine := mutedStyle.Render(version)
	logLine := mutedStyle.Render("Waiting for updater…")
	if len(state.Logs) > 0 {
		logLine = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(clip(state.Logs[len(state.Logs)-1], width-6))
	}
	return cardStyle.Width(width).Render(line + "\n" + versionLine + "\n" + logLine)
}

func statusText(status updater.Status, spin int) string {
	switch status {
	case updater.Queued:
		return "QUEUED"
	case updater.Checking, updater.Updating:
		return spinner[spin%len(spinner)] + " " + strings.ToUpper(string(status))
	case updater.Updated:
		return "✓ UPDATED"
	case updater.UpToDate:
		return "✓ UP TO DATE"
	case updater.RestartRequired:
		return "↻ RESTART"
	case updater.Cancelled:
		return "⏹ CANCELLED"
	case updater.Failed:
		return "✗ FAILED"
	default:
		return "! CHECK"
	}
}

func statusStyle(status updater.Status) lipgloss.Style {
	color := muted
	switch status {
	case updater.Updated, updater.UpToDate:
		color = green
	case updater.Checking, updater.Updating, updater.RestartRequired:
		color = yellow
	case updater.Cancelled:
		color = yellow
	case updater.Failed, updater.Check:
		color = red
	}
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

func applyEvent(state *updater.State, event updater.Event) {
	switch event.Kind {
	case updater.StatusChange:
		state.Status = event.Status
	case updater.VersionFound:
		state.Before = event.Text
	case updater.LogLine:
		state.Logs = append(state.Logs, event.Text)
		if len(state.Logs) > 8 {
			state.Logs = state.Logs[len(state.Logs)-8:]
		}
	case updater.Finished:
		state.Status = event.Status
		state.After = event.After
		state.Error = event.Error
	}
}

func printSummary(states []updater.State) {
	fmt.Print(summaryView(states))
	failed := false
	cancelled := false
	for _, state := range states {
		if state.Status == updater.Failed || state.Status == updater.Check {
			failed = true
		}
		if state.Status == updater.Cancelled {
			cancelled = true
		}
	}
	if failed {
		os.Exit(1)
	}
	if cancelled {
		os.Exit(130)
	}
}

func summaryView(states []updater.State) string {
	updated, current, attention := 0, 0, 0
	for _, state := range states {
		switch state.Status {
		case updater.Updated:
			updated++
		case updater.UpToDate:
			current++
		case updater.RestartRequired, updater.Failed, updater.Check, updater.Cancelled:
			attention++
		}
	}

	title := "✓  All updates complete"
	titleStyle := summaryOut.NewStyle().Bold(true).Foreground(green)
	if attention > 0 {
		title = "!  Review required"
		titleStyle = summaryOut.NewStyle().Bold(true).Foreground(yellow)
	}
	counts := fmt.Sprintf("%d updated   ·   %d already current", updated, current)
	if attention > 0 {
		counts += fmt.Sprintf("   ·   %d need attention", attention)
	}

	var b strings.Builder
	b.WriteString("\n  ")
	b.WriteString(summaryOut.NewStyle().Bold(true).Foreground(cyan).Render("UPCLI"))
	b.WriteString("  ")
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n  ")
	b.WriteString(summaryOut.NewStyle().Foreground(muted).Render(counts))
	b.WriteString("\n\n")

	for _, state := range states {
		label := summaryOut.NewStyle().Bold(true).Width(16).Render(state.Tool.Label)
		status := summaryStatus(state.Status)
		versions := summaryVersions(state)
		b.WriteString("  ")
		b.WriteString(label)
		b.WriteString(status)
		b.WriteString("  ")
		b.WriteString(versions)
		b.WriteByte('\n')

		if state.Error != "" {
			b.WriteString("  ")
			b.WriteString(summaryOut.NewStyle().Foreground(red).Render("↳ " + state.Error))
			b.WriteByte('\n')
		} else if state.Status == updater.RestartRequired {
			b.WriteString("  ")
			b.WriteString(summaryOut.NewStyle().Foreground(yellow).Render("↳ Restart this CLI to finish applying the update."))
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')
	return b.String()
}

func summaryStatus(status updater.Status) string {
	text := statusText(status, 0)
	style := summaryOut.NewStyle().Bold(true).Width(18)
	switch status {
	case updater.Updated, updater.UpToDate:
		style = style.Foreground(green)
	case updater.RestartRequired, updater.Cancelled:
		style = style.Foreground(yellow)
	case updater.Failed, updater.Check:
		style = style.Foreground(red)
	default:
		style = style.Foreground(muted)
	}
	return style.Render(text)
}

func summaryVersions(state updater.State) string {
	before := fallback(state.Before, "unknown")
	if state.After == "" || state.After == state.Before {
		return summaryOut.NewStyle().Foreground(cyan).Render(before)
	}
	oldVersion := summaryOut.NewStyle().Foreground(muted).Render(before)
	newVersion := summaryOut.NewStyle().Bold(true).Foreground(green).Render(state.After)
	return oldVersion + summaryOut.NewStyle().Foreground(muted).Render("  →  ") + newVersion
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0 && os.Getenv("TERM") != "dumb"
}

func fallback(value, alt string) string {
	if value == "" {
		return alt
	}
	return value
}

func clip(value string, width int) string {
	if width < 4 {
		return value
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(value)
}
