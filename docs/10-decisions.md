# Decisions

Why the main design choices were made, as short records adapted from [MADR](https://adr.github.io/madr/).

Where the repository does not record the alternatives weighed, the record names the obvious one and says so.

## ADR 1: Join borders with edge masks

- **Context:** boxes and lines that touch or cross should look joined, not overwrite each other.
- **Decision:** each shape marks which of four sides a cell connects to; masks are OR-ed, and only then mapped to a rune.
- **Alternatives:** drawing runes directly, which would need rewriting neighbors after each shape.
- **Consequences:** junctions come for free in every style; a cell has one style, the last written; text, strokes and arrowheads need a separate opaque layer.

## ADR 2: One record type for every shape kind

- **Context:** shapes go to JSON and are cloned for every undo step.
- **Decision:** one struct holds all fields, and each kind reads only its own.
- **Alternatives:** a type per kind behind an interface, which needs custom JSON and cloning; not recorded as weighed.
- **Consequences:** JSON and cloning stay simple; nothing in the type system stops a malformed shape, which is [why loading can crash](13-open-questions.md#known-issues).

## ADR 3: Undo with whole-document snapshots

- **Context:** undo must never disagree with the document.
- **Decision:** deep-copy the whole shape list per edit, keeping 200.
- **Alternatives:** inverse operations, rejected because they can drift out of sync.
- **Consequences:** trivially correct and cheap for small drawings; memory grows with drawing size.

## ADR 4: JSON project file

- **Context:** drawings must stay editable after saving.
- **Decision:** store shapes, not characters, as indented JSON with named colors and styles.
- **Alternatives:** a binary or custom text format; not recorded as weighed.
- **Consequences:** human-readable and easy to diff; files are verbose.

## ADR 5: Only 16 named ANSI colors

- **Context:** the design should match what a terminal shows.
- **Decision:** colors are the 16 ANSI colors by name, rendered as the terminal's own numbered colors.
- **Alternatives:** 256 colors or truecolor, a [non-goal](01-overview.md#non-goals).
- **Consequences:** editor and `.ans` export agree in any terminal, under its theme; no exact color matching.

## ADR 6: Lines are two-leg elbows

- **Context:** connectors must stay on the grid and join borders.
- **Decision:** a line has a start, an end and a flip for which leg comes first; the first drag direction sets the flip.
- **Alternatives:** free routing or many bends; open polygons cover multi-bend paths.
- **Consequences:** drawing one is a single drag; complex routes use a polygon, which has no arrowhead.

## ADR 7: One pointer for mouse and keyboard

- **Context:** the keyboard should do everything the mouse does.
- **Decision:** a click is press and release at the mouse cell; enter is press, then release, at the keyboard cursor; tools see only the pointer.
- **Alternatives:** separate keyboard commands per tool; not recorded as weighed.
- **Consequences:** tools are written once; a keyboard drag takes two presses.

## ADR 8: Solid slanted polygon sides

- **Context:** a diagonal has no box-drawing edge to join.
- **Decision:** straight sides use edge masks, slanted sides use solid `╱ ╲` runes chosen by on-screen angle, and points get pointed shapes.
- **Alternatives:** stair-stepped outlines, as ellipses use; not recorded as weighed.
- **Consequences:** triangles and diamonds read as such; slanted sides do not join other borders.

## ADR 9: The book carries the full key table

- **Context:** the author's rule is that a doc never copies an option list out of a program's help, but a rebuild without source needs every binding.
- **Decision:** this book holds the full [CLI and key contract](06-interfaces.md) as an explicit exception; the program's help remains its own copy.
- **Alternatives:** linking to `--help` only, which would fail the rebuild goal.
- **Consequences:** a binding change touches the handler, the help page and the book together.

## ADR 10: Go and Bubble Tea

- **Context:** the app needs a terminal UI framework and ships as one binary.
- **Decision:** Go with Bubble Tea v2 and Lip Gloss v2.
- **Alternatives:** unknown; the reason is [an open question](13-open-questions.md#unknowns).
- **Consequences:** no runtime to install; a Go toolchain is needed to build.

Previous: [Quality](09-quality.md) | Next: [Rebuild guide](11-rebuild-guide.md)
