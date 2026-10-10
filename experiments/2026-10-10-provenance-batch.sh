#!/usr/bin/env bash
# One powered certification of the three provisionally accepted replacements.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session39-provenance-batch
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=1
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session39-provenance-batch',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; review before certification/bisect; no automatic source or net change"; }
trap finish EXIT
date -Is > controller.started
sha256sum -c frozen-inputs.sha256 > input-check-before.txt
python3 - <<'PY'
import datetime,hashlib,json,os,platform,subprocess
from pathlib import Path
p=Path.cwd();sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
d=os.statvfs('/mnt/c');assert d.f_bavail*d.f_frsize>=25.5*1024**3
assert json.loads(Path('source-check-receipt.json').read_text())['base_source']=='6c0fc26a17e359442098802f9f33ee673c118c90'
common=['-tc','60+0.6','-concurrency','6','-maxmoves','400','-gametimeout','900','-openings',str(p/'openings.txt'),'-lowpower=false','-elo0','0','-elo1','6','-alpha','0.05','-beta','0.05']
options=[]
for side in ['new','base']:
 for n,v in [('EvalFile',str(p/'base.nnue')),('UseNNUE','true'),('Hash','64'),('Threads','1'),('OwnBook','false')]:options+=['-'+side+'option',n+'='+v]
def argv(kind,count,minimum):
 return ['taskset','-c','0-11','nice','-n','19',str(p/'sprt'),'-new',str(p/(kind+'-ngn')),'-base',str(p/'baseline-ngn'),*common,'-maxgames',str(count),'-mingames',str(minimum),*options]
files=['baseline-ngn','candidate-ngn','base.nnue','sprt','openings.txt','gate-parser.py','batch-parser.py','source-check-receipt.json','controller.sh','manifest.md','pre-batch.tar','base.tar']
hashes={n:sha(p/n) for n in files}
j=dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),record='experiments/2026-10-10-provenance-batch.md',base_source='5b882e2',candidate_source='6c0fc26a17e359442098802f9f33ee673c118c90',change='combined own TT/board/key/SEE package, including separately accepted null-frame speed keep',tc='60+0.6',concurrency=6,cpus=list(range(12)),nice=19,environment={'GOMAXPROCS':'1','GOAMD64':'v3'},machine='Ryzen9800X3D / WSL2',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),job_memory_max='3GiB',concurrent_workloads=['unchanged4GiB GPU training12,14 until completion','possible independent2GiB architecture trial13,15','hopper unchanged'],profile_sha256=json.loads(Path('source-check-receipt.json').read_text())['files_sha256']['source/default.pgo'],aa_argv=argv('baseline',40,1000000),gate_argv=argv('candidate',4000,400),files_sha256=hashes,decision='fresh A/A40 contains0/zero faults; SPRT[0,+6] min400/cap4000, H1 AND final unrounded paired lower95>0/zero faults/intact inputs certifies; otherwise requires bisect; no extension',automatic_promotion=False)
(p/'game-manifest.json').write_text(json.dumps(j,indent=2)+'\n')
hashes['game-manifest.json']=sha(p/'game-manifest.json')
(p/'game-inputs.sha256').write_text(''.join(h+'  '+n+'\n' for n,h in hashes.items()))
PY
COMMON=(-tc 60+0.6 -concurrency 6 -maxmoves 400 -gametimeout 900 -openings "$S/openings.txt" -lowpower=false -elo0 0 -elo1 6 -alpha .05 -beta .05)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
st aa-start 'fresh A/A40 at60+0.6/concurrency6/0-11/nice19; combined certification topology'
timeout --signal=TERM --kill-after=15s 90m taskset -c 0-11 nice -n 19 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 -mingames 1000000 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 gate-parser.py gate aa.log --gate-mode aa > aa-summary.json
st gate-start 'SPRT[0,+6] at60+0.6/min400/cap4000 vs pre-batch5b882e2; no extension'
timeout --signal=TERM --kill-after=15s 48h taskset -c 0-11 nice -n 19 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 4000 -mingames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 batch-parser.py > gate-summary.json
sha256sum -c game-inputs.sha256 > game-input-check-after.txt
sha256sum -c frozen-inputs.sha256 > input-check-after.txt
st gate-done "$(cat gate-summary.json); session review required; no automatic certification/revert"
