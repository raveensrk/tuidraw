# AGENTS

Go + Bubble Tea v2 TUI. Layout:

- [cmd/tuidraw](cmd/tuidraw) - entry point, `-h`, `--help`, `help`.
- [internal/doc](internal/doc) - shapes, project file load and save.
- [internal/render](internal/render) - shapes to runes; edge masks give junction joins.
- [internal/history](internal/history) - undo and redo snapshots.
- [internal/export](internal/export) - plain text export.
- [internal/ui](internal/ui) - editor model, pointer tools, view, `?` help pager.

Key bindings live in one place each: handlers in `internal/ui/model.go`, the
help page text in `internal/ui/help.go`. Change both together.

Test: `timeout 120 go test ./...`.
