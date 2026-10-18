// Package export writes a drawing as plain text or as ANSI color text.
package export

import (
	"os"
	"strconv"
	"strings"

	"github.com/raveensrk/tuidraw/internal/doc"
	"github.com/raveensrk/tuidraw/internal/render"
)

// bounds is the box covering every shape.
func bounds(shapes []doc.Shape) doc.Box {
	b := shapes[0].Bounds()
	for _, s := range shapes[1:] {
		b = b.Union(s.Bounds())
	}
	return b
}

// Text renders shapes cropped to their bounds, with trailing spaces trimmed.
// An empty list gives an empty string.
func Text(shapes []doc.Shape) string {
	if len(shapes) == 0 {
		return ""
	}
	lines := render.Draw(shapes, bounds(shapes)).Lines()
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n") + "\n"
}

// Save writes the text export to path.
func Save(path string, shapes []doc.Shape) error {
	return os.WriteFile(path, []byte(Text(shapes)), 0o644)
}

// SGR codes: ink is 30-37 (bright 90-97), background 40-47 (bright
// 100-107); 39 and 49 restore the terminal defaults; 0 resets all.
const (
	fgBase       = 30
	fgBrightBase = 90
	bgBase       = 40
	bgBrightBase = 100
	fgDefault    = 39
	bgDefault    = 49
	reset        = "\x1b[0m"
)

func sgr(c doc.Color, base, bright, def int) string {
	i := c.Index()
	switch {
	case i < 0:
		return strconv.Itoa(def)
	case i < 8:
		return strconv.Itoa(base + i)
	}
	return strconv.Itoa(bright + i - 8)
}

// ANSI renders shapes like Text, with color escape codes. Codes are sent
// only when the color changes, and each line ends reset, so the output is
// safe to cat, paste or print from a script.
func ANSI(shapes []doc.Shape) string {
	if len(shapes) == 0 {
		return ""
	}
	b := bounds(shapes)
	g := render.Draw(shapes, b)

	var out strings.Builder
	for y := b.Min.Y; y <= b.Max.Y; y++ {
		// Trim trailing cells that are blank with no background.
		end := b.Min.X - 1
		for x := b.Max.X; x >= b.Min.X; x-- {
			if r, _, bg := g.Cell(doc.Point{X: x, Y: y}); r != ' ' || bg != "" {
				end = x
				break
			}
		}

		var fg, bg doc.Color
		colored := false
		for x := b.Min.X; x <= end; x++ {
			r, f, k := g.Cell(doc.Point{X: x, Y: y})
			if r == ' ' {
				f = fg // ink does not matter on a blank; avoid a code
			}
			if f != fg || k != bg {
				out.WriteString("\x1b[" + sgr(f, fgBase, fgBrightBase, fgDefault) + ";" + sgr(k, bgBase, bgBrightBase, bgDefault) + "m")
				fg, bg = f, k
				colored = true
			}
			out.WriteRune(r)
		}
		if colored {
			out.WriteString(reset)
		}
		out.WriteString("\n")
	}
	return out.String()
}

// SaveANSI writes the ANSI export to path.
func SaveANSI(path string, shapes []doc.Shape) error {
	return os.WriteFile(path, []byte(ANSI(shapes)), 0o644)
}
