// Package render turns shapes into characters on a grid.
//
// Borders are not drawn as runes directly. Each rect or line marks, per cell,
// which of its four sides it connects to (an edge mask). Masks from every
// shape are OR-ed together and only then mapped to a box-drawing rune, so
// shapes that touch or cross join up instead of overwriting each other:
//
//	┌──┐            ┌──┬──┐
//	│  │  + ──  ->  │  │  │
//	└──┘            └──┴──┘
//
// Text, brush strokes and arrowheads are opaque: they replace whatever edge
// sits below them in z-order.
package render

import (
	"strings"

	"tuidraw/internal/doc"
)

// Edge bits of a cell's mask.
const (
	N uint8 = 1 << iota
	E
	S
	W
)

// Rune tables indexed by mask (N|E|S|W). Index 0 is unused.
var (
	light   = []rune(" │─└││┌├─┘─┴┐┤┬┼")
	heavy   = []rune(" ┃━┗┃┃┏┣━┛━┻┓┫┳╋")
	double  = []rune(" ║═╚║║╔╠═╝═╩╗╣╦╬")
	rounded = []rune(" │─╰││╭├─╯─┴╮┤┬┼")
	ascii   = []rune(" |-+||++-+-+++++")
)

func table(st doc.Style) []rune {
	switch st {
	case doc.Heavy:
		return heavy
	case doc.Double:
		return double
	case doc.Rounded:
		return rounded
	case doc.ASCII:
		return ascii
	}
	return light
}

// Arrowheads by direction of travel, unicode then ascii.
var arrows = map[uint8][2]rune{
	N: {'▲', '^'},
	E: {'▶', '>'},
	S: {'▼', 'v'},
	W: {'◀', '<'},
}

type cell struct {
	mask   uint8
	style  doc.Style
	edgeZ  int  // z of the topmost shape that added an edge
	solid  rune // opaque rune, 0 if none
	solidZ int
}

// Grid is a rendered window of the infinite canvas.
type Grid struct {
	Box   doc.Box
	cells []cell
	w     int
}

// Draw renders shapes into the window box. Shapes outside are clipped.
func Draw(shapes []doc.Shape, box doc.Box) *Grid {
	w := box.Max.X - box.Min.X + 1
	h := box.Max.Y - box.Min.Y + 1
	g := &Grid{Box: box, cells: make([]cell, max(w*h, 0)), w: w}

	z := 0
	for _, s := range shapes {
		z = g.shape(s, z+1)
	}
	return g
}

// Hit reports whether shape s draws a visible rune at p.
func Hit(s doc.Shape, p doc.Point) bool {
	b := s.Bounds()
	if !b.Has(p) {
		return false
	}
	return Draw([]doc.Shape{s}, doc.Box{Min: p, Max: p}).Rune(p) != ' '
}

// Rune returns the character at p, or a space when empty or outside.
func (g *Grid) Rune(p doc.Point) rune {
	c := g.at(p)
	if c == nil {
		return ' '
	}
	if c.solid != 0 && c.solidZ >= c.edgeZ {
		return c.solid
	}
	if c.mask == 0 {
		return ' '
	}
	return table(c.style)[c.mask]
}

// Lines returns each row of the grid as a string.
func (g *Grid) Lines() []string {
	var out []string
	var sb strings.Builder
	for y := g.Box.Min.Y; y <= g.Box.Max.Y; y++ {
		sb.Reset()
		for x := g.Box.Min.X; x <= g.Box.Max.X; x++ {
			sb.WriteRune(g.Rune(doc.Point{X: x, Y: y}))
		}
		out = append(out, sb.String())
	}
	return out
}

func (g *Grid) at(p doc.Point) *cell {
	if !g.Box.Has(p) {
		return nil
	}
	return &g.cells[(p.Y-g.Box.Min.Y)*g.w+(p.X-g.Box.Min.X)]
}

func (g *Grid) edge(p doc.Point, mask uint8, st doc.Style, z int) {
	c := g.at(p)
	if c == nil {
		return
	}
	c.mask |= mask
	c.style = st
	c.edgeZ = z
}

