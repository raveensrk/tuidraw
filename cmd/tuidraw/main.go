// Command tuidraw is an infinite-canvas editor for designing terminal UIs.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"tuidraw/internal/ui"
)

const short = `usage: tuidraw [file.tuidraw]

Design terminal UIs on an infinite character canvas: boxes, lines, text,
brush strokes. Opens file, or a new drawing when it does not exist.

options:
  -h, --help   short help
  help         long help

example:
  tuidraw login_screen.tuidraw`

const long = short + `

In the editor press ? for every key binding.

files:
  <name>.tuidraw   project, JSON, written by ctrl+s
  <name>.txt       plain text export, written by ctrl+e

environment: none read.

exit codes:
  0   quit normally, or help shown
  1   file could not be opened, or the terminal failed
  2   bad arguments`

func main() {
	args := os.Args[1:]
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			fmt.Println(short)
			return
		case "help":
			fmt.Println(long)
			return
		}
	}
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, short)
		os.Exit(2)
	}

	path := ""
	if len(args) == 1 {
		path = args[0]
	}
	m, err := ui.New(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tuidraw:", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tuidraw:", err)
		os.Exit(1)
	}
}
