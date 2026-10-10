#!/usr/bin/env bash
# Conditional full recipe and fixed gate. Verdict review never promotes playing source.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session37-ngnn4-full
F=$R/session35-factorizer
A=$R/session34-architecture
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=$R/session17f/go-cache PYTHONPATH=$A/source PYTHONDONTWRITEBYTECODE=1
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session37-ngnn4-full',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; review receipts before any adoption; no retry/extension"; }
trap finish EXIT
date -Is > controller.started
st wait 'native pilot/time-check/partial-root/factorizer completion; admit only pilot<=10h and clean factorizer shelve'
python3 - "$R" <<'PY'
import datetime,json,os,select,sys
from pathlib import Path
r=Path(sys.argv[1])
for name in ['session35-ngnn4-pilot','session36-time-checks','session37-partial-root','session35-factorizer']:
 p=r/name
 if not (p/'controller.done').exists():
  fd=os.pidfd_open(int((p/'controller.pid').read_text()))
  poll=select.poll();poll.register(fd,select.POLLIN)
  assert poll.poll(5*60*60*1000),'dependency timeout: '+name
  os.close(fd)
 assert (p/'controller.done').exists(),'dependency missing terminal: '+name
pilot=r/'session35-ngnn4-pilot';factorizer=r/'session35-factorizer'
reason=None
if int((pilot/'controller.exit').read_text()) != 0:
 reason='pilot incomplete or failed'
elif not json.loads((pilot/'receipt.json').read_text())['budget_admitted']:
 reason='pilot projects more than10GPU hours'
elif int((factorizer/'controller.exit').read_text()) != 0:
 reason='factorizer incomplete, invalid or preflight failed'
elif json.loads((factorizer/'gate-summary.json').read_text())['verdict'] != 'shelved':
 reason='factorizer win needs accepted setting and combined CUDA review'
if reason:
 (r/'session37-ngnn4-full/deferred.json').write_text(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),reason=reason))+'\n')
 print(reason);sys.exit(3)
d=os.statvfs('/mnt/c');assert d.f_bavail*d.f_frsize>=25.5*1024**3
PY
sha256sum -c inputs.sha256 > input-check-before.txt
st training-start 'fresh58x256M/H512L16/KB8NB8/factorizer-off; accepted source/net frozen until architecture verdict'
taskset -c 12,14 nice -n 19 /home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python trial.py > trial.log 2>&1
st training-done 'full recipe/reexport/20 integer parity/actual-trained cost complete; freshA/A and400-game gate next'
cp "$F/ngn" baseline-ngn
cp "$A/candidate-ngn" candidate-ngn
cp "$F/base.nnue" base.nnue
cp trained/best.nnue candidate.nnue
cp "$F/sprt" "$F/openings.txt" "$F/gate-parser.py" .
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
hashes={p:sha(p) for p in ['baseline-ngn','candidate-ngn','base.nnue','candidate.nnue','sprt','openings.txt','gate-parser.py','controller.sh','trial.py','manifest.md']}
assert hashes['base.nnue']=='be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
assert hashes['sprt']=='9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a'
assert hashes['openings.txt']=='974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'
common=['-tc','10+0.1','-concurrency','2','-mingames','1000000','-maxmoves','400','-gametimeout','180','-openings',str(r/'openings.txt'),'-lowpower=false']
def argv(games,kind,net):
 a=['taskset','-c','13,15','nice','-n','19',str(r/'sprt'),'-new',str(r/(kind+'-ngn')),'-base',str(r/'baseline-ngn'),*common,'-maxgames',str(games)]
 for side,file in [('new',net),('base','base.nnue')]:
  for n,v in [('EvalFile',str(r/file)),('UseNNUE','true'),('Hash','64'),('Threads','1'),('OwnBook','false')]:a+=['-'+side+'option',n+'='+v]
 return a
j=dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),base_source='ac658e2ce509154775d9567807a5ccf8485b6dca',candidate_source='2acf834945d48378209853464b86cefe72587ccc',tc='10+0.1',cpus=[13,15],nice=19,concurrency=2,environment={'GOMAXPROCS':'1','GOAMD64':'v3'},machine='Ryzen9800X3D / WSL2',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),aa_argv=argv(40,'baseline','base.nnue'),gate_argv=argv(400,'candidate','candidate.nnue'),files_sha256=hashes,decision='A/A40 contains0/zero faults; complete400 printedANDunrounded lower95>0/zero faults/intact inputs; otherwise shelve without extension',automatic_promotion=False)
(r/'game-manifest.json').write_text(json.dumps(j,indent=2)+'\n')
hashes['game-manifest.json']=sha('game-manifest.json')
(r/'game-inputs.sha256').write_text(''.join(h+'  '+p+'\n' for p,h in hashes.items()))
PY
export GOMAXPROCS=1
COMMON=(-tc 10+0.1 -concurrency 2 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
st aa-start 'fresh base-vs-itself A/A40 at exact architecture gate topology13,15'
timeout --signal=TERM --kill-after=15s 30m taskset -c 13,15 nice -n 19 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 gate-parser.py gate aa.log --gate-mode aa > aa-summary.json
NEW[1]="EvalFile=$S/candidate.nnue"
st gate-start 'exactly400 paired10+0.1; separate actual-base and architecture binaries/nets; lower95>0'
timeout --signal=TERM --kill-after=15s 4h taskset -c 13,15 nice -n 19 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 gate-parser.py gate gate.log --gate-mode gate > gate-summary.json
python3 - <<'PY'
import re
from pathlib import Path
assert not re.search(r'PANIC|RECOVERED|FATAL',Path('ngn_crashes.log').read_text())
PY
sha256sum -c game-inputs.sha256 > game-input-check-after.txt
sha256sum -c inputs.sha256 > input-check-after.txt
st gate-done "$(cat gate-summary.json); review before any architecture/net adoption"
