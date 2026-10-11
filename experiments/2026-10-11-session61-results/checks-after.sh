#!/usr/bin/env bash
set -euo pipefail
S=/home/ehrli/ngn-data/night-20261007/session61-correctness
export PATH=/usr/local/go/bin:$PATH GOMAXPROCS=2 GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off
cd "$S"
finish() { local rc=$?; echo "$rc" > "$S/checks.exit"; date -Is > "$S/checks.done"; }
trap finish EXIT
date -Is > checks.started
[ "$(cat before.exit)" = 0 ]
for c in see singular; do
  cp "$S/$c-search.go" "$S/$c/engine/search.go"
  cd "$S/$c"
  if [ "$c" = see ]; then t=TestSEE; f=see_geometry_test.go; else t='Test(SingularCorrectionBounds|SingularVerification|Probcut|Correction)'; f=singular_correction_test.go; fi
  gofmt -w engine/search.go "engine/$f"
  timeout 10m go test -p 2 -short -run "^$t" -count=1 -v ./engine > "$S/$c-narrow.log" 2>&1
  timeout 20m go test -p 2 -short ./engine -count=1 > "$S/$c-engine-short.log" 2>&1
  timeout 20m go test -p 2 -short -race ./engine -count=1 > "$S/$c-engine-race.log" 2>&1
  timeout 20m go test -p 2 -short ./... -count=1 > "$S/$c-full-short.log" 2>&1
  timeout 10m go vet -p 2 ./... > "$S/$c-vet.log" 2>&1
  timeout 10m go build -p 2 -pgo="$S/$c/default.pgo" -o "$S/$c-ngn" . > "$S/$c-build.log" 2>&1
  date -Is > "$S/$c-checks.done"
  python3 - "$S/status.jsonl" "$c" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage=sys.argv[2],event='checks-pass',detail='narrow,engine-short/race,full-short,vet,accepted-PGO build; no games/adoption'))+'\n')
PY
 done
