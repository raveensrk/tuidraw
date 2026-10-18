// Package ui is the Bubble Tea editor: an infinite canvas between a toolbar
// row and a status row.
//
//	row 0      toolbar   tools, style, arrow, brush
//	rows 1..   canvas    world cells, viewport at m.view
//	last row   status    message, cursor coords, file
//
// Mouse and keyboard drive the same pointer: a click is down/up at the mouse
// cell, Enter is down/up at the keyboard cursor.
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"tuidraw/internal/doc"
	"tuidraw/internal/export"
	"tuidraw/internal/history"
)

// Tool is the active drawing tool.
type Tool int

const (
	Select Tool = iota
	RectTool
	OvalTool
	CircleTool
	TriTool
	DiamondTool
	PolyTool
	LineTool
	TextTool
	Brush
	Eraser
)

// tools lists each tool with its key and toolbar label.
var tools = []struct {
	tool  Tool
	key   string
	label string
}{
	{Select, "v", "sel"},
	{RectTool, "r", "rect"},
	{OvalTool, "o", "oval"},
	{CircleTool, "O", "circle"},
	{TriTool, "y", "tri"},
	{DiamondTool, "d", "diamond"},
	{PolyTool, "p", "poly"},
	{LineTool, "l", "line"},
	{TextTool, "t", "text"},
	{Brush, "b", "brush"},
	{Eraser, "e", "erase"},
}

const (
	defaultName  = "drawing.tuidraw"
	doubleClick  = 400 * time.Millisecond
	nudgeStep    = 1
	wheelStep    = 3
	dupOffsetX   = 2
	dupOffsetY   = 1
	clickRectW   = 12 // size of a rect made by a click with no drag
	clickRectH   = 5
	defaultBrush = "█"
)

// dragKind says what an ongoing pointer drag does.
type dragKind int

const (
	dragDraw dragKind = iota
	dragMove
	dragHandle
	dragMarquee
	dragPan
	dragErase
)

type drag struct {
	kind    dragKind
	start   doc.Point // world cell where it began
	last    doc.Point
	screen  doc.Point // screen cell, for panning
	view    doc.Point // view at start, for panning
	handle  int
	pushed  bool      // history recorded for this drag
	preview doc.Shape // shape being drawn
	decided bool      // line: leg order chosen from first move

	// make builds the shape of a box tool from start and pointer.
	make func(a, b doc.Point) doc.Shape
}

// edit is an in-progress text edit. index is -1 for a new text shape.
type edit struct {
	index int
	shape doc.Shape
	caret int // rune offset into shape.Text
}

// prompt is a one-line input in the status row.
type prompt struct {
	label string
	value []rune
	done  func(m *Model, value string) tea.Cmd
}

// Model is the editor state.
type Model struct {
	shapes []doc.Shape
	sel    []int // selected shape indices, ascending
	hist   history.History
	path   string
	dirty  bool

	w, h   int
	view   doc.Point // world cell at canvas top-left
	cursor doc.Point // world cell of the pointer

	tool  Tool
	style doc.Style
	arrow bool
	brush string

	drag      *drag
	edit      *edit
	poly      *doc.Shape // polygon being placed; last point follows the pointer
	prompt    *prompt
	help      *pager
	clip      []doc.Shape
	status    string
	quitArmed bool
	charArmed bool   // next key sets the brush character
	palette   string // "ink" or "paper" while picking a color, else ""

	ink   doc.Color // defaults for new shapes
	paper doc.Color
	fill  bool

	clickAt  time.Time
	clickPos doc.Point
}

// New makes an editor for path. A missing file starts an empty drawing
// that saves to that path.
func New(path string) (*Model, error) {
	m := &Model{path: path, style: doc.Light, brush: defaultBrush, arrow: true}
	if path == "" {
		m.status = "new drawing - ? for help"
		return m, nil
	}

	d, err := doc.Load(path)
	if os.IsNotExist(err) {
		m.status = "new file " + path
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	m.shapes = d.Shapes
	m.status = fmt.Sprintf("opened %s (%d shapes)", path, len(d.Shapes))
	return m, nil
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		if m.help != nil {
			m.help.size(m.w, m.h)
		}
		return m, nil

	case tea.KeyPressMsg:
		return m, m.key(msg)

	case tea.PasteMsg:
		m.paste(msg.Content)
		return m, nil

	case tea.MouseMsg:
		if m.help == nil && m.prompt == nil {
			m.mouse(msg)
		}
		return m, nil
	}
	return m, nil
}

