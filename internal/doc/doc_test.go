package doc

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	d := Doc{Shapes: []Shape{
		{Kind: Rect, Box: Span(Point{4, 3}, Point{0, 0}), Style: Double},
		{Kind: Line, Points: []Point{{0, 5}, {9, 8}}, Style: Light, Flip: true, Arrow: true},
		{Kind: Group, Children: []Shape{
			{Kind: Text, Points: []Point{{1, 1}}, Text: "Login\nUser"},
			{Kind: Stroke, Points: []Point{{2, 2}, {3, 2}}, Char: "█"},
		}},
	}}

	path := filepath.Join(t.TempDir(), "a.tuidraw")
	if err := Save(path, d); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	d.Version = Version
	if !reflect.DeepEqual(got, d) {
		t.Errorf("round trip:\n%+v\nwant\n%+v", got, d)
	}
}

func TestBoundsAndMove(t *testing.T) {
	g := Shape{Kind: Group, Children: []Shape{
		{Kind: Rect, Box: Span(Point{0, 0}, Point{3, 3})},
		{Kind: Text, Points: []Point{{5, 1}}, Text: "abc\nde"},
	}}
	want := Box{Point{0, 0}, Point{7, 3}}
	if b := g.Bounds(); b != want {
		t.Errorf("bounds %v, want %v", b, want)
	}

	c := g.Clone()
	c.Move(2, 1)
	if g.Bounds() != want {
		t.Error("clone shares memory with original")
	}
	if b := c.Bounds(); b != (Box{Point{2, 1}, Point{9, 4}}) {
		t.Errorf("moved bounds %v", b)
	}
}
