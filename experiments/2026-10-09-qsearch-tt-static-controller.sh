#!/usr/bin/env bash
# Code checks only; time diagnostics and higher-priority lanes go first.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session23-qstatic
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='qsearch-tt-static',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
 local rc=$?
 echo "$rc" > controller.exit
 date -Is > controller.done
 st end "rc $rc; code checks only; no strength verdict or adoption"
}
trap finish EXIT
date -Is > controller.started
st wait 'native time-current completion; then C:>=32GiB; CPU12,14 tests'
python3 - "$R/session23-time" <<'PY'
import os,select,sys,time
from pathlib import Path
p=Path(sys.argv[1])
if not (p/'controller.done').exists():
 fd=os.pidfd_open(int((p/'controller.pid').read_text()))
 poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(27*3600*1000),'time dependency timeout'
 os.close(fd)
assert (p/'controller.done').exists(),'time controller ended without marker'
deadline=time.monotonic()+12*3600
while True:
 d=os.statvfs('/mnt/c')
 if d.f_bavail*d.f_frsize>=32*1024**3:break
 assert time.monotonic()<deadline,'C: capacity wait expired'
 time.sleep(300)
PY
st tests-start 'isolated39eb128 qsearch bound refinement; no time/SEE/key/pawn overlay'
for kind in baseline candidate; do
 mkdir "$kind"
 tar -xf base.tar -C "$kind"
done
(cd candidate && git apply ../candidate.patch)
cp candidate/engine/qsearch_tt_static_test.go baseline/engine/
set +e
(cd baseline && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestQStaticCompatibleBoundChangesQuietHorizon$' -count=1 -v ./engine) > baseline-witness.log 2>&1
rc=$?
set -e
[[ $rc == 1 ]]
rg -q 'quiet horizon ignored compatible bound' baseline-witness.log
unlink baseline/engine/qsearch_tt_static_test.go
(cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestQStatic' -count=1 -v ./engine) > candidate-narrow.log 2>&1
for mode in full-short full-race; do
 flags=(); [[ $mode != *race ]] || flags=(-race)
 (cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -short "${flags[@]}" ./... -count=1) > "$mode.log" 2>&1
done
(cd candidate && taskset -c 12,14 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
for kind in baseline candidate; do
 (cd "$kind" && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-ngn" .) > "$kind-build.log" 2>&1
done
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
x=dict(passed=True,base_source='39eb128',net_for_future_gate='be34532c',utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
 go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',os=platform.platform(),machine='WSL Ryzen9800X3D',cpus=[12,14],nice=19,
 source_sha256={f:sha('candidate/'+f) for f in ['engine/search.go','engine/qsearch_tt_static_test.go']},
 binaries={f:sha(f) for f in ['baseline-ngn','candidate-ngn']},inputs={f:sha(f) for f in ['base.tar','candidate.patch','baseline/default.pgo']},
 games_launched=False,strength_verdict=None)
Path('receipt.json').write_text(json.dumps(x,indent=2)+'\n')
PY
st tests-pass 'narrow witnesses, full short/race, vet and PGO builds; actual-base games still required'