// canvasH is the number of canvas rows between toolbar and status.
func (m *Model) canvasH() int { return max(m.h-2, 1) }

// world maps a screen cell to a world cell.
func (m *Model) world(x, y int) doc.Point {
	return doc.Point{X: m.view.X + x, Y: m.view.Y + y - 1}
}

// follow scrolls the view so the cursor stays on screen.
func (m *Model) follow() {
	ch := m.canvasH()
	if m.cursor.X < m.view.X {
		m.view.X = m.cursor.X
	}
	if m.cursor.X >= m.view.X+m.w {
		m.view.X = m.cursor.X - m.w + 1
	}
	if m.cursor.Y < m.view.Y {
		m.view.Y = m.cursor.Y
	}
	if m.cursor.Y >= m.view.Y+ch {
		m.view.Y = m.cursor.Y - ch + 1
	}
}

// push records the current state for undo and marks the file dirty.
func (m *Model) push() {
	m.hist.Push(m.shapes)
	m.dirty = true
}

func (m *Model) selected(i int) bool {
	_, ok := slices.BinarySearch(m.sel, i)
	return ok
}

func (m *Model) setSel(idx ...int) {
	m.sel = slices.Clone(idx)
	slices.Sort(m.sel)
	m.sel = slices.Compact(m.sel)
}

func (m *Model) toggleSel(i int) {
	if pos, ok := slices.BinarySearch(m.sel, i); ok {
		m.sel = slices.Delete(m.sel, pos, pos+1)
		return
	}
	m.setSel(append(m.sel, i)...)
}

func (m *Model) selShapes() []doc.Shape {
	var out []doc.Shape
	for _, i := range m.sel {
		out = append(out, m.shapes[i])
	}
	return out
}

// key handles every key press, routed by mode: help, prompt, text edit,
// brush character pick, then the canvas.
func (m *Model) key(k tea.KeyPressMsg) tea.Cmd {
	s := k.String()

	switch {
	case m.help != nil:
		if m.help.key(k) {
			m.help = nil
		}
		return nil
	case m.prompt != nil:
		return m.promptKey(k)
	case m.edit != nil:
		m.editKey(k)
		return nil
	case m.palette != "":
		m.paletteKey(s)
		return nil
	case m.charArmed:
		m.charArmed = false
		if k.Text != "" && s != "esc" {
			m.brush = string([]rune(k.Text)[:1])
			m.status = "brush " + m.brush
		}
		return nil
	}

	if s != "q" {
		m.quitArmed = false
	}

	// Tool keys.
	for _, t := range tools {
		if s == t.key {
			m.setTool(t.tool)
			return nil
		}
	}

	switch s {
	case "q", "ctrl+q":
		if m.dirty && !m.quitArmed && s == "q" {
			m.quitArmed = true
			m.status = "unsaved changes - q again to quit, ctrl+s to save"
			return nil
		}
		return tea.Quit
	case "?":
		m.help = newPager(helpText, m.w, m.h)
	case "esc":
		if m.poly != nil {
			m.commitPoly(false)
			return nil
		}
		if m.drag != nil {
			m.cancelDrag()
			return nil
		}
		m.sel = nil
		m.status = ""

	// Keyboard cursor: arrows (hjkl would clash with the l line tool).
	// Enter or space presses, then releases, the pointer.
	case "left":
		m.moveCursor(-1, 0)
	case "down":
		m.moveCursor(0, 1)
	case "up":
		m.moveCursor(0, -1)
	case "right":
		m.moveCursor(1, 0)
	case "enter", "space":
		if m.drag == nil {
			m.down(m.cursor, false)
			return nil
		}
		m.up(m.cursor)

	// Nudge the selection.
	case "H", "shift+left":
		m.nudge(-nudgeStep, 0)
	case "J", "shift+down":
		m.nudge(0, nudgeStep)
	case "K", "shift+up":
		m.nudge(0, -nudgeStep)
	case "L", "shift+right":
		m.nudge(nudgeStep, 0)

	case "0":
		m.view = doc.Point{}
		m.cursor = doc.Point{X: m.w / 2, Y: m.canvasH() / 2}
	case "1", "2", "3", "4", "5":
		m.setStyle(doc.Styles[s[0]-'1'])
	case "a":
		m.toggleArrow()
	case "f":
		m.flip()
	case "i":
		m.palette = "ink"
	case "I":
		m.palette = "paper"
	case "F":
		m.toggleFill()
	case "c":
		m.charArmed = true
		m.status = "press a key for the brush character"

	case "x", "delete", "backspace":
		m.remove()
	case "ctrl+a":
		m.setSel()
		for i := range m.shapes {
			m.sel = append(m.sel, i)
		}
	case "ctrl+z", "u":
		m.undo()
	case "ctrl+r", "ctrl+y", "ctrl+shift+z", "U":
		m.redo()
	case "ctrl+c":
		return m.copy()
	case "ctrl+x":
		cmd := m.copy()
		m.remove()
		return cmd
	case "ctrl+v":
		m.pasteShapes()
	case "ctrl+d":
		m.duplicate()
	case "ctrl+g":
		m.group()
	case "]":
		m.raise(false)
	case "[":
		m.lower(false)
	case "}":
		m.raise(true)
	case "{":
		m.lower(true)

	case "ctrl+s":
		return m.save(false)
	case "ctrl+e":
		return m.save(true)
	}
	return nil
}

