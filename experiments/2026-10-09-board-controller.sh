#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session19-board2
export PATH=/usr/local/go/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='board-layout',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; partial provenance remedy, no source promotion"; }
trap finish EXIT
date -Is > controller.started
st wait "pid $$; native corrected TT test completion; CPU13,15 suites and exact identity"
python3 - "$R/session19-tt2/controller.done" "$R/session19-tt2/controller.pid" <<'PY'
import os,select,sys
from pathlib import Path
done,pidfile=map(Path,sys.argv[1:])
if not done.exists():
 fd=os.pidfd_open(int(pidfile.read_text()));p=select.poll();p.register(fd,select.POLLIN)
 assert p.poll(2*60*60*1000),'TT tests dependency timeout';os.close(fd)
assert done.exists(),'TT test process ended without marker'
PY
st tests-start 'single-side metadata, own move/occupancy layout and board names; key assignment unchanged for identity'
for mode in engine-short engine-race full-short full-race; do
 pkg=./engine; [[ $mode != full-* ]] || pkg=./...
 flags=(); [[ $mode != *race ]] || flags=(-race)
 (cd candidate && taskset -c 13,15 nice -n 19 go test -p 2 -short "${flags[@]}" "$pkg" -count=1) > "$mode.log" 2>&1
done
(cd candidate && taskset -c 13,15 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
for kind in baseline candidate; do
 (cd "$kind" && taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-probe" ./cmd/speedprobe) > "$kind-probe-build.log" 2>&1
 (cd "$kind" && taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-ngn" .) > "$kind-build.log" 2>&1
 GOMAXPROCS=1 taskset -c 13 nice -n 19 "$S/$kind-probe" -mode identity -depth 14 -net "$R/session18d/base.nnue" > "$kind-identity.jsonl"
 (cd "$kind" && GOMAXPROCS=1 taskset -c 13 nice -n 19 go test -p 1 -run '^TestPerftReferenceSuite$' -count=1 -v ./engine) > "$kind-perft.log" 2>&1
done
GOMAXPROCS=1 taskset -c 13 nice -n 19 python3 nodecheck.py > nodecheck.json
python3 - <<'PY'
import datetime,hashlib,json
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
rows={k:[json.loads(s) for s in (r/(k+'-identity.jsonl')).read_text().splitlines()] for k in ('baseline','candidate')}
fields=['position','kind','depth','nodes','score','best','pv']
assert len(rows['baseline'])==len(rows['candidate'])==12
for a,b in zip(rows['baseline'],rows['candidate']):
 assert {k:a[k] for k in fields}=={k:b[k] for k in fields},(a,b)
assert json.loads((r/'nodecheck.json').read_text())['identical']
files=[s.split(' b/',1)[1] for s in (r/'candidate.patch').read_text().splitlines() if s.startswith('+++ b/')]
logs=['engine-short.log','engine-race.log','full-short.log','full-race.log','vet.log','baseline-perft.log','candidate-perft.log','baseline-identity.jsonl','candidate-identity.jsonl','nodecheck.json']
receipt=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),passed=True,fixed_depth_identities=12,nodecheck_identities=3,perft_reference_positions=6,keys_reassigned=False,new_draw_order_completed=False,source_sha256={s:sha(r/'candidate'/s) for s in files},logs_sha256={s:sha(r/s) for s in logs},binary_sha256={k:sha(r/(k+'-ngn')) for k in ('baseline','candidate')},adopted=False)
(r/'receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
PY
st tests-pass '12 fixed-depth plus3 canonical UCI identities and six perft oracles; key draw-order replacement remains separate work'
