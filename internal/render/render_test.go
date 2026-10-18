package render

import (
	"strings"
	"testing"

	"tuidraw/internal/doc"
)

func pt(x, y int) doc.Point { return doc.Point{X: x, Y: y} }

func rect(x0, y0, x1, y1 int, st doc.Style) doc.Shape {
	return doc.Shape{Kind: doc.Rect, Box: doc.Span(pt(x0, y0), pt(x1, y1)), Style: st}
}

func draw(shapes ...doc.Shape) string {
	b := shapes[0].Bounds()
	for _, s := range shapes[1:] {
		b = b.Union(s.Bounds())
	}
	return strings.Join(Draw(shapes, b).Lines(), "\n")
}

func TestTables(t *testing.T) {
	for name, tb := range map[string][]rune{"light": light, "heavy": heavy, "double": double, "rounded": rounded, "ascii": ascii} {
		if len(tb) != 16 {
			t.Errorf("%s table has %d runes, want 16", name, len(tb))
		}
	}
	if light[N|E|S|W] != '┼' || light[E|S] != '┌' || light[N|S|W] != '┤' {
		t.Error("light table out of order")
	}
}

func TestRectStyles(t *testing.T) {
	cases := map[doc.Style]string{
		doc.Light:   "┌──┐\n│  │\n└──┘",
		doc.Heavy:   "┏━━┓\n┃  ┃\n┗━━┛",
		doc.Double:  "╔══╗\n║  ║\n╚══╝",
		doc.Rounded: "╭──╮\n│  │\n╰──╯",
		doc.ASCII:   "+--+\n|  |\n+--+",
	}
	for st, want := range cases {
		if got := draw(rect(0, 0, 3, 2, st)); got != want {
			t.Errorf("%s:\n%s\nwant\n%s", st, got, want)
		}
	}
}

func TestJunctions(t *testing.T) {
	// Two boxes sharing a wall, and a line crossing it.
	a := rect(0, 0, 3, 2, doc.Light)
	b := rect(3, 0, 6, 2, doc.Light)
	want := "┌──┬──┐\n│  │  │\n└──┴──┘"
	if got := draw(a, b); got != want {
		t.Errorf("shared wall:\n%s\nwant\n%s", got, want)
	}

	line := doc.Shape{Kind: doc.Line, Points: []doc.Point{pt(0, 1), pt(6, 1)}, Style: doc.Light}
	want = "┌──┬──┐\n├──┼──┤\n└──┴──┘"
	if got := draw(a, b, line); got != want {
		t.Errorf("crossing line:\n%s\nwant\n%s", got, want)
	}
}

func TestElbowAndArrow(t *testing.T) {
	l := doc.Shape{Kind: doc.Line, Points: []doc.Point{pt(0, 0), pt(3, 2)}, Style: doc.Light, Arrow: true}
	want := "───┐\n   │\n   ▼"
	if got := draw(l); got != want {
		t.Errorf("elbow:\n%s\nwant\n%s", got, want)
	}

	l.Flip = true
	want = "│   \n│   \n└──▶"
	if got := draw(l); got != want {
		t.Errorf("flipped:\n%s\nwant\n%s", got, want)
	}
}

func TestTextCoversEdges(t *testing.T) {
	box := rect(0, 0, 5, 2, doc.Light)
	label := doc.Shape{Kind: doc.Text, Points: []doc.Point{pt(1, 0)}, Text: "Hi"}
	want := "┌Hi──┐\n│    │\n└────┘"
	if got := draw(box, label); got != want {
		t.Errorf("label on border:\n%s\nwant\n%s", got, want)
	}
}

func TestHit(t *testing.T) {
	r := rect(0, 0, 4, 4, doc.Light)
	if !Hit(r, pt(0, 2)) {
		t.Error("border cell should hit")
	}
	if Hit(r, pt(2, 2)) {
		t.Error("empty inside should not hit")
	}
}