func (m *Model) setTool(t Tool) {
	m.commitEdit()
	m.commitPoly(false)
	m.drag = nil
	m.tool = t
	m.status = tools[t].label
}

func (m *Model) moveCursor(dx, dy int) {
	m.cursor.X += dx
	m.cursor.Y += dy
	m.follow()
	m.track(m.cursor)
	if m.drag != nil {
		m.move(m.cursor)
	}
}

func (m *Model) nudge(dx, dy int) {
	if len(m.sel) == 0 {
		return
	}
	m.push()
	for _, i := range m.sel {
		m.shapes[i].Move(dx, dy)
	}
}

// dress gives a new shape the current ink, background and fill.
func (m *Model) dress(s *doc.Shape) {
	s.FG, s.BG = m.ink, m.paper
	s.Fill = m.fill && s.Fillable()
}

func (m *Model) dressed(s doc.Shape) doc.Shape {
	m.dress(&s)
	return s
}

// paletteKeys maps palette keys to colors: 0 is the terminal default,
// 1-8 the normal colors, a-h the bright ones.
const paletteKeys = "12345678abcdefgh"

// pickColor sets the ink or paper color being picked, for new shapes and
// the selection.
func (m *Model) pickColor(c doc.Color) {
	which := m.palette
	m.palette = ""
	name := string(c)
	if name == "" {
		name = "default"
	}
	m.status = which + " " + name

	if which == "ink" {
		m.ink = c
	}
	if which == "paper" {
		m.paper = c
	}
	if len(m.sel) == 0 {
		return
	}
	m.push()
	for _, i := range m.sel {
		if which == "ink" {
			m.shapes[i].Paint(&c, nil, nil)
			continue
		}
		m.shapes[i].Paint(nil, &c, nil)
	}
}

func (m *Model) paletteKey(s string) {
	if s == "0" {
		m.pickColor("")
		return
	}
	if i := strings.Index(paletteKeys, s); len(s) == 1 && i >= 0 {
		m.pickColor(doc.Colors[i])
		return
	}
	m.palette = ""
	m.status = "cancelled"
}

// toggleFill flips fill on the selection, or the default when none.
func (m *Model) toggleFill() {
	if len(m.sel) == 0 {
		m.fill = !m.fill
		m.status = fmt.Sprintf("fill %v", m.fill)
		return
	}
	m.push()
	on := !m.shapes[m.sel[0]].Fill
	for _, i := range m.sel {
		m.shapes[i].Paint(nil, nil, &on)
	}
	m.status = fmt.Sprintf("fill %v", on)
}

// setStyle sets the default border style and restyles the selection.
func (m *Model) setStyle(st doc.Style) {
	m.style = st
	m.status = "style " + string(st)
	if len(m.sel) == 0 {
		return
	}
	m.push()
	for _, i := range m.sel {
		m.shapes[i].SetStyle(st)
	}
}