func (g *Grid) solid(p doc.Point, r rune, z int) {
	c := g.at(p)
	if c == nil {
		return
	}
	c.solid = r
	c.solidZ = z
}

// shape draws s at z and returns the last z used (groups use one per child).
func (g *Grid) shape(s doc.Shape, z int) int {
	// Skip shapes fully outside the window, cheap for big canvases.
	if s.Kind != doc.Group && !s.Bounds().Overlaps(g.Box) {
		return z
	}

	switch s.Kind {
	case doc.Rect:
		g.rect(s, z)
	case doc.Line:
		g.line(s, z)
	case doc.Text:
		g.text(s, z)
	case doc.Stroke:
		r := []rune(s.Char)
		if len(r) == 0 {
			r = []rune{'#'}
		}
		for _, p := range s.Points {
			g.solid(p, r[0], z)
		}
	case doc.Group:
		for i, c := range s.Children {
			if i > 0 {
				z++
			}
			z = g.shape(c, z)
		}
	}
	return z
}

func (g *Grid) rect(s doc.Shape, z int) {
	b := s.Box
	for x := b.Min.X; x <= b.Max.X; x++ {
		var m uint8
		if x > b.Min.X {
			m |= W
		}
		if x < b.Max.X {
			m |= E
		}
		top, bot := m, m
		if b.Max.Y > b.Min.Y {
			if x == b.Min.X || x == b.Max.X {
				top |= S
				bot |= N
			}
		}
		g.edge(doc.Point{X: x, Y: b.Min.Y}, top, s.Style, z)
		g.edge(doc.Point{X: x, Y: b.Max.Y}, bot, s.Style, z)
	}
	for y := b.Min.Y + 1; y < b.Max.Y; y++ {
		g.edge(doc.Point{X: b.Min.X, Y: y}, N|S, s.Style, z)
		g.edge(doc.Point{X: b.Max.X, Y: y}, N|S, s.Style, z)
	}
}

// Path returns the cells of a line: horizontal leg then vertical leg, or
// vertical first when flipped.
//
//	a───┐        a
//	    │        │
//	    b        └───b   (flip)
func Path(s doc.Shape) []doc.Point {
	a, b := s.Points[0], s.Points[1]
	corner := doc.Point{X: b.X, Y: a.Y}
	if s.Flip {
		corner = doc.Point{X: a.X, Y: b.Y}
	}

	var out []doc.Point
	walk := func(from, to doc.Point) {
		dx, dy := sign(to.X-from.X), sign(to.Y-from.Y)
		p := from
		for p != to {
			out = append(out, p)
			p.X += dx
			p.Y += dy
		}
	}
	walk(a, corner)
	walk(corner, b)
	return append(out, b)
}

func (g *Grid) line(s doc.Shape, z int) {
	path := Path(s)
	if len(path) == 1 {
		g.edge(path[0], E|W, s.Style, z)
		return
	}

	for i := 0; i < len(path)-1; i++ {
		d := dir(path[i], path[i+1])
		g.edge(path[i], d, s.Style, z)
		g.edge(path[i+1], opposite(d), s.Style, z)
	}

	if s.Arrow {
		n := len(path)
		head := arrows[dir(path[n-2], path[n-1])]
		r := head[0]
		if s.Style == doc.ASCII {
			r = head[1]
		}
		g.solid(path[n-1], r, z)
	}
}

func (g *Grid) text(s doc.Shape, z int) {
	p := s.Points[0]
	for dy, l := range strings.Split(s.Text, "\n") {
		for dx, r := range []rune(l) {
			g.solid(doc.Point{X: p.X + dx, Y: p.Y + dy}, r, z)
		}
	}
}

// dir is the edge bit pointing from a to its neighbour b.
func dir(a, b doc.Point) uint8 {
	switch {
	case b.Y < a.Y:
		return N
	case b.X > a.X:
		return E
	case b.Y > a.Y:
		return S
	}
	return W
}

func opposite(d uint8) uint8 {
	switch d {
	case N:
		return S
	case E:
		return W
	case S:
		return N
	}
	return E
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}
