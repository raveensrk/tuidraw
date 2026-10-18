package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuidraw/internal/doc"
	"tuidraw/internal/render"
)

// Dark theme colors.
var (
	accent  = lipgloss.Color("#5fafff")
	warm    = lipgloss.Color("#ffaf5f")
	dim     = lipgloss.Color("#6c6c6c")
	bar     = lipgloss.Color("#262626")
	ink     = lipgloss.Color("#d0d0d0")
	black   = lipgloss.Color("#000000")
	cursorc = lipgloss.Color("#4e4e4e")
)

// class is how a canvas cell is painted.
type class int

const (
	plain class = iota
	picked
	handle
	preview
	marquee
	cursor
	caret
)

var styles = map[class]lipgloss.Style{
	plain:   lipgloss.NewStyle().Foreground(ink),
	picked:  lipgloss.NewStyle().Foreground(accent),
	handle:  lipgloss.NewStyle().Foreground(black).Background(accent),
	preview: lipgloss.NewStyle().Foreground(warm),
	marquee: lipgloss.NewStyle().Foreground(accent),
	cursor:  lipgloss.NewStyle().Foreground(ink).Background(cursorc),
	caret:   lipgloss.NewStyle().Foreground(black).Background(ink),
}

var (
	barStyle    = lipgloss.NewStyle().Foreground(ink).Background(bar)
	activeStyle = lipgloss.NewStyle().Foreground(black).Background(accent).Bold(true)
)

func (m *Model) View() tea.View {
	var body string
	switch {
	case m.w == 0:
		body = ""
	case m.help != nil:
		body = m.help.view()
	default:
		body = m.toolbar() + "\n" + m.canvas() + "\n" + m.statusLine()
	}
	v := tea.NewView(body)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

// zone is a clickable toolbar span.
type zone struct {
	from, to int
	tool     Tool
}

// toolLabel is a tool's toolbar text: its key, then its name.
func toolLabel(key, label string) string { return " " + key + " " + label + " " }

func (m *Model) toolbarZones() []zone {
	var out []zone
	x := 0
	for _, t := range tools {
		w := len(toolLabel(t.key, t.label))
		out = append(out, zone{x, x + w, t.tool})
		x += w
	}
	return out
}

func (m *Model) toolbar() string {
	var sb strings.Builder
	for _, t := range tools {
		label := toolLabel(t.key, t.label)
		if t.tool == m.tool {
			sb.WriteString(activeStyle.Render(label))
			continue
		}
		sb.WriteString(barStyle.Render(label))
	}
	return pad(sb.String(), m.w, barStyle)
}

func (m *Model) statusLine() string {
	left := m.status
	if m.prompt != nil {
		left = m.prompt.label + string(m.prompt.value) + "█"
	}

	name := "untitled"
	if m.path != "" {
		name = filepath.Base(m.path)
	}
	if m.dirty {
		name += "*"
	}
	arrow := "off"
	if m.arrow {
		arrow = "on"
	}
	right := fmt.Sprintf("%s  arrow %s  brush %s  │  %d,%d", m.style, arrow, m.brush, m.cursor.X, m.cursor.Y)
	if len(m.sel) > 0 {
		right += fmt.Sprintf("  %d selected", len(m.sel))
	}
	right += "  " + name + "  ? help "

	space := m.w - lipgloss.Width(right) - 1
	if lipgloss.Width(left) > space {
		left = string([]rune(left)[:max(space, 0)])
	}
	gap := strings.Repeat(" ", max(space-lipgloss.Width(left), 0))
	return pad(barStyle.Render(" "+left+gap+right), m.w, barStyle)
}

// canvas paints the visible window of the world, then overlays selection,
// handles, the marquee, the caret and the pointer.
func (m *Model) canvas() string {
	ch := m.canvasH()
	box := doc.Box{Min: m.view, Max: doc.Point{X: m.view.X + m.w - 1, Y: m.view.Y + ch - 1}}

	// What to draw: the document, with the shape being edited or drawn.
	shapes := m.shapes
	var extra *doc.Shape
	if m.edit != nil {
		shapes = append([]doc.Shape(nil), m.shapes...)
		if m.edit.index >= 0 {
			shapes[m.edit.index] = m.edit.shape
		}
		if m.edit.index < 0 {
			shapes = append(shapes, m.edit.shape)
		}
		extra = &m.edit.shape
	}
	if m.poly != nil {
		shapes = append(append([]doc.Shape(nil), shapes...), *m.poly)
		extra = m.poly
	}
	if m.drag != nil && m.drag.kind == dragDraw {
		shapes = append(append([]doc.Shape(nil), shapes...), m.drag.preview)
		extra = &m.drag.preview
	}

	grid := render.Draw(shapes, box)
	sel := render.Draw(m.selShapes(), box)
	var pre *render.Grid
	if extra != nil {
		pre = render.Draw([]doc.Shape{*extra}, box)
	}

	marks := map[doc.Point]class{}
	for _, h := range m.handles() {
		marks[h] = handle
	}
	if m.edit != nil {
		marks[m.edit.caretCell()] = caret
	}
	if m.edit == nil {
		marks[m.cursor] = cursor
	}

	var ring doc.Box
	ringOn := m.drag != nil && m.drag.kind == dragMarquee
	if ringOn {
		ring = doc.Span(m.drag.start, m.drag.last)
	}

	rows := make([]string, 0, ch)
	for y := box.Min.Y; y <= box.Max.Y; y++ {
		var line strings.Builder
		var run []rune
		cur := plain
		flush := func() {
			if len(run) > 0 {
				line.WriteString(styles[cur].Render(string(run)))
				run = run[:0]
			}
		}

		for x := box.Min.X; x <= box.Max.X; x++ {
			p := doc.Point{X: x, Y: y}
			r := grid.Rune(p)
			c := plain
			switch {
			case pre != nil && pre.Rune(p) != ' ':
				c = preview
			case sel.Rune(p) != ' ':
				c = picked
			}
			if ringOn && onRing(ring, p) {
				c = marquee
				if r == ' ' {
					r = '·'
				}
			}
			if k, ok := marks[p]; ok {
				c = k
			}
			if c != cur {
				flush()
				cur = c
			}
			run = append(run, r)
		}
		flush()
		rows = append(rows, line.String())
	}
	return strings.Join(rows, "\n")
}

func onRing(b doc.Box, p doc.Point) bool {
	if !b.Has(p) {
		return false
	}
	return p.X == b.Min.X || p.X == b.Max.X || p.Y == b.Min.Y || p.Y == b.Max.Y
}

// pad fills s with styled spaces up to width w, or cuts it to w.
func pad(s string, w int, st lipgloss.Style) string {
	n := w - lipgloss.Width(s)
	if n < 0 {
		return ansi.Truncate(s, w, "")
	}
	return s + st.Render(strings.Repeat(" ", n))
}
