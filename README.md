# upcli

`upcli` updates Codex CLI, Claude Code CLI, and Google's `agy` CLI concurrently, with live per-tool output and an at-a-glance result summary.

The interface takes cues from [Mole](https://github.com/tw93/Mole): a compact, colorful terminal dashboard with clear status markers and a persistent result summary. It uses Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss), the same lightweight Go terminal-UI building blocks used by Mole.

## Run

```sh
go run ./cmd/upcli
```

Or build the executable:

```sh
make build
./bin/upcli
```

Install the `upcli` command into `~/.local/bin`:

```sh
make install
```

If `~/.local/bin` is not on your `PATH`, add `export PATH="$HOME/.local/bin:$PATH"` to your shell profile. On this Mac, `upcli` is already linked there.

## Behavior

- Starts `codex update`, `claude update`, and `agy update` at the same time.
- Renders the live dashboard inline in the current shell and preserves the final report in scrollback.
- Captures each command's stdout and stderr separately, then streams both into that tool's card so parallel output stays attributable.
- Shows detected before/after versions and classifies each result as `UPDATED`, `UP_TO_DATE`, `RESTART_REQUIRED`, `FAILED`, or `CHECK`.
- Uses a non-TUI, tool-prefixed streaming mode when stdout is piped or the terminal does not support interactive rendering.
- Returns a failing exit code when an updater fails or its result cannot be verified.

The updater commands belong to their respective CLIs; this project orchestrates them and does not install the CLIs through a separate package manager.

## Development

```sh
go test ./...
go vet ./...
```

## Example

```text
  UPCLI  !  Review required
  1 updated   ·   1 already current   ·   1 need attention

  Codex CLI       ✓ UPDATED           codex-cli 0.162.1 → codex-cli 0.163.0
  Claude Code     ✓ UP TO DATE        2.1.299
  agy CLI         ✗ FAILED            1.3.5
  ↳ network timeout
```
