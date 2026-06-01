#! /usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <go-os> <go-arch>"
  exit 1
fi

GOOS="$1"
GOARCH="$2"

go build -o bin/ ./cmd/kubectl-bind

if [[ "$GOOS" == "windows" ]]; then
  mv "./bin/kubectl-bind.exe" "../klutch-bind-$GOOS-$GOARCH.exe"
  exit 0
fi

mv "./bin/kubectl-bind" "../klutch-bind-$GOOS-$GOARCH"
