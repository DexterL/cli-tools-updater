# upcli

**Update Codex CLI, Claude Code CLI, and agy in one command—with parallel progress you can actually follow.**

[![Latest release](https://img.shields.io/github/v/release/DexterL/cli-tools-updater)](https://github.com/DexterL/cli-tools-updater/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white)](go.mod)

`upcli` brings three separate CLI updates into one clear, concurrent run. Each tool gets its own live status and output, followed by a compact summary of what changed and what needs attention.

```text
  UPCLI  !  Review required
  1 updated   ·   1 already current   ·   1 need attention

  Codex CLI       ✓ UPDATED           codex-cli 0.162.1 → codex-cli 0.163.0
  Claude Code     ✓ UP TO DATE        2.1.299
  agy CLI         ✗ FAILED            1.3.5
  ↳ network timeout
```

## Why upcli

- **One command, three updates.** Runs `codex update`, `claude update`, and `agy update` concurrently.
- **Output stays readable.** A live, inline terminal dashboard keeps each tool's output with its status; piped output uses simple tool-prefixed logs.
- **Know what happened.** Shows detected before/after versions and highlights failures, restart requests, or results that need checking.
- **Fits existing installs.** Each CLI is updated by its own updater. `upcli` does not replace or bundle those tools.

## Install on macOS Intel

The current release provides a native **macOS Intel (x86_64)** package. Download the archive and checksum from [v0.1.0](https://github.com/DexterL/cli-tools-updater/releases/tag/v0.1.0), then verify and install:

```sh
VERSION=v0.1.0
ASSET="upcli_${VERSION}_darwin_amd64.tar.gz"
BASE="https://github.com/DexterL/cli-tools-updater/releases/download/${VERSION}"

curl -fLO "${BASE}/${ASSET}"
curl -fLO "${BASE}/${ASSET}.sha256"
shasum -a 256 -c "${ASSET}.sha256"
tar -xzf "${ASSET}"
cd "upcli_${VERSION}_darwin_amd64"
./install.sh
```

The installer places `upcli` in `~/.local/bin`. If needed, add that directory to your `PATH`:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Codex CLI, Claude Code CLI, and agy must already be installed and available on your `PATH`. The current release does not include an Apple Silicon package.

## Use

```sh
upcli --version
upcli
```

Running `upcli` starts all three update commands. A non-zero exit status indicates an update failure or a result that could not be verified.

## Build from source

Requires Go 1.24 or later.

```sh
git clone https://github.com/DexterL/cli-tools-updater.git
cd cli-tools-updater
go test ./...
go vet ./...
make install
```

To build the current Intel release archive locally:

```sh
make release-darwin-amd64 VERSION=v0.1.0
```

The archive and its SHA-256 file are written to `dist/`.

## License

[MIT](LICENSE)
