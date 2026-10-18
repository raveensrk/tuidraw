package ui

import (
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/raveensrk/tuidraw/internal/doc"
	"github.com/raveensrk/tuidraw/internal/render"
)

// mouse routes mouse events: toolbar clicks, wheel and right or middle drag
// pan, left button drives the pointer.
func (m *Model) mouse(msg tea.MouseMsg) {
	ev := msg.Mouse()
	p := m.world(ev.X, ev.Y)

	switch msg.(type) {
	case tea.MouseWheelMsg:
		m.wheel(ev)
		return

	case tea.MouseClickMsg:
		if m.palette != "" {
			if ev.Y == m.h-1 {
				m.paletteClick(ev.X)
				return
			}
			m.palette = ""
		}
		if ev.Y == 0 {
			m.toolbarClick(ev.X)
			return
		}
		if ev.Y > m.canvasH() {
			return
		}
		m.cursor = p
		if ev.Button == tea.MouseRight || ev.Button == tea.MouseMiddle {
			m.drag = &drag{kind: dragPan, screen: doc.Point{X: ev.X, Y: ev.Y}, view: m.view}
			return
		}
		if ev.Button == tea.MouseLeft {
			m.down(p, ev.Mod&(tea.ModShift|tea.ModCtrl|tea.ModAlt) != 0)
		}

	case tea.MouseMotionMsg:
		if m.drag != nil && m.drag.kind == dragPan {
			m.view.X = m.drag.view.X - (ev.X - m.drag.screen.X)
			m.view.Y = m.drag.view.Y - (ev.Y - m.drag.screen.Y)
			return
		}
		m.cursor = p
		m.track(p)
		if m.drag != nil {
			m.move(p)
		}

	case tea.MouseReleaseMsg:
		if m.drag != nil && m.drag.kind == dragPan {
			m.drag = nil
			return
		}
		m.cursor = p
		if m.drag != nil {
			m.up(p)
		}
	}
}

func (m *Model) wheel(ev tea.Mouse) {
	dx, dy := 0, 0
	switch ev.Button {
	case tea.MouseWheelUp:
		dy = -wheelStep
	case tea.MouseWheelDown:
		dy = wheelStep
	case tea.MouseWheelLeft:
		dx = -wheelStep
	case tea.MouseWheelRight:
		dx = wheelStep
	}
	// Shift turns vertical wheel into horizontal scroll.
	if ev.Mod&tea.ModShift != 0 {
		dx, dy = dy, dx
	}
	m.view.X += dx
	m.view.Y += dy
}

// down presses the pointer at p. toggle is a modifier click (add or remove
// from the selection).
func (m *Model) down(p doc.Point, toggle bool) {
	double := time.Since(m.clickAt) < doubleClick && m.clickPos == p
	m.clickAt, m.clickPos = time.Now(), p
	m.quitArmed = false

	switch m.tool {
	case Select:
		m.selectDown(p, toggle, double)
	case RectTool, OvalTool, CircleTool, TriTool, DiamondTool:
		base := m.maker()
		mk := func(a, b doc.Point) doc.Shape {
			s := base(a, b)
			m.dress(&s)
			return s
		}
		m.drag = &drag{kind: dragDraw, start: p, last: p, make: mk, preview: mk(p, p)}
	case PolyTool:
		m.polyDown(p, double)
	case LineTool:
		m.drag = &drag{kind: dragDraw, start: p, last: p,
			preview: m.dressed(doc.Shape{Kind: doc.Line, Points: []doc.Point{p, p}, Style: m.style, Arrow: m.arrow})}
	case TextTool:
		m.commitEdit()
		if i := m.hit(p); i >= 0 && m.shapes[i].Kind == doc.Text {
			m.startEdit(i)
			return
		}
		m.edit = &edit{index: -1, shape: m.dressed(doc.Shape{Kind: doc.Text, Points: []doc.Point{p}})}
		m.status = "typing - esc to finish"
	case Brush:
		m.drag = &drag{kind: dragDraw, start: p, last: p,
			preview: m.dressed(doc.Shape{Kind: doc.Stroke, Points: []doc.Point{p}, Char: m.brush})}
	case Eraser:
		m.drag = &drag{kind: dragErase, start: p, last: p}
		m.erase(p)
	}
}

