#!/usr/bin/env bash
# Actual-base checks only, after the two existing isolated time candidates.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session37-partial-root
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session37-partial-root',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; isolated code checks only; long-clock games required"; }
trap finish EXIT
date -Is > controller.started
st wait 'native session36 time-check completion; then12,14/nice19; no games'
python3 - "$R/session36-time-checks" <<'PY'
import os,select,sys
from pathlib import Path
p=Path(sys.argv[1])
if not (p/'controller.done').exists():
 fd=os.pidfd_open(int((p/'controller.pid').read_text()))
 poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(2*60*60*1000),'dependency timeout'
 os.close(fd)
assert (p/'controller.done').exists(),'dependency exited without marker'
d=os.statvfs('/mnt/c');assert d.f_bavail*d.f_frsize>=25.5*1024**3
PY
sha256sum -c inputs.sha256 > input-check-before.txt
mkdir source
tar -xf base.tar -C source
(cd source && git apply ../candidate.patch)
st checks-start 'clean ac658e2 plus completed-root-choice patch only; narrow/full-short/full-race/vet/PGO'
(cd source && timeout 10m taskset -c 12,14 nice -n 19 go test -p 2 -run '^TestPartialRoot' -count=1 -v ./engine) > narrow.log 2>&1
(cd source && timeout 20m taskset -c 12,14 nice -n 19 go test -p 2 -short ./... -count=1) > full-short.log 2>&1
(cd source && timeout 20m taskset -c 12,14 nice -n 19 go test -p 2 -short -race ./... -count=1) > full-race.log 2>&1
(cd source && timeout 10m taskset -c 12,14 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
(cd source && timeout 10m taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/source/default.pgo" -o "$S/candidate-ngn" .) > build.log 2>&1
sha256sum -c inputs.sha256 > input-check-after.txt
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
j=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),base_source='ac658e2ce509154775d9567807a5ccf8485b6dca',candidate_topic='night/partial-root',change='retain completed in-window root improvement at tournament clock stop only',cpus=[12,14],nice=19,machine='Ryzen9800X3D / WSL2',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',checks=['narrow root-choice tests','full-short ./...','full-race ./...','go vet ./...','explicit accepted-PGO build'],files={p:sha(p) for p in ['base.tar','candidate.patch','controller.sh','manifest.md','source/engine/search.go','source/engine/partial_root_test.go','source/default.pgo','candidate-ngn']},strength_games=0,automatic_adoption=False)
Path('receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st checks-done 'receipt.json; code checks only; actual-source60+0.6 gate remains required'
