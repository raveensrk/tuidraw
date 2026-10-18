package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"tuidraw/internal/doc"
	"tuidraw/internal/render"
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
	case RectTool:
		m.drag = &drag{kind: dragDraw, start: p, last: p,
			preview: doc.Shape{Kind: doc.Rect, Box: doc.Span(p, p), Style: m.style}}
	case LineTool:
		m.drag = &drag{kind: dragDraw, start: p, last: p,
			preview: doc.Shape{Kind: doc.Line, Points: []doc.Point{p, p}, Style: m.style, Arrow: m.arrow}}
	case TextTool:
		m.commitEdit()
		if i := m.hit(p); i >= 0 && m.shapes[i].Kind == doc.Text {
			m.startEdit(i)
			return
		}
		m.edit = &edit{index: -1, shape: doc.Shape{Kind: doc.Text, Points: []doc.Point{p}}}
		m.status = "typing - esc to finish"
	case Brush:
		m.drag = &drag{kind: dragDraw, start: p, last: p,
			preview: doc.Shape{Kind: doc.Stroke, Points: []doc.Point{p}, Char: m.brush}}
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
	}
}

func (m *Model) drawMove(d *drag, p doc.Point) {
	s := &d.preview
	switch s.Kind {
	case doc.Rect:
		s.Box = doc.Span(d.start, p)
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
		for _, q := range bresenham(prev, p)[1:] {
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
		switch {
		case s.Kind == doc.Rect && d.start == p:
			// A plain click makes a default-size box.
			s.Box = doc.Box{Min: p, Max: doc.Point{X: p.X + clickRectW - 1, Y: p.Y + clickRectH - 1}}
		case s.Kind == doc.Rect && (s.Box.Min.X == s.Box.Max.X || s.Box.Min.Y == s.Box.Max.Y):
			m.status = "rect needs at least 2x2 cells"
			return
		case s.Kind == doc.Line && d.start == p:
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
		if inside(m.shapes[i], p) {
			return i
		}
	}
	return -1
}

func inside(s doc.Shape, p doc.Point) bool {
	switch s.Kind {
	case doc.Rect, doc.Text:
		return s.Bounds().Has(p)
	case doc.Group:
		for _, c := range s.Children {
			if inside(c, p) {
				return true
			}
		}
	}
	return false
}

// handles returns the resize handles of a lone selected rect or line:
// rect corners clockwise from top-left, or line start and end.
func (m *Model) handles() []doc.Point {
	if len(m.sel) != 1 {
		return nil
	}
	s := m.shapes[m.sel[0]]
	switch s.Kind {
	case doc.Rect:
		b := s.Box
		return []doc.Point{b.Min, {X: b.Max.X, Y: b.Min.Y}, b.Max, {X: b.Min.X, Y: b.Max.Y}}
	case doc.Line:
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
	if s.Kind == doc.Line {
		s.Points[h] = p
		return
	}

	// Rect: the opposite corner stays put. Corners stay at least 2x2.
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

// bresenham returns the cells on a straight line from a to b, both included.
func bresenham(a, b doc.Point) []doc.Point {
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
