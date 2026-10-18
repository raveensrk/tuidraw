package export

import (
	"testing"

	"github.com/raveensrk/tuidraw/internal/doc"
)

func TestText(t *testing.T) {
	shapes := []doc.Shape{
		{Kind: doc.Rect, Box: doc.Span(doc.Point{X: 10, Y: 5}, doc.Point{X: 17, Y: 7}), Style: doc.Rounded},
		{Kind: doc.Text, Points: []doc.Point{{X: 12, Y: 6}}, Text: "OK"},
		{Kind: doc.Line, Points: []doc.Point{{X: 10, Y: 9}, {X: 13, Y: 9}}, Style: doc.Light, Arrow: true},
	}
	want := "╭──────╮\n│ OK   │\n╰──────╯\n\n───▶\n"
	if got := Text(shapes); got != want {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
	if Text(nil) != "" {
		t.Error("empty drawing should export empty")
	}
}

func TestANSI(t *testing.T) {
	shapes := []doc.Shape{
		{Kind: doc.Text, Points: []doc.Point{{X: 0, Y: 0}}, Text: "ab", FG: "red"},
		{Kind: doc.Text, Points: []doc.Point{{X: 2, Y: 0}}, Text: "c", FG: "bright_blue", BG: "yellow"},
		{Kind: doc.Text, Points: []doc.Point{{X: 0, Y: 1}}, Text: "d"},
	}
	want := "\x1b[31;49mab\x1b[94;43mc\x1b[0m\nd\n"
	if got := ANSI(shapes); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}

	// No colors at all: same as the text export.
	plain := shapes[2:]
	if ANSI(plain) != Text(plain) {
		t.Errorf("uncolored ANSI %q differs from text %q", ANSI(plain), Text(plain))
	}
}
