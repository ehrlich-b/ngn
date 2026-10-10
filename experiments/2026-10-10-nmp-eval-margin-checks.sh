#!/usr/bin/env bash
set -euo pipefail
S=/work/lanes/ngn-s44-nmp-eval-margin
I=/work/lanes/ngn-s44-capture-quiet
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
cd "$S"
st() {
 python3 - "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open('status.jsonl','a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session44-nmp-eval-margin',event=sys.argv[1],detail=sys.argv[2]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; code only, actual-base games remain required"; }
trap finish EXIT
date -Is > controller.started
st wait 'native completion of250221/capture checks to preserve forge memory allocation'
python3 - "$I" <<'PY'
import os,select,sys
from pathlib import Path
p=Path(sys.argv[1])
if not (p/'controller.done').exists():
 fd=os.pidfd_open(int((p/'controller.pid').read_text()));poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(90*60*1000),'capture checks wait expired'
 os.close(fd)
assert (p/'controller.done').exists()
PY
mkdir baseline candidate
tar -xf base.tar -C baseline
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch)
taskset -c 8-11 nice -n 10 gofmt -w candidate/engine/search.go candidate/engine/search_nmp_eval_margin_test.go
cp candidate/engine/search_nmp_eval_margin_test.go baseline/engine/
st checks-start 'single eval-dependent null depth on56b35ef; old-fail/new-pass and full checks'
if (cd baseline && timeout 10m taskset -c 8-11 nice -n 10 go test -p 2 -run '^TestNMPEvalMarginRoutesNullChild/192$' -count=1 -v ./engine) > old-witness.log 2>&1; then
 echo 'Old source unexpectedly passed live call-site witness' >&2;exit 1
fi
rg -q 'gap=192 main move loops=' old-witness.log
(cd candidate && timeout 15m taskset -c 8-11 nice -n 10 go test -p 2 -run '^TestNMPEvalMargin' -count=1 -v ./engine) > new-witness.log 2>&1
for mode in short race; do
 flags=(); [[ $mode != race ]] || flags=(-race)
 (cd candidate && timeout 30m taskset -c 8-11 nice -n 10 go test -p 2 -short "${flags[@]}" ./... -count=1) > "full-$mode.log" 2>&1
done
(cd candidate && timeout 15m taskset -c 8-11 nice -n 10 go vet -p 2 ./...) > vet.log 2>&1
(cd candidate && timeout 15m taskset -c 8-11 nice -n 10 go build -p 2 -pgo="$S/candidate/default.pgo" -o "$S/candidate-ngn" .) > build.log 2>&1
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
j=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),base_source='56b35ef',patch_sha256=sha(r/'candidate.patch'),source_sha256={f:sha(r/'candidate'/f) for f in ['engine/search.go','engine/search_nmp_eval_margin_test.go']},binary_sha256=sha(r/'candidate-ngn'),old_fail_new_pass=True,tests_pass=True,checks=['live null child191/192cp and larger gaps','pawn-only guard','full short ./...','full race ./...','vet','explicit accepted PGO build'],machine='forge Debian13',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',cpus=[8,9,10,11],nice=10,memory_max=2147483648,strength_games=0,automatic_games=False,automatic_adoption=False)
(r/'receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st checks-pass 'old-fail/new-pass and full source checks; isolated/uncommitted; priority gates decide first'
