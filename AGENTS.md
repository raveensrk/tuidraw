# AGENTS

Go + Bubble Tea v2 TUI. Layout:

- [cmd/tuidraw](cmd/tuidraw) - entry point, `-h`, `--help`, `help`.
- [internal/doc](internal/doc) - shapes, project file load and save.
- [internal/render](internal/render) - shapes to runes; edge masks give junction joins.
- [internal/history](internal/history) - undo and redo snapshots.
- [internal/export](internal/export) - plain text and ANSI export.
- [internal/ui](internal/ui) - editor model, pointer tools, view, `?` help pager.

- [doc](doc) - `showcase.tuidraw` and the README `screenshot.png`.

Redo the screenshot by hand (window chrome varies, so it is not scripted):
run the app on `doc/showcase.tuidraw` in a clean tmux server
(`tmux -L shot -f /dev/null`, status off, 92x25, so no personal config
shows), select the play button, attach a Terminal.app window, capture it
with `screencapture -l <window id> -o`, crop the title bar, scale to 1600 px
wide. Freeze mis-renders this app's output; do not use it.

Key bindings live in one place each: handlers in `internal/ui/model.go`, the
help page text in `internal/ui/help.go`. Change both together.

Test: `timeout 120 go test ./...`.
