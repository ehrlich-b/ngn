#!/usr/bin/env bash
# Invoke only after the next session reviews completed training/parity/cost.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session39-ngnn4-key-games
T=$R/session38-ngnn4-bounded
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=1
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session39-ngnn4-key-games',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; review before architecture/net adoption; no retry or extension"; }
trap finish EXIT
date -Is > controller.started
sha256sum -c frozen-inputs.sha256 > input-check-before.txt
python3 - "$T" <<'PY'
import datetime,hashlib,json,os,platform,subprocess,sys
from pathlib import Path
p=Path.cwd();t=Path(sys.argv[1]);sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
assert (t/'controller.done').exists() and int((t/'controller.exit').read_text())==0
training=json.loads((t/'training-receipt.json').read_text());cost=json.loads((t/'actual-cost.json').read_text())
assert training['epochs']==58 and training['total_positions']==14848000000
assert training['reexport_identical'] and training['parity_cases']==20
assert cost['trained_net']==training['net_sha256'] and len(cost['rounds'])==4
assert sha(t/'trained/best.nnue')==training['net_sha256']
d=os.statvfs('/mnt/c');assert d.f_bavail*d.f_frsize>=25.5*1024**3
receipt=json.loads(Path('source-check-receipt.json').read_text())
assert receipt['base_source']=='6c0fc26a17e359442098802f9f33ee673c118c90'
assert sha(p/'candidate-ngn')==receipt['files_sha256']['candidate-ngn']
(p/'candidate.nnue').write_bytes((t/'trained/best.nnue').read_bytes())
(p/'training-receipt.json').write_bytes((t/'training-receipt.json').read_bytes())
(p/'actual-cost.json').write_bytes((t/'actual-cost.json').read_bytes())
common=['-tc','10+0.1','-concurrency','2','-mingames','1000000','-maxmoves','400','-gametimeout','180','-openings',str(p/'openings.txt'),'-lowpower=false']
def argv(kind,net,count):
 a=['taskset','-c','13,15','nice','-n','19',str(p/'sprt'),'-new',str(p/(kind+'-ngn')),'-base',str(p/'baseline-ngn'),*common,'-maxgames',str(count)]
 for side,file in [('new',net),('base','base.nnue')]:
  for n,v in [('EvalFile',str(p/file)),('UseNNUE','true'),('Hash','64'),('Threads','1'),('OwnBook','false')]:a+=['-'+side+'option',n+'='+v]
 return a
names=['baseline-ngn','candidate-ngn','base.nnue','candidate.nnue','sprt','openings.txt','gate-parser.py','source-check-receipt.json','nomination.json','training-receipt.json','actual-cost.json','controller.sh','manifest.md','architecture.patch']
hashes={n:sha(p/n) for n in names}
nom=json.loads(Path('nomination.json').read_text())
j=dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),record='experiments/2026-10-10-ngnn4-key-games.md',base_source=receipt['base_source'],candidate_source=nom['architecture_source_commit'],architecture_parent=receipt['architecture_parent'],training_source='2acf834945d48378209853464b86cefe72587ccc',trainer_unchanged=True,tc='10+0.1',cpus=[13,15],nice=19,concurrency=2,environment={'GOMAXPROCS':'1','GOAMD64':'v3'},machine='Ryzen9800X3D / WSL2',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),memory_max='2GiB',concurrent_workloads=['possible combined provenance SPRT scope3GiB/0-11','hopper unchanged'],profile_sha256=receipt['files_sha256']['source/default.pgo'],aa_argv=argv('baseline','base.nnue',40),gate_argv=argv('candidate','candidate.nnue',400),files_sha256=hashes,decision='A/A40 contains0/zero faults; exactly400/200 pairs, printedANDunrounded lower95>0/zero faults/intact inputs; otherwise shelve; no extension',automatic_promotion=False)
(p/'game-manifest.json').write_text(json.dumps(j,indent=2)+'\n')
hashes['game-manifest.json']=sha(p/'game-manifest.json')
(p/'game-inputs.sha256').write_text(''.join(h+'  '+n+'\n' for n,h in hashes.items()))
PY
COMMON=(-tc 10+0.1 -concurrency 2 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
st aa-start 'fresh A/A40 on actual accepted key base13,15; separate architecture source checked'
timeout --signal=TERM --kill-after=15s 30m taskset -c 13,15 nice -n 19 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 gate-parser.py gate aa.log --gate-mode aa > aa-summary.json
NEW[1]="EvalFile=$S/candidate.nnue"
st gate-start 'exactly400 paired10+0.1 vs current key base; architecture lower95>0 rule'
timeout --signal=TERM --kill-after=15s 4h taskset -c 13,15 nice -n 19 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 gate-parser.py gate gate.log --gate-mode gate > gate-summary.json
python3 - <<'PY'
import re
from pathlib import Path
assert not re.search(r'PANIC|RECOVERED|FATAL',Path('ngn_crashes.log').read_text())
PY
sha256sum -c game-inputs.sha256 > game-input-check-after.txt
sha256sum -c frozen-inputs.sha256 > input-check-after.txt
st gate-done "$(cat gate-summary.json); session review required before any architecture/net adoption"