func (m *Model) selectDown(p doc.Point, toggle, double bool) {
	m.commitEdit()

	// A handle of the single selected shape: resize.
	if h := m.handleAt(p); h >= 0 {
		m.drag = &drag{kind: dragHandle, start: p, last: p, handle: h}
		return
	}

	i := m.hit(p)
	if i < 0 {
		if !toggle {
			m.sel = nil
		}
		m.drag = &drag{kind: dragMarquee, start: p, last: p}
		return
	}

	if toggle {
		m.toggleSel(i)
		return
	}
	if double && m.shapes[i].Kind == doc.Text {
		m.startEdit(i)
		return
	}
	if !m.selected(i) {
		m.setSel(i)
	}
	m.drag = &drag{kind: dragMove, start: p, last: p}
}

// move drags the pointer to p.
func (m *Model) move(p doc.Point) {
	d := m.drag
	if p == d.last {
		return
	}
	dx, dy := p.X-d.last.X, p.Y-d.last.Y
	d.last = p

	switch d.kind {
	case dragDraw:
		m.drawMove(d, p)
	case dragMove:
		if !d.pushed {
			m.push()
			d.pushed = true
		}
		for _, i := range m.sel {
			m.shapes[i].Move(dx, dy)
		}
	case dragHandle:
		if !d.pushed {
			m.push()
			d.pushed = true
		}
		m.resize(d.handle, p)
	case dragErase:
		for _, q := range render.Bresenham(doc.Point{X: p.X - dx, Y: p.Y - dy}, p)[1:] {
			m.erase(q)
		}
	}
}

// erase removes what the eraser touches at p: the brush cells under it, or
// the whole of any other shape drawn there. One drag is one undo step.
func (m *Model) erase(p doc.Point) {
	d := m.drag
	for i := len(m.shapes) - 1; i >= 0; i-- {
		s := &m.shapes[i]
		if !render.Hit(*s, p) {
			continue
		}
		if !d.pushed {
			m.push()
			d.pushed = true
		}
		m.sel = nil

		if s.Kind != doc.Stroke {
			m.shapes = slices.Delete(m.shapes, i, i+1)
			return
		}
		s.Points = slices.DeleteFunc(s.Points, func(q doc.Point) bool { return q == p })
		if len(s.Points) == 0 {
			m.shapes = slices.Delete(m.shapes, i, i+1)
		}
		return
	}
}

func (m *Model) drawMove(d *drag, p doc.Point) {
	s := &d.preview
	if d.make != nil {
		*s = d.make(d.start, p)
		return
	}
	switch s.Kind {
	case doc.Line:
		// The first move decides the elbow: start along the way you moved.
		if !d.decided {
			dx, dy := abs(p.X-d.start.X), abs(p.Y-d.start.Y)
			s.Flip = dy > dx
			d.decided = dx != dy
		}
		s.Points[1] = p
	case doc.Stroke:
		prev := s.Points[len(s.Points)-1]
		for _, q := range render.Bresenham(prev, p)[1:] {
			s.Points = append(s.Points, q)
		}
	}
}

// up releases the pointer at p and commits whatever the drag made.
func (m *Model) up(p doc.Point) {
	d := m.drag
	m.drag = nil
	if d == nil {
		return
	}
	if p != d.last {
		m.drag = d
		m.move(p)
		m.drag = nil
	}

	switch d.kind {
	case dragMarquee:
		box := doc.Span(d.start, p)
		for i, s := range m.shapes {
			if s.Bounds().Overlaps(box) && !m.selected(i) {
				m.setSel(append(m.sel, i)...)
			}
		}
	case dragDraw:
		s := d.preview
		if d.make != nil && d.start == p {
			// A plain click makes a default-size shape.
			s = d.make(p, doc.Point{X: p.X + clickRectW - 1, Y: p.Y + clickRectH - 1})
		}
		if b := s.Bounds(); d.make != nil && (b.Min.X == b.Max.X || b.Min.Y == b.Max.Y) {
			m.status = "shape needs at least 2x2 cells"
			return
		}
		if s.Kind == doc.Line && d.start == p {
			return
		}
		m.push()
		m.shapes = append(m.shapes, s)
		m.setSel(len(m.shapes) - 1)
	}
}

// cancelDrag drops a drag in progress, undoing any edit it made.
func (m *Model) cancelDrag() {
	d := m.drag
	m.drag = nil
	if d.pushed {
		m.shapes, _ = m.hist.Pop()
	}
	m.status = "cancelled"
}