// toggleArrow flips arrowheads on selected lines, or the default when none.
func (m *Model) toggleArrow() {
	lines := m.selLines()
	if len(lines) == 0 {
		m.arrow = !m.arrow
		m.status = fmt.Sprintf("arrow %v", m.arrow)
		return
	}
	m.push()
	for _, i := range lines {
		m.shapes[i].Arrow = !m.shapes[i].Arrow
	}
}

// flip swaps which leg of selected elbow lines comes first.
func (m *Model) flip() {
	lines := m.selLines()
	if len(lines) == 0 {
		return
	}
	m.push()
	for _, i := range lines {
		m.shapes[i].Flip = !m.shapes[i].Flip
	}
}

func (m *Model) selLines() []int {
	var out []int
	for _, i := range m.sel {
		if m.shapes[i].Kind == doc.Line {
			out = append(out, i)
		}
	}
	return out
}

func (m *Model) remove() {
	if len(m.sel) == 0 {
		return
	}
	m.push()
	for _, i := range slices.Backward(m.sel) {
		m.shapes = slices.Delete(m.shapes, i, i+1)
	}
	m.status = fmt.Sprintf("deleted %d", len(m.sel))
	m.sel = nil
}

func (m *Model) undo() {
	shapes, ok := m.hist.Undo(m.shapes)
	if !ok {
		m.status = "nothing to undo"
		return
	}
	m.shapes, m.sel, m.dirty = shapes, nil, true
	m.status = "undo"
}

func (m *Model) redo() {
	shapes, ok := m.hist.Redo(m.shapes)
	if !ok {
		m.status = "nothing to redo"
		return
	}
	m.shapes, m.sel, m.dirty = shapes, nil, true
	m.status = "redo"
}

// copy puts the selection on the internal clipboard and its text export on
// the system clipboard: pbcopy when present, OSC 52 otherwise.
func (m *Model) copy() tea.Cmd {
	if len(m.sel) == 0 {
		return nil
	}
	m.clip = doc.Clones(m.selShapes())
	text := export.Text(m.clip)
	m.status = fmt.Sprintf("copied %d", len(m.clip))

	if bin, err := exec.LookPath("pbcopy"); err == nil {
		cmd := exec.Command(bin)
		cmd.Stdin = strings.NewReader(text)
		if cmd.Run() == nil {
			return nil
		}
	}
	return tea.SetClipboard(text)
}

// pasteShapes drops the internal clipboard with its top-left at the cursor.
func (m *Model) pasteShapes() {
	if len(m.clip) == 0 {
		return
	}
	b := m.clip[0].Bounds()
	for _, s := range m.clip[1:] {
		b = b.Union(s.Bounds())
	}
	m.insert(doc.Clones(m.clip), m.cursor.X-b.Min.X, m.cursor.Y-b.Min.Y)
}

// paste handles bracketed paste from the terminal. Text matching the
// internal clipboard pastes shapes; anything else becomes a text shape.
func (m *Model) paste(text string) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if m.edit != nil {
		m.editInsert(text)
		return
	}
	if len(m.clip) > 0 && strings.TrimRight(text, "\n") == strings.TrimRight(export.Text(m.clip), "\n") {
		m.pasteShapes()
		return
	}
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return
	}
	m.insert([]doc.Shape{m.dressed(doc.Shape{Kind: doc.Text, Points: []doc.Point{m.cursor}, Text: text})}, 0, 0)
}

func (m *Model) duplicate() {
	if len(m.sel) == 0 {
		return
	}
	m.insert(doc.Clones(m.selShapes()), dupOffsetX, dupOffsetY)
}

// insert adds shapes on top, moved by dx, dy, and selects them.
func (m *Model) insert(list []doc.Shape, dx, dy int) {
	m.push()
	start := len(m.shapes)
	for i := range list {
		list[i].Move(dx, dy)
	}
	m.shapes = append(m.shapes, list...)
	m.sel = nil
	for i := start; i < len(m.shapes); i++ {
		m.sel = append(m.sel, i)
	}
	m.status = fmt.Sprintf("added %d", len(list))
}

