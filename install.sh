#!/bin/bash
# install.sh - build tuidraw and install it with `go install`.
#
# Runs the tests first, then installs the binary into $GOBIN, or
# $(go env GOPATH)/bin when GOBIN is unset. Warns when that directory is not
# on PATH. Safe to rerun: it overwrites the old binary.
#
# usage: ./install.sh [--skip-tests]
#
# options:
#   -h, --help     short help
#   help           long help
#   --skip-tests   install without running the tests
#
# env:   GOBIN, GOPATH (read through `go env`)
# files: writes <bin dir>/tuidraw
# exit:  0 installed or help shown, 1 go missing or tests/build failed,
#        2 bad arguments
set -euo pipefail

TEST_TIMEOUT=120 # seconds; the suite takes about 5 s

usage() { sed -n '2p;8,13p' "$0" | sed 's/^# \{0,1\}//'; }

skip=0
case "${1:-}" in
-h | --help) usage; exit 0 ;;
help) sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
--skip-tests) skip=1 ;;
"") ;;
*) usage >&2; exit 2 ;;
esac

cd "$(dirname "$0")"

if ! command -v go >/dev/null; then
	echo "install.sh: go not found; install it with: brew install go" >&2
	exit 1
fi

if [ "$skip" = 0 ]; then
	echo "running tests (about 5 s)"
	timeout "$TEST_TIMEOUT" go test ./...
fi

go install ./cmd/tuidraw

bin="$(go env GOBIN)"
[ -n "$bin" ] || bin="$(go env GOPATH)/bin"
echo "installed $bin/tuidraw"

case ":$PATH:" in
*":$bin:"*) ;;
*) echo "warning: $bin is not on PATH; add: export PATH=\"$bin:\$PATH\"" >&2 ;;
esac
