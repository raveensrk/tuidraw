package history

import (
	"testing"

	"github.com/raveensrk/tuidraw/internal/doc"
)

func shapes(n int) []doc.Shape {
	out := make([]doc.Shape, n)
	for i := range out {
		out[i] = doc.Shape{Kind: doc.Text, Points: []doc.Point{{X: i}}, Text: "x"}
	}
	return out
}

func TestUndoRedo(t *testing.T) {
	var h History
	cur := shapes(0)

	h.Push(cur)
	cur = shapes(1)
	h.Push(cur)
	cur = shapes(2)

	cur, _ = h.Undo(cur)
	if len(cur) != 1 {
		t.Fatalf("undo gave %d shapes, want 1", len(cur))
	}
	cur, _ = h.Undo(cur)
	if len(cur) != 0 {
		t.Fatalf("second undo gave %d, want 0", len(cur))
	}
	if _, ok := h.Undo(cur); ok {
		t.Error("undo past the start should fail")
	}

	cur, _ = h.Redo(cur)
	cur, _ = h.Redo(cur)
	if len(cur) != 2 {
		t.Fatalf("redo gave %d, want 2", len(cur))
	}

	// A new edit clears redo.
	cur, _ = h.Undo(cur)
	h.Push(cur)
	if _, ok := h.Redo(cur); ok {
		t.Error("redo after a new edit should fail")
	}
}

func TestSnapshotsAreCopies(t *testing.T) {
	var h History
	cur := shapes(1)
	h.Push(cur)
	cur[0].Move(5, 5)

	prev, _ := h.Undo(cur)
	if prev[0].Points[0].X != 0 {
		t.Error("snapshot changed with the live document")
	}
}
