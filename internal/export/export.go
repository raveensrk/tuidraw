// Package export writes a drawing as plain text.
package export

import (
	"os"
	"strings"

	"tuidraw/internal/doc"
	"tuidraw/internal/render"
)

// Text renders shapes cropped to their bounds, with trailing spaces trimmed.
// An empty list gives an empty string.
func Text(shapes []doc.Shape) string {
	if len(shapes) == 0 {
		return ""
	}
	b := shapes[0].Bounds()
	for _, s := range shapes[1:] {
		b = b.Union(s.Bounds())
	}

	lines := render.Draw(shapes, b).Lines()
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n") + "\n"
}

// Save writes the text export to path.
func Save(path string, shapes []doc.Shape) error {
	return os.WriteFile(path, []byte(Text(shapes)), 0o644)
}
