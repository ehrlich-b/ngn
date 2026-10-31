#!/usr/bin/env bash
set -euo pipefail
S=/work/lanes/ngn-s45-singular-double
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
cd "$S"
st() {
 python3 - "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open('status.jsonl','a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session45-singular-double',event=sys.argv[1],detail=sys.argv[2]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; code only, actual-base rig games required"; }
trap finish EXIT
date -Is > controller.started
# The pilot is complete. Preserve its logs; replace only its preparation source.
cp candidate/engine/search.go pilot-search.go
cp candidate/engine/search_singular_double_test.go pilot-test.go
cp new-witness.log pilot-new-witness.log
cp old-witness.log pilot-old-witness.log
cp baseline/engine/search.go candidate/engine/search.go
(cd candidate && git apply ../runtime.patch)
cp search_singular_double_test.go candidate/engine/
gofmt -w candidate/engine/search.go candidate/engine/search_singular_double_test.go
cp candidate/engine/search_singular_double_test.go baseline/engine/
st checks-start 'bounded single extra singular ply on56b35ef; old-fail/new-pass and full checks'
if (cd baseline && timeout 10m go test -p 2 -run '^TestSingularDoubleChildDepth/gap65$' -count=1 -v ./engine) > old-witness.log 2>&1; then
 echo 'Old source passed live child-depth witness' >&2;exit 1
fi
rg -q 'live singular child depth=6 hit=true, want 7' old-witness.log
(cd candidate && timeout 15m go test -p 2 -run '^TestSingularDouble' -count=1 -v ./engine) > new-witness.log 2>&1
for mode in short race; do
 flags=(); [[ $mode != race ]] || flags=(-race)
 (cd candidate && timeout 30m go test -p 2 -short "${flags[@]}" ./... -count=1) > "full-$mode.log" 2>&1
done
(cd candidate && timeout 15m go vet -p 2 ./...) > vet.log 2>&1
(cd candidate && timeout 15m go build -p 2 -pgo="$S/candidate/default.pgo" -o "$S/candidate-ngn" .) > build.log 2>&1
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
j=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),base_source='56b35ef',runtime_patch_sha256=sha(r/'runtime.patch'),source_sha256={f:sha(r/'candidate'/f) for f in ['engine/search.go','engine/search_singular_double_test.go']},binary_sha256=sha(r/'candidate-ngn'),old_fail_new_pass=True,tests_pass=True,checks=['live child depth64/65cp and depths5/6/12/13','PV child depth','path-budget clamp','verification stop unwind','full short ./...','full race ./...','vet','explicit accepted PGO build'],machine='forge Debian13',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',cpus=[8,9,10,11],nice=10,memory_max=2147483648,strength_games=0)
(r/'receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st checks-pass 'old-fail/new-pass and full source checks; isolated/uncommitted; rig games remain later'
