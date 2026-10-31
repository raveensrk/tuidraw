#!/bin/sh
# build.sh - render diagrams, check and export the tuidraw book.
#
# usage: docs/build.sh diagrams|check|export
#
# commands:
#   diagrams   render every source in diagrams/ to SVG
#   check      markdownlint, offline link and anchor check, full re-render
#   export     write _build/tuidraw.pdf and _build/tuidraw.epub
#   -h, --help short help
#   help       long help
#
# needs: d2 0.9.0 or newer (brew install d2), node with npx, Graphviz dot if
#        a .dot source exists, lychee (brew install lychee), pandoc and typst
#        (brew install pandoc typst)
# env:   none read
# files: writes diagrams/*.svg and _build/
# exit:  0 done or help shown, 1 a tool or check failed, 2 bad arguments
set -eu

MERMAID_CLI='@mermaid-js/mermaid-cli@12.0.0'
MARKDOWNLINT='markdownlint-cli2@0.23.3'

usage() { sed -n '2p;4,11p' "$0" | sed 's/^# \{0,1\}//'; }

need() {
	command -v "$1" >/dev/null || {
		echo "build.sh: $1 not found; install it with: $2" >&2
		exit 1
	}
}

diagrams() {
	need npx "brew install node"
	need d2 "brew install d2"
	for src in diagrams/*.mmd; do
		npx -y -p "$MERMAID_CLI" mmdc -q -c diagrams/mermaid.json -b white \
			-i "$src" -o "${src%.mmd}.svg"
		# Mermaid starts each later word's tspan with a space; keep it, or
		# rsvg and PDF renderers run the words together.
		svg="${src%.mmd}.svg"
		sed '1s/<svg /<svg xml:space="preserve" /' "$svg" >"$svg.tmp"
		mv "$svg.tmp" "$svg"
	done
	for src in diagrams/*.d2; do
		d2 "$src" "${src%.d2}.svg"
	done
	for src in diagrams/*.dot; do
		[ -e "$src" ] || continue
		need dot "brew install graphviz"
		dot -Tsvg -Gbgcolor=white "$src" -o "${src%.dot}.svg"
	done
}

check() {
	need npx "brew install node"
	need lychee "brew install lychee"
	diagrams # first, so the link check sees every SVG
	npx -y "$MARKDOWNLINT" "*.md"
	lychee --offline --include-fragments --no-progress ./*.md
}

export_book() {
	need pandoc "brew install pandoc"
	need typst "brew install typst"
	mkdir -p _build
	set -- README.md [0-9][0-9]-*.md
	pandoc -f gfm --file-scope --toc --lua-filter=pandoc-links.lua --metadata title="The tuidraw book" \
		--pdf-engine=typst -o _build/tuidraw.pdf "$@"
	pandoc -f gfm --file-scope --toc --lua-filter=pandoc-links.lua --metadata title="The tuidraw book" \
		-o _build/tuidraw.epub "$@"
	echo "wrote docs/_build/tuidraw.pdf and docs/_build/tuidraw.epub"
}

case "${1:-}" in
-h | --help) usage; exit 0 ;;
help) sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
diagrams | check | export) ;;
*) usage >&2; exit 2 ;;
esac

cd "$(dirname "$0")"
case "$1" in
diagrams) diagrams ;;
check) check ;;
export) export_book ;;
esac
