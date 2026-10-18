// Package history keeps undo and redo as whole-document snapshots.
// Drawings are small, so a deep copy per edit is cheap and can never drift
// out of sync the way inverse operations can.
package history

import "github.com/raveensrk/tuidraw/internal/doc"

// Limit is how many undo steps are kept.
const Limit = 200

// History holds past and future snapshots around the current document.
type History struct {
	past   [][]doc.Shape
	future [][]doc.Shape
}

// Push records the state before an edit and clears redo.
func (h *History) Push(shapes []doc.Shape) {
	h.past = append(h.past, doc.Clones(shapes))
	if len(h.past) > Limit {
		h.past = h.past[1:]
	}
	h.future = nil
}

// Undo returns the previous state, given the current one, and false when
// there is nothing to undo.
func (h *History) Undo(cur []doc.Shape) ([]doc.Shape, bool) {
	if len(h.past) == 0 {
		return cur, false
	}
	prev := h.past[len(h.past)-1]
	h.past = h.past[:len(h.past)-1]
	h.future = append(h.future, doc.Clones(cur))
	return prev, true
}

// Redo returns the next state, given the current one, and false when there
// is nothing to redo.
func (h *History) Redo(cur []doc.Shape) ([]doc.Shape, bool) {
	if len(h.future) == 0 {
		return cur, false
	}
	next := h.future[len(h.future)-1]
	h.future = h.future[:len(h.future)-1]
	h.past = append(h.past, doc.Clones(cur))
	return next, true
}

// Pop drops the last snapshot and returns it, for cancelling an edit
// without leaving a redo step behind.
func (h *History) Pop() ([]doc.Shape, bool) {
	if len(h.past) == 0 {
		return nil, false
	}
	prev := h.past[len(h.past)-1]
	h.past = h.past[:len(h.past)-1]
	return prev, true
}
