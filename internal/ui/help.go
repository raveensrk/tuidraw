package ui

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const helpText = `tuidraw - design terminal UIs on an infinite character canvas

Draw boxes, ovals, circles, polygons, lines, text and brush strokes on a
grid where one cell is one terminal character. Shapes stay editable: select, move, resize, restyle.
Borders that touch or cross join up (┼ ├ ┬). Drawings save as .tuidraw
(JSON) next to where you ran it, and export as plain .txt.

TOOLS
  v              select: click, drag to move, drag empty space to marquee
  r              rect: drag a box; a click makes a 12x5 box
  o              oval: drag its bounding box
  O              circle: drag; kept about twice as wide as tall, which
                 looks round because cells are tall
  y              triangle: drag its bounding box
  d              diamond: drag its bounding box
  p              polygon: click each corner; click the first corner to
                 close it, double-click or esc to leave it open
  l              line: drag; elbow follows the way you first move
  t              text: click to type, click a text to edit it
  b              brush: drag to paint the brush character
  click toolbar  pick a tool with the mouse

POINTER
  mouse          left button draws or selects
  shift/ctrl/alt+click   add or remove a shape from the selection
  double-click   edit a text shape (select tool)
  right/middle drag      pan the canvas
  wheel          scroll; shift+wheel scrolls sideways
  arrows         move the keyboard cursor (view follows)
  enter, space   press the pointer at the cursor; press again to release
  0              back to the origin

SELECTION
  esc            cancel a drag, else clear the selection
  ctrl+a         select all
  shift+arrows, H J K L   nudge the selection one cell
  drag a handle  resize a lone rect or oval by a corner; move a line end
                 or a polygon corner
  x, del, bksp   delete
  ctrl+c         copy (also to the system clipboard as text)
  ctrl+x         cut
  ctrl+v         paste at the cursor
  ctrl+d         duplicate
  ctrl+g         group the selection, or ungroup a lone group
  ] [            bring forward, send backward
  } {            bring to front, send to back

STYLE
  1 2 3 4 5      border style: light heavy double rounded ascii
                 applies to the selection and to new shapes
  a              toggle arrowhead on selected lines, else the default
  f              flip which leg of a selected elbow line comes first
  c then a key   set the brush character

TEXT EDITING
  type           insert at the caret
  enter          new line
  left right     move the caret
  home end       start, end of line
  bksp del       delete
  esc            finish (empty text is removed)

FILE
  ctrl+s         save (asks for a name the first time)
  ctrl+e         export plain text to <name>.txt
  u, ctrl+z      undo
  U, ctrl+r, ctrl+y   redo
  q              quit (twice if unsaved), ctrl+q quits at once

COLORS AND SYMBOLS
  blue           selected shapes; blue blocks are resize handles
  orange         shape being drawn or typed
  ╱ ╲ < >        slanted polygon sides and points; straight sides join
                 boxes like any border
  grey block     keyboard and mouse cursor cell
  · dotted ring  marquee selection

PROMPTS
  The status row asks for a file name on first save. Type, enter accepts,
  esc cancels, ctrl+u clears.

THIS PAGE
  j k, down up   one line         space f, b   one page
  d, u           half a page      g, G         top, bottom
  /              search; enter jumps to the first match below
  n, N           next, previous match (wraps)
  esc            clear search, then leave       q   leave`

var (
	matchStyle   = lipgloss.NewStyle().Foreground(black).Background(dim)
	currentStyle = lipgloss.NewStyle().Foreground(black).Background(warm)
)

// pager is a less-like viewer for the help page.
type pager struct {
	lines     []string
	top       int
	w, h      int
	searching bool
	input     []rune
	re        *regexp.Regexp
	hits      []int // line numbers with a match
	cur       int   // index into hits, -1 for none
}

func newPager(text string, w, h int) *pager {
	return &pager{lines: strings.Split(text, "\n"), w: w, h: h, cur: -1}
}

func (p *pager) size(w, h int) { p.w, p.h = w, h }

func (p *pager) rows() int { return max(p.h-1, 1) }

func (p *pager) scroll(n int) {
	p.top = max(min(p.top+n, len(p.lines)-p.rows()), 0)
}

// key handles a key and reports whether to leave the page.
func (p *pager) key(k tea.KeyPressMsg) bool {
	s := k.String()
	if p.searching {
		switch s {
		case "esc":
			p.searching = false
			p.input = nil
		case "enter":
			p.searching = false
			p.find(string(p.input))
		case "backspace":
			if len(p.input) > 0 {
				p.input = p.input[:len(p.input)-1]
			}
		default:
			if k.Text != "" {
				p.input = append(p.input, []rune(k.Text)...)
			}
		}
		return false
	}

	page := p.rows()
	switch s {
	case "q":
		return true
	case "esc":
		if p.re != nil {
			p.re, p.hits, p.cur = nil, nil, -1
			return false
		}
		return true
	case "j", "down":
		p.scroll(1)
	case "k", "up":
		p.scroll(-1)
	case "space", "f", "pgdown":
		p.scroll(page)
	case "b", "pgup":
		p.scroll(-page)
	case "d":
		p.scroll(page / 2)
	case "u":
		p.scroll(-page / 2)
	case "g", "home":
		p.top = 0
	case "G", "end":
		p.scroll(len(p.lines))
	case "/":
		p.searching = true
		p.input = nil
	case "n":
		p.step(1)
	case "N":
		p.step(-1)
	}
	return false
}

// find compiles the pattern (smartcase) and jumps to the first match at or
// below the top line.
func (p *pager) find(pat string) {
	p.re, p.hits, p.cur = nil, nil, -1
	if pat == "" {
		return
	}
	expr := regexp.QuoteMeta(pat)
	if !strings.ContainsFunc(pat, unicode.IsUpper) {
		expr = "(?i)" + expr
	}
	p.re = regexp.MustCompile(expr)
	for i, l := range p.lines {
		if p.re.MatchString(l) {
			p.hits = append(p.hits, i)
		}
	}
	if len(p.hits) == 0 {
		return
	}
	p.cur = 0
	for i, h := range p.hits {
		if h >= p.top {
			p.cur = i
			break
		}
	}
	p.show()
}

func (p *pager) step(d int) {
	if len(p.hits) == 0 {
		return
	}
	p.cur = (p.cur + d + len(p.hits)) % len(p.hits)
	p.show()
}

// show scrolls so the current match is on screen.
func (p *pager) show() {
	line := p.hits[p.cur]
	if line < p.top || line >= p.top+p.rows() {
		p.top = 0
		p.scroll(line)
	}
}

func (p *pager) view() string {
	var sb strings.Builder
	for i := p.top; i < p.top+p.rows(); i++ {
		if i < len(p.lines) {
			sb.WriteString(p.paint(i))
		}
		sb.WriteString("\n")
	}

	foot := " help - q to leave, / to search"
	switch {
	case p.searching:
		foot = " /" + string(p.input) + "█"
	case p.re != nil && len(p.hits) == 0:
		foot = " no match"
	case p.re != nil:
		foot = fmt.Sprintf(" match %d of %d", p.cur+1, len(p.hits))
	}
	sb.WriteString(pad(barStyle.Render(foot), p.w, barStyle))
	return sb.String()
}

// paint highlights search matches on line i; the current match stronger.
func (p *pager) paint(i int) string {
	l := p.lines[i]
	if p.re == nil {
		return l
	}
	st := matchStyle
	if p.cur >= 0 && p.hits[p.cur] == i {
		st = currentStyle
	}
	return p.re.ReplaceAllStringFunc(l, func(s string) string { return st.Render(s) })
}
