#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session66-correctness
cd "$s"
finish() { local rc=$?; echo "$rc" > "$s/checks.exit"; date -Is > "$s/checks.done"; }
trap finish EXIT
date -Is > checks.started
export PATH=/home/ehrli/go/bin:/usr/local/go/bin:$PATH
export GOMAXPROCS=2 GOAMD64=v3
go version > environment.txt
sha256sum base/default.pgo >> environment.txt
for candidate in see singular; do
  cd "$s/$candidate"
  if [[ "$candidate" == see ]]; then pattern='TestSEEPostMoveGeometry|TestStaticExchange';
  else pattern='TestSingular|TestProbCut|Test.*Correction'; fi
  go test -p 2 -short ./engine -count=1 -run "$pattern" -v > "$s/$candidate-narrow.log" 2>&1
  go test -p 2 -short ./engine -count=1 > "$s/$candidate-engine-short.log" 2>&1
  go test -p 2 -short -race ./engine -count=1 > "$s/$candidate-engine-race.log" 2>&1
  go test -p 2 -short ./... -count=1 > "$s/$candidate-full-short.log" 2>&1
  go test -p 2 -short -race ./... -count=1 > "$s/$candidate-full-race.log" 2>&1
  go vet -p 2 ./... > "$s/$candidate-vet.log" 2>&1
  go build -p 2 -pgo=default.pgo -o "$s/$candidate-ngn" . > "$s/$candidate-build.log" 2>&1
  if [[ "$candidate" == see ]]; then regression=engine/see_geometry_test.go;
  else regression=engine/singular_correction_test.go; fi
  sha256sum engine/search.go "$regression" "$s/$candidate-ngn" > "$s/$candidate-inputs.sha256"
  date -Is > "$s/$candidate.done"
done
