// Package doc holds the editable drawing: an ordered list of shapes on an
// infinite character grid. Index order is z-order: later shapes draw on top.
package doc

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Version is the project file format version.
const Version = 1

// Kind names a shape type.
type Kind string

const (
	Rect    Kind = "rect"
	Ellipse Kind = "ellipse"
	Polygon Kind = "polygon"
	Line    Kind = "line"
	Text    Kind = "text"
	Stroke  Kind = "stroke"
	Group   Kind = "group"
)

// Style is a border style for rects and lines.
type Style string

const (
	Light   Style = "light"
	Heavy   Style = "heavy"
	Double  Style = "double"
	Rounded Style = "rounded"
	ASCII   Style = "ascii"
)

// Styles in the order the 1-5 keys pick them.
var Styles = []Style{Light, Heavy, Double, Rounded, ASCII}

// Color is one of the 16 ANSI terminal colors by name, or "" for the
// terminal's default. Names keep project files readable.
type Color string

// Colors in ANSI order: index 0-7 normal, 8-15 bright. The palette keys
// 1-8 and a-h pick them in this order.
var Colors = []Color{
	"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white",
	"bright_black", "bright_red", "bright_green", "bright_yellow",
	"bright_blue", "bright_magenta", "bright_cyan", "bright_white",
}

// Index is the color's ANSI number 0-15, or -1 for the default.
func (c Color) Index() int {
	for i, x := range Colors {
		if x == c {
			return i
		}
	}
	return -1
}

// Point is a cell on the grid. X grows right, Y grows down.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Box is an inclusive cell rectangle: Min and Max are both inside.
type Box struct {
	Min Point `json:"min"`
	Max Point `json:"max"`
}

// Span builds a box from two corners in any order.
func Span(a, b Point) Box {
	return Box{
		Min: Point{min(a.X, b.X), min(a.Y, b.Y)},
		Max: Point{max(a.X, b.X), max(a.Y, b.Y)},
	}
}

// Has reports whether p is inside the box.
func (b Box) Has(p Point) bool {
	return p.X >= b.Min.X && p.X <= b.Max.X && p.Y >= b.Min.Y && p.Y <= b.Max.Y
}

// Union grows the box to cover o.
func (b Box) Union(o Box) Box {
	return Box{
		Min: Point{min(b.Min.X, o.Min.X), min(b.Min.Y, o.Min.Y)},
		Max: Point{max(b.Max.X, o.Max.X), max(b.Max.Y, o.Max.Y)},
	}
}

// Overlaps reports whether two boxes share a cell.
func (b Box) Overlaps(o Box) bool {
	return b.Min.X <= o.Max.X && o.Min.X <= b.Max.X && b.Min.Y <= o.Max.Y && o.Min.Y <= b.Max.Y
}

// Shape is one drawable object. One struct for every kind keeps JSON and
// cloning simple; each kind reads only its own fields:
//
//	rect:    Box, Style
//	ellipse: Box, Style (circles are ellipses about twice as wide as tall)
//	polygon: Points (vertices), Closed, Style
//	line:   Points[0] start, Points[1] end, Style, Flip, Arrow
//	text:   Points[0] top-left, Text
//	stroke: Points (painted cells), Char
//	group:  Children
type Shape struct {
	Kind     Kind    `json:"kind"`
	Box      Box     `json:"box,omitzero"`
	Style    Style   `json:"style,omitempty"`
	Points   []Point `json:"points,omitempty"`
	Closed   bool    `json:"closed,omitempty"` // polygon: last vertex joins first
	Flip     bool    `json:"flip,omitempty"`   // line: vertical leg first
	Arrow    bool    `json:"arrow,omitempty"`  // line: arrowhead at end
	FG       Color   `json:"fg,omitempty"`     // ink: outline, text and brush color
	BG       Color   `json:"bg,omitempty"`     // background behind the shape's cells
	Fill     bool    `json:"fill,omitempty"`   // rect, ellipse, closed polygon: opaque inside
	Text     string  `json:"text,omitempty"`
	Char     string  `json:"char,omitempty"`
	Children []Shape `json:"children,omitempty"`
}

// Bounds is the smallest box covering every cell the shape draws.
func (s Shape) Bounds() Box {
	switch s.Kind {
	case Rect, Ellipse:
		return s.Box
	case Text:
		lines := strings.Split(s.Text, "\n")
		w := 1
		for _, l := range lines {
			w = max(w, len([]rune(l)))
		}
		p := s.Points[0]
		return Box{p, Point{p.X + w - 1, p.Y + len(lines) - 1}}
	case Group:
		b := s.Children[0].Bounds()
		for _, c := range s.Children[1:] {
			b = b.Union(c.Bounds())
		}
		return b
	}

	// line, polygon and stroke: cover their points
	b := Box{s.Points[0], s.Points[0]}
	for _, p := range s.Points[1:] {
		b = b.Union(Box{p, p})
	}
	return b
}

// Move shifts the shape by dx, dy in place.
func (s *Shape) Move(dx, dy int) {
	s.Box.Min.X += dx
	s.Box.Min.Y += dy
	s.Box.Max.X += dx
	s.Box.Max.Y += dy
	for i := range s.Points {
		s.Points[i].X += dx
		s.Points[i].Y += dy
	}
	for i := range s.Children {
		s.Children[i].Move(dx, dy)
	}
}

// Clone deep-copies the shape.
func (s Shape) Clone() Shape {
	c := s
	c.Points = append([]Point(nil), s.Points...)
	c.Children = Clones(s.Children)
	return c
}

// Clones deep-copies a shape list.
func Clones(list []Shape) []Shape {
	if list == nil {
		return nil
	}
	out := make([]Shape, len(list))
	for i, s := range list {
		out[i] = s.Clone()
	}
	return out
}

// SetStyle changes the border style of outlined shapes, recursing into groups.
func (s *Shape) SetStyle(st Style) {
	if s.Kind == Rect || s.Kind == Ellipse || s.Kind == Polygon || s.Kind == Line {
		s.Style = st
	}
	for i := range s.Children {
		s.Children[i].SetStyle(st)
	}
}

// Fillable reports whether the shape has an inside to fill.
func (s Shape) Fillable() bool {
	return s.Kind == Rect || s.Kind == Ellipse || (s.Kind == Polygon && s.Closed)
}

// Paint sets ink, background or fill, recursing into groups. A nil
// argument leaves that property alone.
func (s *Shape) Paint(fg, bg *Color, fill *bool) {
	if s.Kind == Group {
		for i := range s.Children {
			s.Children[i].Paint(fg, bg, fill)
		}
		return
	}
	if fg != nil {
		s.FG = *fg
	}
	if bg != nil {
		s.BG = *bg
	}
	if fill != nil && s.Fillable() {
		s.Fill = *fill
	}
}

// Doc is the whole drawing.
type Doc struct {
	Version int     `json:"version"`
	Shapes  []Shape `json:"shapes"`
}

// Load reads a project file.
func Load(path string) (Doc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Doc{}, err
	}
	var d Doc
	if err := json.Unmarshal(data, &d); err != nil {
		return Doc{}, fmt.Errorf("%s: %w", path, err)
	}
	if d.Version > Version {
		return Doc{}, fmt.Errorf("%s: format version %d is newer than %d", path, d.Version, Version)
	}
	return d, nil
}

// Save writes a project file atomically: temp file, then rename.
func Save(path string, d Doc) error {
	d.Version = Version
	if d.Shapes == nil {
		d.Shapes = []Shape{}
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
