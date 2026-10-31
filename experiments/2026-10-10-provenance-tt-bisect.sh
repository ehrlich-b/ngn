#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session46-tt-bisect
P=$R/session39-provenance-batch
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=1
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session46-tt-bisect',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; diagnostic only; session review required; no source changes"; }
trap finish EXIT
date -Is > controller.started
st wait 'native completion of original provenance3465; valid requires-bisect terminal only'
python3 - "$P" <<'PY'
import json,os,select,sys
from pathlib import Path
p=Path(sys.argv[1])
if not (p/'controller.done').exists():
 fd=os.pidfd_open(int((p/'controller.pid').read_text()));poll=select.poll();poll.register(fd,select.POLLIN);assert poll.poll(48*3600*1000),'parent wait expired';os.close(fd)
assert int((p/'controller.exit').read_text())==0
for file in ['game-input-check-after.txt','input-check-after.txt']:
 lines=(p/file).read_text().splitlines();assert lines and all(line.endswith(': OK') for line in lines)
j=json.loads((p/'gate-summary.json').read_text());assert j['verdict'] in ['requires-bisect','certification-eligible']
Path('parent-summary.json').write_text(json.dumps(j,indent=2)+'\n')
PY
if [[ $(python3 -c 'import json;print(json.load(open("parent-summary.json"))["verdict"])') == certification-eligible ]]; then
 st not-needed 'parent certification eligible; no diagnostic games';exit 0
fi
python3 - <<'PY'
import datetime,hashlib,json,os,platform,subprocess
from pathlib import Path
p=Path.cwd();sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
d=os.statvfs('/mnt/c');assert d.f_bavail*d.f_frsize>=25.5*1024**3
original=json.loads((p/'original-game-manifest.json').read_text())
for f in ['baseline-ngn','candidate-ngn','base.nnue','sprt','openings.txt','candidate.patch','tests-receipt.json']:
 assert sha(p/f)==original['files_sha256'][f],f
common=['-tc','60+0.6','-concurrency','6','-mingames','1000000','-maxmoves','400','-gametimeout','900','-openings',str(p/'openings.txt'),'-lowpower=false']
def argv(kind,count):
 a=['taskset','-c','0-11','nice','-n','19',str(p/'sprt'),'-new',str(p/(kind+'-ngn')),'-base',str(p/'baseline-ngn'),*common,'-maxgames',str(count)]
 for side in ['new','base']:
  for n,v in [('EvalFile',str(p/'base.nnue')),('UseNNUE','true'),('Hash','64'),('Threads','1'),('OwnBook','false')]:a+=['-'+side+'option',n+'='+v]
 return a
names=['baseline-ngn','candidate-ngn','base.nnue','sprt','openings.txt','candidate.patch','tests-receipt.json','original-game-manifest.json','parent-summary.json','gate-parser.py','controller.sh','manifest.md']
hashes={n:sha(p/n) for n in names}
j=dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),record='experiments/2026-10-10-provenance-tt-bisect.md',purpose='diagnostic TT-only long-clock isolation; no playing adoption',base_source=original['accepted_runtime'],candidate_source='original source_snapshot '+original['source_snapshot']+' plus frozen own TT patch',tc='60+0.6',concurrency=6,cpus=list(range(12)),nice=19,environment={'GOMAXPROCS':'1','GOAMD64':'v3'},machine='Ryzen9800X3D/WSL2',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),memory_max_bytes=3*1024**3,profile_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',concurrent_workloads=['expanded training12,14/4GiB','time13,15/2500MiB','hopper untouched'],files_sha256=hashes,aa_argv=argv('baseline',40),gate_argv=argv('candidate',400),decision='fresh A/A40 contains0/zero faults; exactly400 paired; upper95<0 implicates TT, lower95>0 positive TT, otherwise unresolved; no source changes')
(p/'game-manifest.json').write_text(json.dumps(j,indent=2)+'\n');hashes['game-manifest.json']=sha(p/'game-manifest.json')
(p/'game-inputs.sha256').write_text(''.join(h+'  '+n+'\n' for n,h in hashes.items()))
PY
COMMON=(-tc 60+0.6 -concurrency 6 -mingames 1000000 -maxmoves 400 -gametimeout 900 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
st aa-start 'fresh TT-only diagnostic A/A40 at60+0.6/c6 on freed0-11'
timeout --signal=TERM --kill-after=15s 90m taskset -c 0-11 nice -n 19 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 gate-parser.py gate aa.log --gate-mode aa > aa-summary.json
sha256sum -c game-inputs.sha256 > input-check-after-aa.txt
st gate-start 'exactly400 paired TT-only diagnostic60+0.6; no adoption or extension'
timeout --signal=TERM --kill-after=15s 12h taskset -c 0-11 nice -n 19 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 gate-parser.py gate gate.log --gate-mode gate > gate-summary.json
python3 - <<'PY'
import json,re
from pathlib import Path
assert not re.search(r'PANIC|RECOVERED|FATAL',Path('ngn_crashes.log').read_text())
j=json.loads(Path('gate-summary.json').read_text());ci=j['exact_elo_ci95']
result={'result': 'TT-regression-implicated' if ci[2]<0 else 'TT-positive' if ci[1]>0 else 'TT-unresolved','exact_elo_ci95':ci,'playing_adoption':False,'gate_summary':j}
Path('diagnostic-summary.json').write_text(json.dumps(result,indent=2)+'\n')
PY
sha256sum -c game-inputs.sha256 > input-check-after.txt
st gate-done "$(cat diagnostic-summary.json)"
