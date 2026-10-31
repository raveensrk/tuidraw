# Quality

How tuidraw is tested, and the acceptance criteria its tests pin.

## Approach

Each package has table or example tests in its own folder, with no network or terminal.
UI tests drive the model directly by sending it key and mouse events, then read its shapes, or its view as a string.
There is no test of the real terminal, of `cmd/tuidraw`, or of `install.sh`; those are checked by hand.

The UI tests use an 80 by 24 window and a temporary project path.
Screen row 1 is world row 0.

## Acceptance criteria: rendering

| Given | When | Then |
|---|---|---|
| Each rune table | Measured | 16 runes; light has `┼` at N+E+S+W, `┌` at E+S, `┤` at N+S+W |
| Rect (0,0)-(3,2) | Drawn in each style | See the box figures below |
| Rects (0,0)-(3,2) and (3,0)-(6,2) | Drawn | Figure "shared wall" |
| Those, plus a line (0,1) to (6,1) | Drawn | Figure "crossing line" |
| Arrow line (0,0) to (3,2) | Drawn, then flipped | Figure "elbows" |
| Rect (0,0)-(5,2), text `Hi` at (1,0) | Drawn | `┌Hi──┐` on the top row |
| Rect (0,0)-(4,4) | Hit tested | Border cell hits, empty inside does not |
| Rounded ellipse in (0,0)-(10,5) | Drawn | Figure "ellipse" |
| Light rect (0,0)-(10,4), light ellipse (4,1)-(16,3) | Drawn | Figure "oval join" |
| Ellipse paths for boxes 16x6, 30x8, 4x2, 2x9 (max corner) | Walked as a loop | Every step moves exactly one cell |
| Closed polygons and an open one | Drawn | Figures "triangle", "diamond" and "polyline join" |
| Text `hidden` at (1,1), filled blue rect (0,0)-(4,2) on top | Drawn to (6,2) | Figure "fill"; inside bg blue, outside none |
| Red rect, green text `x` on its border | Read | Border ink red; the label is `x` in green |
| Blue filled ellipse (0,0)-(10,5) | Read | Outline (2,0) has no bg; inside (5,2) is blue |

```text
Styles:     ┌──┐  ┏━━┓  ╔══╗  ╭──╮  +--+
            │  │  ┃  ┃  ║  ║  │  │  |  |
            └──┘  ┗━━┛  ╚══╝  ╰──╯  +--+

Shared wall:  ┌──┬──┐    Crossing line:  ┌──┬──┐
              │  │  │                    ├──┼──┤
              └──┴──┘                    └──┴──┘

Elbows:  ───┐    │
            │    │
            ▼    └──▶

Ellipse:    ╭─────╮      Oval join:  ┌─────────┐
          ╭─╯     ╰─╮                │    ╭────┼────╮
          │         │                │    │    │    │
          │         │                │    ╰────┼────╯
          ╰─╮     ╭─╯                └─────────┘
            ╰─────╯

Triangle (6,0)(12,4)(0,4):      Diamond (6,0)(12,3)(6,6)(0,3):
      ╱╲                             ╱╱╲
    ╱╱ ╲╲                          ╱╱  ╲╲
   ╱     ╲                       ╱╱      ╲╲
 ╱╱       ╲╲                    <╲         ╲>
╱───────────╲                     ╲╲      ╱╱
                                    ╲╲  ╱╱
                                      ╲╱

Polyline join (2,0)(2,2)(6,2) over rect (0,0)-(4,3):   Fill:
┌─┬─┐                                                  ┌───┐
│ │ │                                                  │   │en
│ └─┼──                                                └───┘
└───┘
```

## Acceptance criteria: data and history

