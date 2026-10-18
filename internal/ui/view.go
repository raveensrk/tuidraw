package ui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/raveensrk/tuidraw/internal/doc"
	"github.com/raveensrk/tuidraw/internal/render"
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
	selTint = lipgloss.Color("#1c3a5e")
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
	picked:  lipgloss.NewStyle().Foreground(accent).Background(selTint),
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
func toolLabel(key, label string) string { return " " + key + " " + label }

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
	if m.palette != "" {
		return m.paletteLine()
	}

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
	fill := ""
	if m.fill {
		fill = " filled"
	}

	// Right side: style, colors as swatches, position, file.
	parts := []string{
		barStyle.Render(fmt.Sprintf("%s  arrow %s  brush %s  ink ", m.style, arrow, m.brush)),
		swatch(m.ink),
		barStyle.Render(" bg "),
		swatch(m.paper),
		barStyle.Render(fmt.Sprintf("%s  │  %d,%d", fill, m.cursor.X, m.cursor.Y)),
	}
	tail := ""
	if len(m.sel) > 0 {
		tail += fmt.Sprintf("  %d selected", len(m.sel))
	}
	parts = append(parts, barStyle.Render(tail+"  "+name+"  ? help "))
	right := strings.Join(parts, "")

	space := m.w - lipgloss.Width(right) - 1
	if lipgloss.Width(left) > space {
		left = string([]rune(left)[:max(space, 0)])
	}
	gap := strings.Repeat(" ", max(space-lipgloss.Width(left), 0))
	return pad(barStyle.Render(" "+left+gap)+right, m.w, barStyle)
}

// termColor maps a palette color to the terminal's own ANSI color, so the
// editor shows exactly what the .ans export will.
func termColor(c doc.Color) color.Color {
	return lipgloss.Color(strconv.Itoa(c.Index()))
}

// swatch shows a color as two cells, or a dotted box for the default.
func swatch(c doc.Color) string {
	if c == "" {
		return barStyle.Render("··")
	}
	return lipgloss.NewStyle().Foreground(termColor(c)).Background(bar).Render("██")
}

// paletteStart is where the first swatch begins in the palette line.
func (m *Model) paletteStart() int { return len(" " + m.palette + ":  0 none ") }

// paletteLine replaces the status row while a color is being picked.
func (m *Model) paletteLine() string {
	var sb strings.Builder
	sb.WriteString(barStyle.Render(" " + m.palette + ":  0 none "))
	for i, c := range doc.Colors {
		sb.WriteString(barStyle.Render(" " + string(paletteKeys[i])))
		sb.WriteString(swatch(c))
	}
	sb.WriteString(barStyle.Render("   key or click, esc cancels"))
	return pad(sb.String(), m.w, barStyle)
}

// paletteClick picks the color under screen column x of the palette line.
func (m *Model) paletteClick(x int) {
	const stride = 4 // each entry is " k██": space, key, two swatch cells
	start := m.paletteStart()
	if x < start {
		if x >= start-len("0 none ") {
			m.pickColor("")
		}
		return
	}
	if i := (x - start) / stride; i < len(doc.Colors) {
		m.pickColor(doc.Colors[i])
	}
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

	// Runs of cells with the same look render as one styled string.
	type look struct {
		class  class
		fg, bg doc.Color
	}
	cache := map[look]lipgloss.Style{}
	styleOf := func(k look) lipgloss.Style {
		if st, ok := cache[k]; ok {
			return st
		}
		st := styles[k.class]
		if k.fg != "" && k.class != handle && k.class != caret && k.class != cursor {
			st = st.Foreground(termColor(k.fg))
		}
		if k.bg != "" {
			st = st.Background(termColor(k.bg))
		}
		if k.class == picked {
			st = st.Background(selTint)
		}
		cache[k] = st
		return st
	}

	rows := make([]string, 0, ch)
	for y := box.Min.Y; y <= box.Max.Y; y++ {
		var line strings.Builder
		var run []rune
		cur := look{}
		flush := func() {
			if len(run) > 0 {
				line.WriteString(styleOf(cur).Render(string(run)))
				run = run[:0]
			}
		}

		for x := box.Min.X; x <= box.Max.X; x++ {
			p := doc.Point{X: x, Y: y}
			r, fg, bg := grid.Cell(p)
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
			if k := (look{c, fg, bg}); k != cur {
				flush()
				cur = k
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
