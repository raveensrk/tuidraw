package doc

import (
	"os"
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

func TestOldFileLoads(t *testing.T) {
	// A v1 file from before colors existed.
	path := filepath.Join(t.TempDir(), "old.tuidraw")
	old := `{"version":1,"shapes":[{"kind":"rect","box":{"min":{"x":0,"y":0},"max":{"x":3,"y":2}},"style":"light"}]}`
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if s := d.Shapes[0]; s.FG != "" || s.BG != "" || s.Fill {
		t.Errorf("old shape got colors: %+v", s)
	}
}

func TestColorIndex(t *testing.T) {
	if Color("").Index() != -1 || Color("black").Index() != 0 || Color("bright_white").Index() != 15 {
		t.Error("color index wrong")
	}
}
