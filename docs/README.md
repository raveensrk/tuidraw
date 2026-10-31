# The tuidraw book

The design source of truth for tuidraw: enough to understand, maintain or rebuild it without the source code.

tuidraw is an infinite-canvas editor for designing terminal UIs, and is itself a terminal app.

## Contents

Read in this order for a rebuild; each chapter also stands alone as a reference.

| Chapter | Purpose |
|---|---|
| [Overview](01-overview.md) | Problem, users, scope, non-goals, success criteria |
| [Architecture](02-architecture.md) | System context, packages, tech stack |
| [Data](03-data.md) | Shape model, project file, invariants, undo snapshots |
| [Rendering](04-rendering.md) | How shapes become characters and colors |
| [Editor](05-editor.md) | Editor state, input routing, tools, editing behavior |
| [Interfaces](06-interfaces.md) | CLI, key bindings, export formats, clipboard |
| [UI](07-ui.md) | Screens, layout, colors, status messages, help pager |
| [Build and run](08-build-run.md) | Toolchain, dependencies, commands, configuration |
| [Quality](09-quality.md) | Tests as acceptance criteria, performance, security |
| [Decisions](10-decisions.md) | Why the main design choices were made |
| [Rebuild guide](11-rebuild-guide.md) | Ordered milestones from an empty folder |
| [Glossary](12-glossary.md) | Domain terms |
| [Open questions](13-open-questions.md) | Unknowns and known issues |

## Conventions

- One fact lives in one place; other places link to it.
- GitHub Flavored Markdown, with callouts only as GitHub alerts, no raw HTML, tables of at most five columns.
- One sentence per line.
- Each chapter has an H1 title, a one-line purpose, a body, then previous and next links.
- Links to source files are for maintainers; nothing in the book depends on them.
- Diagram sources live in `diagrams/` as `<chapter>-<name>.mmd` or `.d2`, with the rendered `.svg` committed next to each.
- D2 with the TALA layout engine draws architecture diagrams; Mermaid draws the rest.
- Diagrams render on an opaque white background and stay near 15 nodes.
- Change a behavior, interface, data shape or dependency together with the chapter and diagram that describe it.

## Render and check

Run from anywhere; the script finds its own folder.

```bash
docs/build.sh diagrams
```

```bash
docs/build.sh check
```

```bash
docs/build.sh export
```

| Command | Does |
|---|---|
| `diagrams` | Renders every source in `diagrams/` to SVG |
| `check` | Runs markdownlint, an offline link and anchor check, and a full re-render |
| `export` | Writes the whole book as PDF and EPUB into `_build/`, which git ignores |

The script names the versions it needs and how to install what is missing.

Next: [Overview](01-overview.md)