// hit returns the topmost shape at p, or -1. Drawn cells win over the
// empty inside of a rect, so a big box never hides what sits in it.
func (m *Model) hit(p doc.Point) int {
	for i := len(m.shapes) - 1; i >= 0; i-- {
		if render.Hit(m.shapes[i], p) {
			return i
		}
	}
	for i := len(m.shapes) - 1; i >= 0; i-- {
		if render.Inside(m.shapes[i], p) {
			return i
		}
	}
	return -1
}

// handles returns the resize handles of a lone selected shape: box
// corners clockwise from top-left for rects and ellipses, line ends, or
// polygon vertices.
func (m *Model) handles() []doc.Point {
	if len(m.sel) != 1 {
		return nil
	}
	s := m.shapes[m.sel[0]]
	switch s.Kind {
	case doc.Rect, doc.Ellipse:
		b := s.Box
		return []doc.Point{b.Min, {X: b.Max.X, Y: b.Min.Y}, b.Max, {X: b.Min.X, Y: b.Max.Y}}
	case doc.Line, doc.Polygon:
		return s.Points
	}
	return nil
}

func (m *Model) handleAt(p doc.Point) int {
	for i, h := range m.handles() {
		if h == p {
			return i
		}
	}
	return -1
}

// resize moves handle h of the lone selected shape to p.
func (m *Model) resize(h int, p doc.Point) {
	s := &m.shapes[m.sel[0]]
	if s.Kind == doc.Line || s.Kind == doc.Polygon {
		s.Points[h] = p
		return
	}

	// Rect or ellipse: the opposite corner stays put. Corners stay at least 2x2.
	far := m.handles()[(h+2)%4]
	if p.X == far.X || p.Y == far.Y {
		return
	}
	s.Box = doc.Span(far, p)

	// Corners were renumbered by Span; follow the one under the pointer.
	m.drag.handle = m.handleAt(p)
}

// Text editing.

func (m *Model) startEdit(i int) {
	m.edit = &edit{index: i, shape: m.shapes[i].Clone(), caret: len([]rune(m.shapes[i].Text))}
	m.setSel(i)
	m.status = "editing - esc to finish"
}

// commitEdit writes the text being edited back as a shape. Empty text
// deletes it.
func (m *Model) commitEdit() {
	e := m.edit
	if e == nil {
		return
	}
	m.edit = nil
	m.status = ""
	empty := strings.TrimSpace(e.shape.Text) == ""

	switch {
	case e.index < 0 && empty:
		return
	case e.index < 0:
		m.push()
		m.shapes = append(m.shapes, e.shape)
		m.setSel(len(m.shapes) - 1)
	case empty:
		m.push()
		m.shapes = append(m.shapes[:e.index], m.shapes[e.index+1:]...)
		m.sel = nil
	case e.shape.Text != m.shapes[e.index].Text:
		m.push()
		m.shapes[e.index] = e.shape
	}
}

func (m *Model) editKey(k tea.KeyPressMsg) {
	e := m.edit
	text := []rune(e.shape.Text)

	switch k.String() {
	case "esc", "ctrl+s":
		m.commitEdit()
	case "enter":
		m.editInsert("\n")
	case "backspace":
		if e.caret > 0 {
			e.shape.Text = string(append(text[:e.caret-1:e.caret-1], text[e.caret:]...))
			e.caret--
		}
	case "delete":
		if e.caret < len(text) {
			e.shape.Text = string(append(text[:e.caret:e.caret], text[e.caret+1:]...))
		}
	case "left":
		e.caret = max(e.caret-1, 0)
	case "right":
		e.caret = min(e.caret+1, len(text))
	case "home", "ctrl+a":
		for e.caret > 0 && text[e.caret-1] != '\n' {
			e.caret--
		}
	case "end", "ctrl+e":
		for e.caret < len(text) && text[e.caret] != '\n' {
			e.caret++
		}
	default:
		if k.Text != "" {
			m.editInsert(k.Text)
		}
	}
}

func (m *Model) editInsert(s string) {
	e := m.edit
	text := []rune(e.shape.Text)
	in := []rune(s)
	out := append(append(append([]rune{}, text[:e.caret]...), in...), text[e.caret:]...)
	e.shape.Text = string(out)
	e.caret += len(in)
}

// caretCell is the world cell of the text caret.
func (e *edit) caretCell() doc.Point {
	before := []rune(e.shape.Text)[:e.caret]
	lines := strings.Split(string(before), "\n")
	p := e.shape.Points[0]
	return doc.Point{X: p.X + len([]rune(lines[len(lines)-1])), Y: p.Y + len(lines) - 1}
}

