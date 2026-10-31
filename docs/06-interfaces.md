# Interfaces

Every contract tuidraw exposes: command line, keys, export files and clipboard.

The project file has its own contract in [Data](03-data.md#project-file).

## Command line

Source: [`cmd/tuidraw/main.go`](../cmd/tuidraw/main.go).

```text
tuidraw [file.tuidraw]
```

| Arguments | Behavior | Exit |
|---|---|---|
| `-h` or `--help` | Short help on stdout: usage, one paragraph, options, example | 0 |
| `help` | Long help: the short help, plus a pointer to `?`, files, environment, exit codes | 0 |
| None | Start an unnamed drawing | 0 on quit |
| One other argument | Treat it as the path, even if it starts with `-` | 0 on quit |
| Two or more | Short help on stderr | 2 |

| Failure | Stderr | Exit |
|---|---|---|
| File cannot be opened or parsed | `tuidraw: <error>` | 1 |
| Terminal failed | `tuidraw: <error>` | 1 |

The program reads no environment variables.

## Key bindings

Key names follow Bubble Tea: `ctrl+s`, `shift+left`, `space`, `esc`.
A capital letter is that shifted key.

### Tools

| Key | Tool | Toolbar label |
|---|---|---|
| `v` | Select | `sel` |
| `r` | Rect | `rect` |
| `o` | Oval | `oval` |
| `O` | Circle | `circle` |
| `y` | Triangle | `tri` |
| `d` | Diamond | `diamond` |
| `p` | Polygon | `poly` |
| `l` | Line | `line` |
| `t` | Text | `text` |
| `b` | Brush | `brush` |
| `e` | Eraser | `erase` |

### Canvas

| Key | Action |
|---|---|
| `?` | Open help |
| `q` | Quit, twice if unsaved |
| `ctrl+q` | Quit at once |
| `esc` | Finish a polygon, else cancel a drag, else clear selection and status |
| Arrows | Move the keyboard cursor |
| `enter`, `space` | Press the pointer, or release it during a drag |
| `H` `J` `K` `L`, `shift+` arrows | Nudge the selection left, down, up, right |
| `0` | View to the origin, cursor to the middle of the screen |
| `1` `2` `3` `4` `5` | Border style: light, heavy, double, rounded, ascii |
| `a` | Toggle arrow |
| `f` | Flip the elbow of selected lines |
| `i` | Pick ink from the palette |
| `I` | Pick background from the palette |
| `F` | Toggle fill |
| `c` | Next key sets the brush character |
| `x`, `delete`, `backspace` | Delete the selection |
| `ctrl+a` | Select all |
| `u`, `ctrl+z` | Undo |
| `U`, `ctrl+r`, `ctrl+y`, `ctrl+shift+z` | Redo |
| `ctrl+c` | Copy |
| `ctrl+x` | Cut |
| `ctrl+v` | Paste at the cursor |
| `ctrl+d` | Duplicate |
| `ctrl+g` | Group, or ungroup a lone group |
| `]`, `[` | Forward, backward one step |
| `}`, `{` | To front, to back |
| `ctrl+s` | Save |
| `ctrl+e` | Export `.txt` and `.ans` |

Other keys do nothing.
Arrows drive the cursor because `hjkl` would clash with `l`, the line tool.

### Palette

| Key | Color |
|---|---|
| `0` | Terminal default |
| `1` to `8` | black, red, green, yellow, blue, magenta, cyan, white |
| `a` to `h` | bright versions in the same order |

Text edit and prompt keys are in [Editor](05-editor.md#key-routing); help page keys in [UI](07-ui.md#help-pager).

### Mouse

| Input | Action |
|---|---|
| Left button | Press, drag, release the pointer |
| Shift, ctrl or alt with left press | Toggle a shape in the selection |
| Double-click with the select tool | Edit a text shape |
| Right or middle drag | Pan |
| Wheel | Scroll; shift scrolls sideways |
| Toolbar click | Choose a tool |

## Export files

Source: [`internal/export/export.go`](../internal/export/export.go).

Both exports render the whole drawing cropped to the union of all top-level shape bounds, and keep blank rows inside that box.
An empty drawing gives empty files.
Files are written directly with mode 0644, not atomically, overwriting without asking.

### Text export

Each row has trailing spaces trimmed and ends in `\n`.

```text
╭──────╮
│ OK   │
╰──────╯

───▶
```

### ANSI export

```text
for each row:
    end = last column whose rune is not a space or whose bg is set
    fg, bg = default, default;  colored = false
    for x in first column..end:
        r, f, k = cell at x
        if r is a space: f = fg               # ink is irrelevant on a blank
        if f != fg or k != bg:
            emit ESC "[" code(f, 30, 90, 39) ";" code(k, 40, 100, 49) "m"
            fg, bg = f, k;  colored = true
        emit r
    if colored: emit ESC "[0m"
    emit newline

code(c, base, bright, default):
    default if c is default;  base+n if c is color n in 0..7;  bright+n-8 if in 8..15
```

A drawing with no colors therefore exports the same bytes as the text export.

Text `ab` in red followed by `c` in bright blue on yellow, and `d` uncolored on the next row, gives:

```text
ESC[31;49mabESC[94;43mcESC[0m
d
```

## System clipboard

| Step | Behavior |
|---|---|
| 1 | If `pbcopy` is on `PATH`, pipe the text to it and stop if it succeeds |
| 2 | Otherwise ask the terminal to set the clipboard with [OSC 52](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html) |

Previous: [Editor](05-editor.md) | Next: [UI](07-ui.md)
