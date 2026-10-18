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
// sits below them in z-order. A filled shape wipes every cell it covers
// before drawing its outline, so it hides what lies below.
//
// Colors ride along: each cell keeps the ink of whichever layer shows, and
// the background of the topmost shape that set one.
package render

import (
	"math"
	"strings"

	"github.com/raveensrk/tuidraw/internal/doc"
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
	mask    uint8
	style   doc.Style
	edgeZ   int // z of the topmost shape that added an edge
	edgeFG  doc.Color
	solid   rune // opaque rune, 0 if none
	solidZ  int
	solidFG doc.Color
	bg      doc.Color
}

// Grid is a rendered window of the infinite canvas.
type Grid struct {
	Box   doc.Box
	cells []cell
	w     int

	cur doc.Shape // shape being drawn, for its colors
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

// Cell returns the character at p with its ink and background colors.
func (g *Grid) Cell(p doc.Point) (rune, doc.Color, doc.Color) {
	c := g.at(p)
	if c == nil {
		return ' ', "", ""
	}
	fg := c.edgeFG
	if c.solid != 0 && c.solidZ >= c.edgeZ {
		fg = c.solidFG
	}
	return g.Rune(p), fg, c.bg
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
	c.edgeFG = g.cur.FG
	g.paint(c)
}

// paint gives the cell the current shape's background, when it has one or
// is filled (a filled shape with no color clears what was below).
func (g *Grid) paint(c *cell) {
	if g.cur.BG != "" || g.cur.Fill {
		c.bg = g.cur.BG
	}
}

// wipe empties a cell for a filled shape: nothing below shows through.
func (g *Grid) wipe(p doc.Point, z int) {
	c := g.at(p)
	if c == nil {
		return
	}
	*c = cell{edgeZ: z, solidZ: z}
	g.paint(c)
}

func (g *Grid) solid(p doc.Point, r rune, z int) {
	c := g.at(p)
	if c == nil {
		return
	}
	c.solid = r
	c.solidZ = z
	c.solidFG = g.cur.FG
	g.paint(c)
}

// shape draws s at z and returns the last z used (groups use one per child).
func (g *Grid) shape(s doc.Shape, z int) int {
	// Skip shapes fully outside the window, cheap for big canvases.
	if s.Kind != doc.Group && !s.Bounds().Overlaps(g.Box) {
		return z
	}

	if s.Kind != doc.Group {
		g.cur = s
		if s.Fill && s.Fillable() {
			g.fill(s, z)
		}
	}

	switch s.Kind {
	case doc.Rect:
		g.rect(s, z)
	case doc.Ellipse:
		g.ellipse(s, z)
	case doc.Polygon:
		g.polygon(s, z)
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

// fill wipes every cell the shape covers, outline included.
func (g *Grid) fill(s doc.Shape, z int) {
	b := s.Bounds()
	for y := max(b.Min.Y, g.Box.Min.Y); y <= min(b.Max.Y, g.Box.Max.Y); y++ {
		for x := max(b.Min.X, g.Box.Min.X); x <= min(b.Max.X, g.Box.Max.X); x++ {
			if p := (doc.Point{X: x, Y: y}); Inside(s, p) {
				g.wipe(p, z)
			}
		}
	}
	if s.Kind == doc.Ellipse {
		for _, p := range EllipsePath(s.Box) {
			g.wipe(p, z)
		}
	}
	if s.Kind == doc.Polygon {
		n := len(s.Points)
		for i := range n {
			for _, p := range Bresenham(s.Points[i], s.Points[(i+1)%n]) {
				g.wipe(p, z)
			}
		}
	}
}

// Inside reports whether p is inside a rect, text block, ellipse, closed
// polygon or any child of a group. Lines and strokes have no inside.
func Inside(s doc.Shape, p doc.Point) bool {
	switch s.Kind {
	case doc.Rect, doc.Text:
		return s.Bounds().Has(p)
	case doc.Ellipse:
		b := s.Box
		a := max(float64(b.Max.X-b.Min.X)/2, 0.5)
		r := max(float64(b.Max.Y-b.Min.Y)/2, 0.5)
		nx := (float64(p.X) - float64(b.Min.X+b.Max.X)/2) / a
		ny := (float64(p.Y) - float64(b.Min.Y+b.Max.Y)/2) / r
		return nx*nx+ny*ny <= 1
	case doc.Polygon:
		return s.Closed && inPolygon(s.Points, p)
	case doc.Group:
		for _, c := range s.Children {
			if Inside(c, p) {
				return true
			}
		}
	}
	return false
}

// inPolygon is the even-odd ray test at the cell centre.
func inPolygon(pts []doc.Point, p doc.Point) bool {
	in := false
	x, y := float64(p.X), float64(p.Y)
	for i, j := 0, len(pts)-1; i < len(pts); j, i = i, i+1 {
		a, b := pts[i], pts[j]
		ay, by := float64(a.Y), float64(b.Y)
		if (ay > y) == (by > y) {
			continue
		}
		cx := float64(a.X) + (y-ay)/(by-ay)*float64(b.X-a.X)
		if x < cx {
			in = !in
		}
	}
	return in
}

// Cells are about twice as tall as wide. Slopes are judged as seen, so a
// circle needs a box about twice as wide as it is tall.
const aspect = 2.0

// Slope bands, in degrees as seen on screen: flatter than flat draws a
// horizontal rune, steeper than steep a vertical one, between them a
// diagonal.
const (
	flat  = 22.5
	steep = 67.5
)

// slope picks the outline rune for a direction of travel (dx, dy) in cells.
//
//	─  flatter than 22.5°    ╲  down-right or up-left
//	│  steeper than 67.5°    ╱  up-right or down-left
func slope(dx, dy float64, st doc.Style) rune {
	t := table(st)
	deg := math.Atan2(math.Abs(dy*aspect), math.Abs(dx)) * 180 / math.Pi
	switch {
	case deg < flat:
		return t[E|W]
	case deg > steep:
		return t[N|S]
	case st == doc.ASCII && (dx > 0) == (dy > 0):
		return '\\'
	case st == doc.ASCII:
		return '/'
	case (dx > 0) == (dy > 0):
		return '╲'
	}
	return '╱'
}

// ellipse outlines the ellipse that fits the box as a closed stair-step
// path of edge masks, so it takes the border style and joins other shapes:
//
//	  ╭─────╮
//	╭─╯     ╰─╮
//	│         │
//	╰─╮     ╭─╯
//	  ╰─────╯
//
// It walks the curve in small angle steps, keeps each new cell, and splits
// diagonal steps through the neighbour nearer the curve.
func (g *Grid) ellipse(s doc.Shape, z int) {
	// Light ovals take rounded corners; square ones look like boxes.
	if s.Style == doc.Light {
		s.Style = doc.Rounded
	}
	path := EllipsePath(s.Box)
	n := len(path)
	if n < 2 {
		for _, p := range path {
			g.edge(p, E|W, s.Style, z)
		}
		return
	}
	for i := range n {
		a, b := path[i], path[(i+1)%n]
		if a == b {
			continue
		}
		d := dir(a, b)
		g.edge(a, d, s.Style, z)
		g.edge(b, opposite(d), s.Style, z)
	}
}

// EllipsePath returns the closed, 4-connected cell loop of the ellipse that
// fits box.
func EllipsePath(box doc.Box) []doc.Point {
	cx := float64(box.Min.X+box.Max.X) / 2
	cy := float64(box.Min.Y+box.Max.Y) / 2
	a := float64(box.Max.X-box.Min.X) / 2
	r := float64(box.Max.Y-box.Min.Y) / 2

	// reach is how far out a cell sits, in normalized units: 1 on the curve.
	reach := func(p doc.Point) float64 {
		nx := (float64(p.X) - cx) / math.Max(a, 0.5)
		ny := (float64(p.Y) - cy) / math.Max(r, 0.5)
		return nx*nx + ny*ny
	}
	// outer picks the corner cell further out, so stairs turn outward.
	outer := func(c1, c2 doc.Point) doc.Point {
		if reach(c2) > reach(c1) {
			return c2
		}
		return c1
	}

	steps := int(8*(a+r)) + 8
	var out []doc.Point
	add := func(p doc.Point) {
		if len(out) == 0 {
			out = append(out, p)
			return
		}
		last := out[len(out)-1]
		if p == last {
			return
		}
		if p.X != last.X && p.Y != last.Y {
			// Diagonal step: go through the outer corner cell.
			out = append(out, outer(doc.Point{X: p.X, Y: last.Y}, doc.Point{X: last.X, Y: p.Y}))
		}
		out = append(out, p)
	}

	for i := range steps {
		t := 2 * math.Pi * float64(i) / float64(steps)
		add(doc.Point{
			X: int(math.Round(cx + a*math.Cos(t))),
			Y: int(math.Round(cy + r*math.Sin(t))),
		})
	}

	// Close the loop: drop a repeated start, then bridge back to it.
	for len(out) > 1 && out[len(out)-1] == out[0] {
		out = out[:len(out)-1]
	}
	if n := len(out); n > 1 {
		last, first := out[n-1], out[0]
		if last.X != first.X && last.Y != first.Y {
			out = append(out, outer(doc.Point{X: first.X, Y: last.Y}, doc.Point{X: last.X, Y: first.Y}))
		}
	}
	return unspike(out)
}

// unspike removes out-and-back steps (a, b, a) from a closed loop; they
// would draw a stub sticking out of the outline.
func unspike(loop []doc.Point) []doc.Point {
	for changed := true; changed && len(loop) > 2; {
		changed = false
		n := len(loop)
		for i := range n {
			if loop[(i+n-1)%n] == loop[(i+1)%n] {
				// Drop the tip and one copy of the base.
				lo, hi := i, (i+1)%n
				if hi < lo {
					lo, hi = hi, lo
				}
				loop = append(loop[:hi], loop[hi+1:]...)
				loop = append(loop[:lo], loop[lo+1:]...)
				changed = true
				break
			}
		}
	}
	return loop
}

// polygon joins its vertices in order, and the last to the first when
// closed. Straight sides use edge masks so they join boxes; slanted sides
// are solid diagonal runes.
func (g *Grid) polygon(s doc.Shape, z int) {
	n := len(s.Points)
	sides := n - 1
	if s.Closed && n >= 3 {
		sides = n
	}

	for i := range sides {
		a, b := s.Points[i], s.Points[(i+1)%n]
		if a == b {
			continue
		}
		if a.X == b.X || a.Y == b.Y {
			path := Path(doc.Shape{Points: []doc.Point{a, b}})
			for j := 0; j < len(path)-1; j++ {
				d := dir(path[j], path[j+1])
				g.edge(path[j], d, s.Style, z)
				g.edge(path[j+1], opposite(d), s.Style, z)
			}
			continue
		}
		r := slope(float64(b.X-a.X), float64(b.Y-a.Y), s.Style)
		for _, p := range Bresenham(a, b) {
			g.solid(p, r, z)
		}
	}

	// Shape vertices where two slanted sides meet, so points look pointed:
	//
	//	 ╱╲   top       ╲╱   bottom       <  left     >  right
	for i := range n {
		if !s.Closed && (i == 0 || i == n-1) {
			continue
		}
		v := s.Points[i]
		u1 := s.Points[(i+n-1)%n]
		u2 := s.Points[(i+1)%n]
		g.vertex(v, u1, u2, s.Style, z)
	}
}

// vertex draws the point where sides v-u1 and v-u2 meet, when both slant.
func (g *Grid) vertex(v, u1, u2 doc.Point, st doc.Style, z int) {
	ax, ay := sign(u1.X-v.X), sign(u1.Y-v.Y)
	bx, by := sign(u2.X-v.X), sign(u2.Y-v.Y)
	if ax == 0 || ay == 0 || bx == 0 || by == 0 {
		return
	}

	up, down := '╱', '╲'
	if st == doc.ASCII {
		up, down = '/', '\\'
	}
	right := doc.Point{X: v.X + 1, Y: v.Y}
	switch {
	case ay > 0 && by > 0 && ax != bx: // both sides go down: top point
		g.solid(v, up, z)
		g.solid(right, down, z)
	case ay < 0 && by < 0 && ax != bx: // both go up: bottom point
		g.solid(v, down, z)
		g.solid(right, up, z)
	case ax > 0 && bx > 0 && ay != by: // both go right: left point
		g.solid(v, '<', z)
	case ax < 0 && bx < 0 && ay != by: // both go left: right point
		g.solid(v, '>', z)
	}
}

// Bresenham returns the cells on a straight line from a to b, both included.
func Bresenham(a, b doc.Point) []doc.Point {
	dx, dy := abs(b.X-a.X), -abs(b.Y-a.Y)
	sx, sy := 1, 1
	if a.X > b.X {
		sx = -1
	}
	if a.Y > b.Y {
		sy = -1
	}

	out := []doc.Point{a}
	e := dx + dy
	for a != b {
		e2 := 2 * e
		if e2 >= dy {
			e += dy
			a.X += sx
		}
		if e2 <= dx {
			e += dx
			a.Y += sy
		}
		out = append(out, a)
	}
	return out
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
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
