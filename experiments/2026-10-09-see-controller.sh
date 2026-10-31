#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session19-see
export PATH=/usr/local/go/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='see-values',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; code checks only; no games or source promotion"; }
trap finish EXIT
date -Is > controller.started
st wait "pid $$; native CUDA-window completion; newly free CPU12,14 tests; TT13,15 and Rodent0-11 remain disjoint"
python3 - "$R/session19-gpu-release2/controller.done" "$R/session19-gpu-release2/controller.pid" <<'PY'
import os,select,sys
from pathlib import Path
done,pidfile=map(Path,sys.argv[1:])
if not done.exists():
 fd=os.pidfd_open(int(pidfile.read_text()));p=select.poll();p.register(fd,select.POLLIN)
 assert p.poll(6*60*60*1000),'CUDA dependency timeout';os.close(fd)
assert done.exists(),'CUDA admission process ended without marker'
PY
[[ $(cat "$R/session19-gpu-release2/controller.exit") == 0 ]]
[[ $(ps -p 630389 -o stat=) == *T* ]]
mkdir baseline candidate
tar -xf base.tar -C baseline
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch)
st tests-start 'NGN-chosen 100/300/340/500/950 exchange and promotion-order values; evaluation/net unchanged'
for mode in engine-short engine-race full-short full-race; do
 pkg=./engine; [[ $mode != full-* ]] || pkg=./...
 flags=(); [[ $mode != *race ]] || flags=(-race)
 (cd candidate && taskset -c 12,14 nice -n 19 go test -p 2 -short "${flags[@]}" "$pkg" -count=1) > "$mode.log" 2>&1
done
(cd candidate && taskset -c 12,14 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
for kind in baseline candidate; do
 (cd "$kind" && taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-ngn" .) > "$kind-build.log" 2>&1
done
python3 - <<'PY'
import datetime,hashlib,json
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
files=['engine/piece.go','engine/piece_test.go','engine/independent_piece_test.go']
receipt=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),passed=True,cpus=[12,14],nice=19,source_sha256={s:sha(r/'candidate'/s) for s in files},binary_sha256={k:sha(r/(k+'-ngn')) for k in ('baseline','candidate')},games_launched=False)
(r/'receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
PY
st tests-pass 'NGN exchange-value candidate code checks pass; freeze actual accepted base for a separate standard gate after TT/board work'