// group wraps two or more selected shapes in a group at the topmost one's
// place, or ungroups a single selected group.
func (m *Model) group() {
	if len(m.sel) == 1 && m.shapes[m.sel[0]].Kind == doc.Group {
		m.push()
		i := m.sel[0]
		kids := m.shapes[i].Children
		m.shapes = slices.Replace(m.shapes, i, i+1, kids...)
		m.sel = nil
		for j := range kids {
			m.sel = append(m.sel, i+j)
		}
		m.status = "ungrouped"
		return
	}
	if len(m.sel) < 2 {
		return
	}

	m.push()
	g := doc.Shape{Kind: doc.Group, Children: m.selShapes()}
	top := m.sel[len(m.sel)-1]
	for _, i := range slices.Backward(m.sel) {
		m.shapes = slices.Delete(m.shapes, i, i+1)
	}
	at := top - len(m.sel) + 1
	m.shapes = slices.Insert(m.shapes, at, g)
	m.setSel(at)
	m.status = "grouped"
}

// raise moves the selection up one step, or to the top when all is set.
func (m *Model) raise(all bool) {
	if len(m.sel) == 0 {
		return
	}
	m.push()
	if all {
		m.reorder(false)
		return
	}
	for _, i := range slices.Backward(m.sel) {
		if i+1 < len(m.shapes) && !m.selected(i+1) {
			m.shapes[i], m.shapes[i+1] = m.shapes[i+1], m.shapes[i]
			m.sel[slices.Index(m.sel, i)] = i + 1
		}
	}
}

// lower moves the selection down one step, or to the bottom when all is set.
func (m *Model) lower(all bool) {
	if len(m.sel) == 0 {
		return
	}
	m.push()
	if all {
		m.reorder(true)
		return
	}
	for _, i := range m.sel {
		if i > 0 && !m.selected(i-1) {
			m.shapes[i], m.shapes[i-1] = m.shapes[i-1], m.shapes[i]
			m.sel[slices.Index(m.sel, i)] = i - 1
		}
	}
}

// reorder moves the selection to the bottom (first) or top, keeping order.
func (m *Model) reorder(first bool) {
	var picked, rest []doc.Shape
	for i, s := range m.shapes {
		if m.selected(i) {
			picked = append(picked, s)
			continue
		}
		rest = append(rest, s)
	}
	start := len(rest)
	m.shapes = append(rest, picked...)
	if first {
		start = 0
		m.shapes = append(picked, rest...)
	}
	m.sel = nil
	for i := range picked {
		m.sel = append(m.sel, start+i)
	}
}

// save writes the project, or the .txt export next to it. With no path yet
// it asks for one first.
func (m *Model) save(txt bool) tea.Cmd {
	if m.path == "" {
		m.prompt = &prompt{
			label: "save as: ",
			value: []rune(defaultName),
			done: func(m *Model, v string) tea.Cmd {
				if !strings.HasSuffix(v, ".tuidraw") {
					v += ".tuidraw"
				}
				m.path = v
				return m.save(txt)
			},
		}
		return nil
	}

	if txt {
		base := strings.TrimSuffix(m.path, filepath.Ext(m.path))
		if err := export.Save(base+".txt", m.shapes); err != nil {
			m.status = "export failed: " + err.Error()
			return nil
		}
		if err := export.SaveANSI(base+".ans", m.shapes); err != nil {
			m.status = "export failed: " + err.Error()
			return nil
		}
		m.status = "exported " + base + ".txt and .ans"
		return nil
	}

	if err := doc.Save(m.path, doc.Doc{Shapes: m.shapes}); err != nil {
		m.status = "save failed: " + err.Error()
		return nil
	}
	m.dirty = false
	m.status = "saved " + m.path
	return nil
}

func (m *Model) promptKey(k tea.KeyPressMsg) tea.Cmd {
	p := m.prompt
	switch k.String() {
	case "esc":
		m.prompt = nil
		m.status = "cancelled"
	case "enter":
		m.prompt = nil
		v := strings.TrimSpace(string(p.value))
		if v == "" {
			return nil
		}
		return p.done(m, v)
	case "backspace":
		if len(p.value) > 0 {
			p.value = p.value[:len(p.value)-1]
		}
	case "ctrl+u":
		p.value = nil
	default:
		if k.Text != "" {
			p.value = append(p.value, []rune(k.Text)...)
		}
	}
	return nil
}
