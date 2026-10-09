#!/usr/bin/env bash
# Standard provenance-replacement gate required by merged plan item0b(a).
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session19-tt-gate
export GOMAXPROCS=1 GOAMD64=v3
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='cluster-tt-gate',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; review fixed-rule result before any source promotion"; }
trap finish EXIT
date -Is > controller.started
st wait "pid $$; native corrected TT full checks, then board diagnostic terminal to free CPU13,15"
for job in session19-tt2 session19-board2 session19-board-key-order2; do
 python3 - "$R/$job/controller.done" "$R/$job/controller.pid" <<'PY'
import os,select,sys
from pathlib import Path
done,pidfile=map(Path,sys.argv[1:])
if not done.exists():
 fd=os.pidfd_open(int(pidfile.read_text()));p=select.poll();p.register(fd,select.POLLIN)
 assert p.poll(2*60*60*1000),'CPU dependency timeout';os.close(fd)
assert done.exists(),'CPU dependency process ended without marker'
PY
done
[[ $(cat "$R/session19-tt2/controller.exit") == 0 ]]
cp "$R/session19-tt2/baseline-ngn" baseline-ngn
cp "$R/session19-tt2/candidate-ngn" candidate-ngn
cp "$R/session19-tt2/receipt.json" tests-receipt.json
cp "$R/session19-tt2/candidate.patch" candidate.patch
cp "$R/session18d/base.nnue" base.nnue
cp "$R/session18d/openings.txt" openings.txt
cp "$R/session18d/sprt" sprt
COMMON=(-tc 10+0.1 -concurrency 2 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
python3 - <<'PY'
import datetime,hashlib,json
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
files=['baseline-ngn','candidate-ngn','tests-receipt.json','candidate.patch','base.nnue','openings.txt','sprt','gate-parser.py','controller.sh']
hashes={s:sha(r/s) for s in files}
common=['-tc','10+0.1','-concurrency','2','-mingames','1000000','-maxmoves','400','-gametimeout','180','-openings',str(r/'openings.txt'),'-lowpower=false']
opts=[]
for role in ('new','base'):
 for n,v in [('EvalFile',str(r/'base.nnue')),('UseNNUE','true'),('Hash','64'),('Threads','1'),('OwnBook','false')]:opts+=['-'+role+'option',n+'='+v]
def argv(candidate,n):return ['taskset','-c','13,15','nice','-n','19',str(r/'sprt'),'-new',str(r/candidate),'-base',str(r/'baseline-ngn'),*common,'-maxgames',str(n),*opts]
x=dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),predeclared_record='experiments/2026-10-09-cluster-tt.md',plan_rule='item0b(a) standard gate overrides prior prospective long-clock rule before games',accepted_runtime='5b882e2',source_snapshot='a49ebfc',files_sha256=hashes,tc='10+0.1',cpus=[13,15],concurrency=2,nice=19,environment={'GOMAXPROCS':'1','GOAMD64':'v3'},aa_argv=argv('baseline-ngn',40),gate_argv=argv('candidate-ngn',400),decision='400 games/200 pairs, printed/recomputed lower95>0 and zero operational failures/intact pins; otherwise shelve without extension',automatic_promotion=False,factorizer_pending=True)
(r/'game-manifest.json').write_text(json.dumps(x,indent=2)+'\n')
with (r/'inputs.sha256').open('w') as f:
 for s,h in hashes.items():f.write(h+'  '+s+'\n')
 f.write(sha(r/'game-manifest.json')+'  game-manifest.json\n')
PY
sha256sum -c inputs.sha256 > input-check-before.txt
st aa-start '40 same-base games; CPU13,15; no TT/source adoption'
timeout --signal=TERM --kill-after=15s 30m taskset -c 13,15 nice -n 19 "$S/sprt" -new "$S/baseline-ngn" -base "$S/baseline-ngn" "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 gate-parser.py gate aa.log --gate-mode aa > aa-summary.json
st gate-start '400 paired games; standard provenance gate; fixed lower95>0/no failures'
timeout --signal=TERM --kill-after=15s 3h taskset -c 13,15 nice -n 19 "$S/sprt" -new "$S/candidate-ngn" -base "$S/baseline-ngn" "${COMMON[@]}" -maxgames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 gate-parser.py gate gate.log --gate-mode gate > gate-summary.json
sha256sum -c inputs.sha256 > input-check-after.txt
st done "$(cat gate-summary.json); no automatic promotion while factorizer's old-base verdict remains pending"
