#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session80
finish() { local rc=$?; echo "$rc" > "$D/candidate-checks.exit"; date -Is > "$D/candidate-checks.done"; }
trap finish EXIT
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache
cd "$D/candidates/singular-old/source"
set +e
go test -p 1 -short ./engine -run '^TestSingularCorrectionBounds$' -count=1 -v > "$D/candidates/singular-old/old-fail.log" 2>&1
old_rc=$?
set -e
[ "$old_rc" -ne 0 ]
grep -q 'FAIL: TestSingularCorrectionBounds/white/restricted_fail_low' "$D/candidates/singular-old/old-fail.log"
grep -q 'FAIL: TestSingularCorrectionBounds/black/restricted_fail_low' "$D/candidates/singular-old/old-fail.log"
for name in singular qsearch continuation pawn; do
 C=$D/candidates/$name
 cd "$C/source"
 case "$name" in singular) pattern='^TestSingularCorrectionBounds$';; qsearch) pattern='^TestQStatic';; continuation) pattern='^TestContinuationCorrection';; pawn) pattern='^TestPawnQuiet';; esac
 go test -p 1 -short ./engine -run "$pattern" -count=1 -v > "$C/focused.log" 2>&1
 go test -p 1 -short ./engine -count=1 > "$C/engine-short.log" 2>&1
 go test -p 1 -short -race ./engine -count=1 > "$C/engine-race.log" 2>&1
 go test -p 1 -short ./... -count=1 > "$C/full-short.log" 2>&1
 go test -p 1 -short -race ./... -count=1 > "$C/full-race.log" 2>&1
 go vet -p 1 ./... > "$C/vet.log" 2>&1
 go build -p 1 -pgo=default.pgo -o "$C/candidate-ngn" . > "$C/build.log" 2>&1
 python3 - "$C" "$D/candidates/$name.patch" <<'PY'
import datetime,hashlib,json,pathlib,sys
c=pathlib.Path(sys.argv[1]); patch=pathlib.Path(sys.argv[2])
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
(c/'checks-result.json').write_text(json.dumps(dict(base='f203e2f',candidate=c.name,passed=['focused','engine short/race','full short/race','vet','PGO build'],binary_sha256=sha(c/'candidate-ngn'),patch_sha256=sha(patch),completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat()),indent=2)+'\n')
print(c.name,'checks PASS',flush=True)
PY
 done
