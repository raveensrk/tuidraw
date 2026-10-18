package ui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"tuidraw/internal/doc"
	"tuidraw/internal/export"
)

func setup(t *testing.T) *Model {
	t.Helper()
	m, err := New(filepath.Join(t.TempDir(), "t.tuidraw"))
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return m
}

func press(m *Model, keys ...string) {
	for _, k := range keys {
		var key tea.Key
		switch k {
		case "enter":
			key = tea.Key{Code: tea.KeyEnter}
		case "esc":
			key = tea.Key{Code: tea.KeyEscape}
		case "right":
			key = tea.Key{Code: tea.KeyRight}
		case "down":
			key = tea.Key{Code: tea.KeyDown}
		default:
			if rest, ok := strings.CutPrefix(k, "ctrl+"); ok {
				key = tea.Key{Code: rune(rest[0]), Mod: tea.ModCtrl}
				break
			}
			r := []rune(k)[0]
			key = tea.Key{Code: r, Text: k}
		}
		m.Update(tea.KeyPressMsg(key))
	}
}

// dragTo drags the left button between two screen cells.
func dragTo(m *Model, x0, y0, x1, y1 int) {
	m.Update(tea.MouseClickMsg{X: x0, Y: y0, Button: tea.MouseLeft})
	m.Update(tea.MouseMotionMsg{X: x1, Y: y1, Button: tea.MouseLeft})
	m.Update(tea.MouseReleaseMsg{X: x1, Y: y1, Button: tea.MouseLeft})
}

func TestDrawAndJoin(t *testing.T) {
	m := setup(t)

	// Screen row 1 is world row 0.
	press(m, "r")
	dragTo(m, 0, 1, 4, 3)
	press(m, "l", "a") // arrowheads default on; off for a plain divider
	dragTo(m, 0, 2, 4, 2)
	press(m, "t")
	m.Update(tea.MouseClickMsg{X: 1, Y: 1, Button: tea.MouseLeft})
	press(m, "O", "K", "esc")

	want := "┌OK─┐\n├───┤\n└───┘\n"
	if got := export.Text(m.shapes); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestMoveUndoGroup(t *testing.T) {
	m := setup(t)
	press(m, "r")
	dragTo(m, 0, 1, 3, 3)
	dragTo(m, 10, 1, 13, 3)

	// Select both with a marquee, group, move by dragging a border.
	press(m, "v")
	dragTo(m, 0, 5, 20, 1)
	if len(m.sel) != 2 {
		t.Fatalf("marquee selected %d, want 2", len(m.sel))
	}
	press(m, "ctrl+g")
	if len(m.shapes) != 1 || m.shapes[0].Kind != doc.Group {
		t.Fatalf("group failed: %+v", m.shapes)
	}
	dragTo(m, 0, 1, 2, 2)
	if b := m.shapes[0].Bounds(); b.Min != (doc.Point{X: 2, Y: 1}) {
		t.Errorf("moved to %v", b.Min)
	}

	press(m, "u")
	if b := m.shapes[0].Bounds(); b.Min != (doc.Point{}) {
		t.Errorf("undo left group at %v", b.Min)
	}
	// Undo clears the selection; click the group again to ungroup it.
	m.Update(tea.MouseClickMsg{X: 0, Y: 1, Button: tea.MouseLeft})
	m.Update(tea.MouseReleaseMsg{X: 0, Y: 1, Button: tea.MouseLeft})
	press(m, "ctrl+g")
	if len(m.shapes) != 2 {
		t.Errorf("ungroup gave %d shapes", len(m.shapes))
	}
}

func TestKeyboardPointerAndSave(t *testing.T) {
	m := setup(t)
	m.cursor = doc.Point{}

	// Draw a rect with the keyboard only.
	press(m, "r", "enter", "right", "right", "right", "down", "down", "enter")
	if len(m.shapes) != 1 || m.shapes[0].Box != doc.Span(doc.Point{}, doc.Point{X: 3, Y: 2}) {
		t.Fatalf("keyboard rect: %+v", m.shapes)
	}

	press(m, "3", "ctrl+s")
	d, err := doc.Load(m.path)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Shapes) != 1 || d.Shapes[0].Style != doc.Double || m.dirty {
		t.Errorf("saved %+v dirty=%v", d.Shapes, m.dirty)
	}
}

func TestResizeHandle(t *testing.T) {
	m := setup(t)
	press(m, "r")
	dragTo(m, 0, 1, 3, 3)

	// The new rect is selected; drag its bottom-right corner out.
	press(m, "v")
	dragTo(m, 3, 3, 6, 5)
	if b := m.shapes[0].Box; b.Max != (doc.Point{X: 6, Y: 4}) {
		t.Errorf("resized to %v", b)
	}
}

func TestViewFits(t *testing.T) {
	m := setup(t)
	v := m.View().Content
	if n := strings.Count(v, "\n") + 1; n != 24 {
		t.Errorf("view has %d rows, want 24", n)
	}
	press(m, "?")
	if !strings.Contains(m.View().Content, "TOOLS") {
		t.Error("help page missing")
	}
	press(m, "/", "a", "r", "r", "o", "w", "enter")
	if !strings.Contains(m.View().Content, "match 1 of") {
		t.Error("help search found nothing")
	}
}
