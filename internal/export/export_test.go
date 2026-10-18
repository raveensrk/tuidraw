package export

import (
	"testing"

	"tuidraw/internal/doc"
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
