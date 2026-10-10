#!/usr/bin/env bash
# One current-plan-authorized key confirmation on the actual accepted base.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session38-key-plan
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session38-key-plan',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; review before adoption; no retry or extension"; }
trap finish EXIT
date -Is > controller.started
sha256sum -c frozen-inputs.sha256 > frozen-check-before.txt
python3 - <<'PY'
import json,os
from pathlib import Path
j=json.loads(Path('nomination.json').read_text())
assert j['diagnostics_reviewed'] and j['diagnostic_defects']==0
assert j['base_source']=='ac658e2ce509154775d9567807a5ccf8485b6dca' and j['candidate_change']=='existing-key-initializer-only'
d=os.statvfs('/mnt/c'); assert d.f_bavail*d.f_frsize>=25.5*1024**3
PY
st checks-reused 'reviewed full-short/full-race/vet/explicit-PGO current-source forge receipt; no repeated source checks/build'
export GOMAXPROCS=1
COMMON=(-tc 10+0.1 -concurrency 2 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
p=Path.cwd();sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
nom=json.loads((p/'nomination.json').read_text())
names=['code-check-receipt.json','baseline-ngn','candidate-ngn','sprt','openings.txt','base.nnue','gate-parser.py','candidate.patch','base.tar','nomination.json','controller.sh']
hashes={n:sha(p/n) for n in names}
checked=json.loads((p/'code-check-receipt.json').read_text())
sources=['engine/hash.go','engine/uci_process_test.go']
source_hashes={f:checked['files_sha256']['source/'+f] for f in sources}
assert hashes['candidate-ngn']==checked['files_sha256']['candidate-ngn']
assert hashes['baseline-ngn']==nom['baseline_binary_sha256']
assert hashes['base.nnue']==nom['base_net_sha256']
assert hashes['candidate.patch']==nom['candidate_patch_sha256']
common=['-tc','10+0.1','-concurrency','2','-mingames','1000000','-maxmoves','400','-gametimeout','180','-openings',str(p/'openings.txt'),'-lowpower=false']
options=[]
for side in ['new','base']:
 for n,v in [('EvalFile',str(p/'base.nnue')),('UseNNUE','true'),('Hash','64'),('Threads','1'),('OwnBook','false')]: options+=['-'+side+'option',n+'='+v]
def argv(candidate,games): return ['taskset','-c','13,15','nice','-n','19',str(p/'sprt'),'-new',str(p/candidate),'-base',str(p/'baseline-ngn'),*common,'-maxgames',str(games),*options]
j=dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),record='experiments/2026-10-09-key-plan-current.md',base_source=nom['base_source'],candidate_topic=nom['candidate_topic'],candidate_source_sha256=source_hashes,tc='10+0.1',concurrency=2,cpus=[13,15],nice=19,environment={'GOMAXPROCS':'1','GOAMD64':'v3'},machine='Ryzen9800X3D / WSL2',job_memory_max='2GiB',concurrent_gpu_training='4GiB scope on12,14 throughout A/A and gate',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),profile_sha256=checked['files_sha256']['source/default.pgo'],code_checks_machine='forge',code_checks_receipt_sha256=hashes['code-check-receipt.json'],aa_argv=argv('baseline-ngn',40),gate_argv=argv('candidate-ngn',400),files_sha256=hashes,decision='A/A40 complete20 pairs, printed/recomputed paired CI contains0 and zero faults; then exactly400 games/200 pairs, unrounded upper95>0 and zero flags/crashes/illegal/no-move/intact inputs: provisional provenance keep; otherwise shelve; combined batch SPRT [0,+6] remains required; no retries/extensions',automatic_promotion=False)
(p/'game-manifest.json').write_text(json.dumps(j,indent=2)+'\n')
hashes['game-manifest.json']=sha(p/'game-manifest.json')
(p/'inputs.sha256').write_text(''.join(h+'  '+n+'\n' for n,h in hashes.items()))
PY
sha256sum -c inputs.sha256 > input-check-before.txt
st aa-start 'plan-authorized ac658e2 confirmation; fresh A/A40; all earlier key verdicts preserved'
timeout --signal=TERM --kill-after=15s 30m taskset -c 13,15 nice -n 19 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 gate-parser.py gate aa.log --gate-mode aa > aa-summary.json
st gate-start 'exactly400 paired10+0.1 games vs immediate ac658e2 base; upper95>0 provenance rule'
timeout --signal=TERM --kill-after=15s 3h taskset -c 13,15 nice -n 19 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 gate-parser.py gate gate.log --gate-mode gate > original-parser-summary.json
sha256sum -c inputs.sha256 > input-check-after.txt
sha256sum -c frozen-inputs.sha256 > frozen-check-after.txt
python3 - <<'PY'
import json,re
from pathlib import Path
j=json.loads(Path('original-parser-summary.json').read_text())
assert not re.search(r'PANIC|RECOVERED|FATAL',Path('ngn_crashes.log').read_text())
j['verdict']='provisional-provenance-keep' if j['exact_elo_ci95'][2]>0 else 'shelved'
Path('gate-summary.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st gate-done "$(cat gate-summary.json); no automatic adoption; combined provenance certification remains required"
