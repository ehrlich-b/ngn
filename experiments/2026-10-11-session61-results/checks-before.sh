#!/usr/bin/env bash
set -euo pipefail
S=/home/ehrli/ngn-data/night-20261007/session61-correctness
export PATH=/usr/local/go/bin:$PATH GOMAXPROCS=2 GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off
cd "$S"
finish() { local rc=$?; echo "$rc" > "$S/before.exit"; date -Is > "$S/before.done"; }
trap finish EXIT
date -Is > before.started
go version > environment.txt
for c in see singular; do
  cd "$S/$c"
  gofmt -w engine/*geometry_test.go 2>/dev/null || gofmt -w engine/singular_correction_test.go
  set +e
  if [ "$c" = see ]; then t=TestSEEPostMoveGeometry; else t=TestSingularCorrectionBounds; fi
  timeout 10m go test -p 2 -short -run "^$t$" -count=1 -v ./engine > "$S/$c-before.log" 2>&1
  rc=$?
  set -e
  echo "$rc" > "$S/$c-before.exit"
  if [ "$rc" -ne 1 ]; then exit 2; fi
 done