| Given | When | Then |
|---|---|---|
| A rect, a line, and a group of text and stroke | Saved and loaded | Equal to the original, version 1 |
| Group of rect (0,0)-(3,3) and text `abc\nde` at (5,1) | Bounds read | (0,0)-(7,3) |
| That group's clone | Moved by (2,1) | Clone bounds (2,1)-(9,4); original unchanged |
| A version 1 file from before colors | Loaded | Colors empty, fill off |
| Color names | Indexed | `""` is -1, `black` 0, `bright_white` 15 |
| Two pushes | Undo twice, then once more | Earlier states, then failure |
| Those undone | Redo twice | Final state |
| An undo then a push | Redo | Failure |
| A pushed snapshot | Live document changes, then undo | Snapshot unchanged |

## Acceptance criteria: exports

| Given | When | Then |
|---|---|---|
| Rounded rect (10,5)-(17,7), text `OK` at (12,6), arrow line (10,9) to (13,9) | Text export | `"╭──────╮\n│ OK   │\n╰──────╯\n\n───▶\n"` |
| No shapes | Text export | Empty string |
| [The colored example](06-interfaces.md#ansi-export) | ANSI export | Those exact bytes |
| An uncolored shape | ANSI export | Same as its text export |

## Acceptance criteria: editor

| Given | When | Then |
|---|---|---|
| Rect drag (0,1)-(4,3), line tool with arrow off, drag (0,2)-(4,2), text `OK` clicked at (1,1), esc | Text export | `"┌OK─┐\n├───┤\n└───┘\n"` |
| Two rects | Marquee both, `ctrl+g`, drag a border (0,1) to (2,2) | One group, moved to (2,1) in world cells |
| That group | `u` | Back at the origin |
| That group, clicked again after undo | `ctrl+g` | Two shapes |
| Cursor (0,0), rect tool | Enter, 3 rights, 2 downs, enter | Rect (0,0)-(3,2) from the keyboard alone |
| That rect | `3`, `ctrl+s`, reload | One double-style rect, not dirty |
| Rect from (0,1)-(3,3), select tool | Drag corner (3,3) to (6,5) | Box max is (6,4) |
| Clean editor, 24 rows | View | Exactly 24 rows |
| Help open | `/arrow`, enter | Page shows `TOOLS`, footer `match 1 of` |
| Circle tool | Drag (0,1)-(1,4) | Box (0,0)-(6,3) |
| Triangle tool, diamond tool | Drag each | Closed polygons of 3 and 4 points |
| Oval (30,1)-(38,4), select tool | Drag corner to (40,6) | Box max (40,5) |
| Polygon tool | Click 4 corners, then the first | One closed polygon of 4 points |
| Polygon tool | Click 3 points, esc | One open polygon of 3 points |
| Polygon tool | Click once, esc | No new shape |
| Ink red, paper blue, fill on | Draw a rect | It is red on blue, filled |
| That rect selected | `i`, `a` | Ink becomes `bright_black` |
| It selected | `I`, esc | Unchanged, palette closed |
| It selected, `i` | Click inside the third palette entry | Ink becomes green |
| A 6-cell stroke, eraser | Drag across 2 cells | 4 cells remain |
| A rect and that stroke, eraser | Click the rect's border | Only the stroke remains; `u` restores the rect |
| A 200-cell status message, 80 wide | Status row read | The message ends `x…`, then two spaces, then `light` |

## Performance

Each frame renders only the visible window and skips shapes outside it, with its cost growing in the screen size and shape count.
There are no benchmarks or targets; the test suite takes about 5 s.
Undo copies the whole document per edit and keeps [at most 200 snapshots](03-data.md#undo-snapshots).

## Security

| Area | Position |
|---|---|
| Network | None used |
| Files | Reads the file named; writes it, its `.tmp`, and sibling exports, with the user's permissions |
| Subprocess | Only `pbcopy`, resolved from `PATH` |
| Untrusted project files | Not validated; [can crash the editor](13-open-questions.md#known-issues) |
| Terminal escapes | Text from a project goes to the terminal through the rendering framework and into `.ans` files unfiltered |

Previous: [Build and run](08-build-run.md) | Next: [Decisions](10-decisions.md)
