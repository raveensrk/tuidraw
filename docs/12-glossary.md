# Glossary

Terms with a specific meaning in this book.

| Term | Meaning |
|---|---|
| Canvas | The part of the screen showing the world, between toolbar and status row |
| Cell | One character position on the grid |
| Cursor | The world cell of the pointer, driven by both mouse and keyboard |
| Dirty | The drawing has changes not yet saved |
| Edge mask | The four bits saying which sides of a cell a border connects to |
| Elbow | A line's two straight legs meeting at one corner |
| Flip | Making an elbow's vertical leg come first |
| Handle | A cell of a lone selected shape that can be dragged to reshape it |
| Hit | A shape drawing a non-space rune at a cell |
| Ink | Foreground color |
| Junction | A cell where borders join, such as `┼` |
| Loose end | The last polygon point while placing, which follows the pointer |
| Marquee | The rectangle dragged on empty space to select shapes |
| Palette | The row of 16 colors shown while picking ink or background |
| Paper | The default background color for new shapes |
| Pointer | The press, move and release abstraction shared by mouse and keyboard |
| Project | A `.tuidraw` file: shapes that stay editable |
| Rune | A Unicode code point, drawn in one cell |
| SGR | Select Graphic Rendition, the ANSI escape codes that set colors |
| Solid | An opaque rune that replaces, rather than joins, what is below |
| Stroke | A brush shape: a list of painted cells |
| View | The world cell at the canvas's top-left |
| Window | The box of world cells a render covers |
| World | The infinite grid of cells that shapes live on |
| Z-order | Which shape is on top; later in the list is higher |

Previous: [Rebuild guide](11-rebuild-guide.md) | Next: [Open questions](13-open-questions.md)
