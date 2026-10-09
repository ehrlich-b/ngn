#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session19-board-key-order2
export PATH=/usr/local/go/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='board-key-order',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; diagnostic only, no node-neutral adoption or strength claim"; }
trap finish EXIT
date -Is > controller.started
st wait "pid $$; native board-layout CPU completion; new key draw-order identity diagnostic"
python3 - "$R/session19-board2/controller.done" "$R/session19-board2/controller.pid" <<'PY'
import os,select,sys
from pathlib import Path
done,pidfile=map(Path,sys.argv[1:])
if not done.exists():
 fd=os.pidfd_open(int(pidfile.read_text()));p=select.poll();p.register(fd,select.POLLIN)
 assert p.poll(2*60*60*1000),'board CPU dependency timeout';os.close(fd)
assert done.exists(),'board test process ended without marker'
PY
[[ $(cat "$R/session19-board2/controller.exit") == 0 ]]
mkdir candidate
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch)
st tests-start 'own metadata-first/square-major key draws; semantic hash tests and fixed-depth comparison'
(cd candidate && taskset -c 13,15 nice -n 19 go test -p 2 -short -run 'Hash|EnPassant|PerftReference' -count=1 ./engine) > hash-perft.log 2>&1
(cd candidate && taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$R/session19-board2/baseline/default.pgo" -o "$S/candidate-probe" ./cmd/speedprobe) > probe-build.log 2>&1
GOMAXPROCS=1 taskset -c 13 nice -n 19 "$S/candidate-probe" -mode identity -depth 14 -net "$R/session18d/base.nnue" > candidate-identity.jsonl
python3 - <<'PY'
import datetime,hashlib,json
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
a=[json.loads(s) for s in (r.parent/'session19-board2/baseline-identity.jsonl').read_text().splitlines()]
b=[json.loads(s) for s in (r/'candidate-identity.jsonl').read_text().splitlines()]
fields=['position','kind','depth','nodes','score','best','pv']
assert len(a)==len(b)==12
differences=[dict(baseline={k:x[k] for k in fields},candidate={k:y[k] for k in fields}) for x,y in zip(a,b) if any(x[k]!=y[k] for k in fields)]
receipt=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),hash_perft_passed=True,node_identical=not differences,compared=12,differences=differences,new_draw_order_completed=True,source_hash_sha256=sha(r/'candidate/engine/hash.go'),patch_sha256=sha(r/'candidate.patch'),probe_sha256=sha(r/'candidate-probe'),logs_sha256={s:sha(r/s) for s in ['hash-perft.log','probe-build.log','candidate-identity.jsonl']},adopted=False,decision='node-neutral admission fails if any difference; no games or adoption authorized by this diagnostic')
(r/'receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
print(json.dumps(dict(node_identical=not differences,differences=len(differences))))
PY
st diagnostic-done 'new key order measured; inspect receipt before claiming any node-neutral board replacement'
