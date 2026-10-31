# Rebuild guide

Ordered milestones from an empty folder to an equivalent tuidraw.

Each milestone builds on the last, and the criteria in [Quality](09-quality.md) are its tests.

| # | Milestone | Read | Verify |
|---|---|---|---|
| 1 | Module with the [pinned dependencies](08-build-run.md#dependencies) and the [package layout](02-architecture.md#packages) | Architecture, Build and run | `go build ./...` passes |
| 2 | Shape model, geometry, project file | [Data](03-data.md) | [Data criteria](09-quality.md#acceptance-criteria-data-and-history), and [`showcase.tuidraw`](assets/showcase.tuidraw) survives a load and save with the same shapes |
| 3 | History | [Undo snapshots](03-data.md#undo-snapshots) | History criteria |
| 4 | Grid, masks, rect, line, text, stroke, group, z-order | [Rendering](04-rendering.md) | Styles, junction, elbow and label figures |
| 5 | Ellipse, polygon, fill, colors, hit, inside | [Rendering](04-rendering.md) | [All rendering criteria](09-quality.md#acceptance-criteria-rendering) |
| 6 | Text and ANSI export | [Export files](06-interfaces.md#export-files) | [Export criteria](09-quality.md#acceptance-criteria-exports) |
| 7 | Editor shell: state, screen layout, toolbar, status row, canvas painting, command line | [Editor state](05-editor.md#state), [UI](07-ui.md), [Command line](06-interfaces.md#command-line) | App opens the showcase; 24-row view criterion |
| 8 | Pointer, select, move, handles, every drawing tool, eraser | [Editor](05-editor.md#pointer-drag) | Drawing and tool [editor criteria](09-quality.md#acceptance-criteria-editor) |
| 9 | All commands: text editing, keys, palette, clipboard, undo, save, export, quit | [Editor](05-editor.md), [Key bindings](06-interfaces.md#key-bindings) | All editor criteria; every key in the table does what it says |
| 10 | Help pager and help text; `install.sh` | [Help pager](07-ui.md#help-pager), [install.sh](08-build-run.md#installsh) | Pager criterion; `./install.sh` installs a working binary |

## Final check

1. Open `showcase.tuidraw` and compare with the [screenshot](assets/screenshot.png).
2. Save it under a new name, and confirm the shapes are unchanged.
3. Export it, and `cat` the `.ans` in a terminal.
4. Draw a box and a crossing line in a real terminal with the mouse, then again with only the keyboard.

Previous: [Decisions](10-decisions.md) | Next: [Glossary](12-glossary.md)
