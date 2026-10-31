# Open questions

What the repository cannot answer, and behavior that looks unintended.

## Unknowns

| Question | Where it matters |
|---|---|
| Why Go and Bubble Tea, rather than another language or framework? | [ADR 10](10-decisions.md#adr-10-go-and-bubble-tea), [Tech stack](02-architecture.md#tech-stack) |

## Known issues

A rebuild should reproduce the exact contracts, but need not reproduce these.
None has been fixed; each was confirmed by running it except where noted.

| Issue | Behavior | Where it matters |
|---|---|---|
| Malformed files crash | A shape without its required `points` passes Load, then panics when drawn | [Loading](03-data.md#loading) |
| Empty group | A group with no children opens, then panics once its bounds are needed, such as on export (from reading the code) | [Invariants](03-data.md#invariants) |
| Group fill stays on | `F` on a selected group can only turn fill on, because a group's own `fill` is never set | [Style commands](05-editor.md#style-commands) |
| `ctrl+s` while typing | It finishes the text edit, but does not save | [Text editing](05-editor.md#text-editing) |
| `ctrl+e` on an unnamed drawing | It adopts the path and exports, but writes no project file | [Saving](05-editor.md#saving-exporting-quitting) |
| Empty export | An empty drawing still writes empty `.txt` and `.ans` files, reported as exported | [Export files](06-interfaces.md#export-files) |
| Paste in other modes | Pasting while help or a prompt is open adds a text shape to the drawing | [Clipboard](05-editor.md#clipboard) |
| Polygon points past bounds | A top or bottom point at the rightmost column writes outside the bounds, so exports crop it | [Polygon](04-rendering.md#polygon) |
| Flag-like paths | `tuidraw -x` opens a file named `-x` | [Command line](06-interfaces.md#command-line) |
| Wide characters | Each rune takes one cell, so wide runes shift the row; layout of them is a [non-goal](01-overview.md#non-goals) | [Text](04-rendering.md#text) |

Previous: [Glossary](12-glossary.md)