// Toolbar.

func (m *Model) toolbarClick(x int) {
	for _, z := range m.toolbarZones() {
		if x >= z.from && x < z.to {
			m.setTool(z.tool)
			return
		}
	}
}

// maker returns how the active box tool builds its shape from the drag
// start a and the pointer b.
func (m *Model) maker() func(a, b doc.Point) doc.Shape {
	st := m.style
	switch m.tool {
	case OvalTool:
		return func(a, b doc.Point) doc.Shape {
			return doc.Shape{Kind: doc.Ellipse, Box: doc.Span(a, b), Style: st}
		}
	case CircleTool:
		return func(a, b doc.Point) doc.Shape {
			return doc.Shape{Kind: doc.Ellipse, Box: circleBox(a, b), Style: st}
		}
	case TriTool:
		return func(a, b doc.Point) doc.Shape {
			x := doc.Span(a, b)
			mid := (x.Min.X + x.Max.X) / 2
			return doc.Shape{Kind: doc.Polygon, Closed: true, Style: st, Points: []doc.Point{
				{X: mid, Y: x.Min.Y}, x.Max, {X: x.Min.X, Y: x.Max.Y},
			}}
		}
	case DiamondTool:
		return func(a, b doc.Point) doc.Shape {
			x := doc.Span(a, b)
			mx, my := (x.Min.X+x.Max.X)/2, (x.Min.Y+x.Max.Y)/2
			return doc.Shape{Kind: doc.Polygon, Closed: true, Style: st, Points: []doc.Point{
				{X: mx, Y: x.Min.Y}, {X: x.Max.X, Y: my}, {X: mx, Y: x.Max.Y}, {X: x.Min.X, Y: my},
			}}
		}
	}
	return func(a, b doc.Point) doc.Shape {
		return doc.Shape{Kind: doc.Rect, Box: doc.Span(a, b), Style: st}
	}
}

// circleBox grows from a toward b into a box that looks round: cells are
// about twice as tall as wide, so it is 2h-1 wide for h rows. The drag's
// larger side sets the size.
func circleBox(a, b doc.Point) doc.Box {
	h := max(abs(b.Y-a.Y)+1, (abs(b.X-a.X)+2)/2)
	w := 2*h - 1
	sx, sy := 1, 1
	if b.X < a.X {
		sx = -1
	}
	if b.Y < a.Y {
		sy = -1
	}
	return doc.Span(a, doc.Point{X: a.X + sx*(w-1), Y: a.Y + sy*(h-1)})
}

// Polygon placing: each click adds a vertex. Clicking the first vertex
// closes it; a double-click or esc finishes it open.

func (m *Model) polyDown(p doc.Point, double bool) {
	if m.poly == nil {
		poly := m.dressed(doc.Shape{Kind: doc.Polygon, Style: m.style, Points: []doc.Point{p, p}})
		m.poly = &poly
		m.status = "click to add points - click the first to close, double-click or esc to finish"
		return
	}

	pts := m.poly.Points
	switch {
	case len(pts) >= 4 && p == pts[0]:
		m.commitPoly(true)
	case double:
		m.commitPoly(false)
	default:
		m.poly.Points = append(pts[:len(pts)-1], p, p)
	}
}

// track moves the rubber-band end of a polygon being placed.
func (m *Model) track(p doc.Point) {
	if m.poly != nil {
		m.poly.Points[len(m.poly.Points)-1] = p
	}
}

// commitPoly adds the polygon being placed, dropping its rubber-band end
// and repeated points. It needs 2 points open or 3 closed.
func (m *Model) commitPoly(closed bool) {
	s := m.poly
	if s == nil {
		return
	}
	m.poly = nil
	m.status = ""

	pts := s.Points[:len(s.Points)-1]
	var out []doc.Point
	for _, q := range pts {
		if len(out) == 0 || out[len(out)-1] != q {
			out = append(out, q)
		}
	}
	if closed && len(out) > 1 && out[len(out)-1] == out[0] {
		out = out[:len(out)-1]
	}
	if len(out) < 2 || (closed && len(out) < 3) {
		return
	}

	s.Points, s.Closed = out, closed
	m.push()
	m.shapes = append(m.shapes, *s)
	m.setSel(len(m.shapes) - 1)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
