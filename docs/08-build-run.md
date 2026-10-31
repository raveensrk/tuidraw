# Build and run

The toolchain, dependencies, commands and configuration.

## Requirements

| Need | Detail |
|---|---|
| Go | 1.27.2 or newer, to build |
| Terminal | With mouse support, such as iTerm2, Terminal.app, kitty, WezTerm or Ghostty |

## Dependencies

Module path: `github.com/raveensrk/tuidraw`.

| Direct module | Version | Purpose |
|---|---|---|
| `charm.land/bubbletea/v2` | v2.1.0 | Terminal event loop and view |
| `charm.land/lipgloss/v2` | v2.0.6 | Styles and width measurement |
| `github.com/charmbracelet/x/ansi` | v0.11.9 | Truncating styled strings; stripping them in tests |

`go.mod` and `go.sum` pin the indirect modules these pull in; a rebuild lets `go mod tidy` choose them.

## Commands

All of these were run for this book unless marked.

| Task | Command |
|---|---|
| Test | `timeout 120 go test ./...` |
| Build | `go build -o tuidraw ./cmd/tuidraw` |
| Run | `go run ./cmd/tuidraw drawing.tuidraw` |
| Install from a clone, testing first | `./install.sh` |
| Install from a clone, no tests | `./install.sh --skip-tests` |
| Install from the module (unverified) | `go install github.com/raveensrk/tuidraw/cmd/tuidraw@latest` |

The repository's `.gitignore` covers the built `/tuidraw` binary, `*.tmp` from atomic saves, and `.DS_Store`.

### install.sh

| Argument | Effect | Exit |
|---|---|---|
| `-h`, `--help` | Short help | 0 |
| `help` | Long help | 0 |
| `--skip-tests` | Install without testing | 0 on success |
| None | Test with a 120 s timeout, then install | 0 on success |
| Anything else | Short help on stderr | 2 |

It works from its own folder and exits 1 if `go` is missing or tests or the build fail.
The binary goes to `$GOBIN`, or `$(go env GOPATH)/bin` when that is unset, overwriting any old one.
It prints `installed <path>`, and warns on stderr with the export line to use when that folder is not on `PATH`.

## Configuration

tuidraw has no configuration file and reads no environment variables.
All editor defaults are in [Editor](05-editor.md#state) and last only for the session.

## Deployment

There is no service or release process; distribution is `go install` from the public module, or a clone.

## Reference assets

| File | What |
|---|---|
| [`assets/showcase.tuidraw`](assets/showcase.tuidraw) | Sample project: a music player mockup, usable as a fixture |
| [`assets/screenshot.png`](assets/screenshot.png) | Screenshot of it, used by the root README and the [UI chapter](07-ui.md) |

### Redoing the screenshot

Done by hand, because window chrome varies.
Freeze mis-renders this app's output, so do not use it.

1. Run the app on the showcase in a clean tmux server: `tmux -L shot -f /dev/null`, status off, 92 by 25, so no personal configuration shows.
2. Select the play button.
3. Attach a Terminal.app window.
4. Capture it with `screencapture -l <window id> -o`.
5. Crop the title bar and scale to 1600 pixels wide.

Previous: [UI](07-ui.md) | Next: [Quality](09-quality.md)
