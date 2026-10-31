#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session35-factorizer
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session35-factorizer',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; review before promotion; no retry or extension"; }
trap finish EXIT
date -Is > controller.started
sha256sum -c frozen-inputs.sha256 > frozen-check-before.txt
python3 - <<'PY'
import os
d=os.statvfs('/mnt/c'); assert d.f_bavail*d.f_frsize>=25.5*1024**3
PY
mkdir source
tar -xf base.tar -C source
st build-start 'clean accepted ac658e2 source;13,15; explicit original PGO; no behavior overlay'
(cd source && taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$S/source/default.pgo" -o "$S/ngn" .) > build.log 2>&1
export GOMAXPROCS=1
COMMON=(-tc 10+0.1 -concurrency 2 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
hashes={n:sha(r/n) for n in ['ngn','sprt','openings.txt','base.nnue','candidate.nnue','gate-parser.py','base.tar','controller.sh','manifest.md']}
assert hashes['base.nnue']=='be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
assert hashes['candidate.nnue']=='d55a5096f1ad3882a806d2af8d2c699f275f087f8cacd6070cd60b3a41e9f2ce'
assert hashes['sprt']=='9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a'
assert hashes['openings.txt']=='974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'
common=['-tc','10+0.1','-concurrency','2','-mingames','1000000','-maxmoves','400','-gametimeout','180','-openings',str(r/'openings.txt'),'-lowpower=false']
def argv(games,net):
 a=['taskset','-c','13,15','nice','-n','19',str(r/'sprt'),'-new',str(r/'ngn'),'-base',str(r/'ngn'),*common,'-maxgames',str(games)]
 for side,file in [('new',net),('base','base.nnue')]:
  for n,v in [('EvalFile',str(r/file)),('UseNNUE','true'),('Hash','64'),('Threads','1'),('OwnBook','false')]:a+=['-'+side+'option',n+'='+v]
 return a
j=dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),base_source='ac658e2ce509154775d9567807a5ccf8485b6dca',candidate_change='factorizer-trained export only',tc='10+0.1',concurrency=2,cpus=[13,15],nice=19,environment={'GOMAXPROCS':'1','GOAMD64':'v3'},machine='Ryzen9800X3D / WSL2',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),profile_sha256=sha(r/'source/default.pgo'),aa_argv=argv(40,'base.nnue'),gate_argv=argv(400,'candidate.nnue'),files_sha256=hashes,decision='complete A/A40 paired CI contains0/zero faults; complete400 gate printed and unrounded lower95>0/zero faults/intact inputs; otherwise shelve without extension',automatic_promotion=False)
(r/'game-manifest.json').write_text(json.dumps(j,indent=2)+'\n')
hashes['game-manifest.json']=sha(r/'game-manifest.json')
(r/'inputs.sha256').write_text(''.join(h+'  '+n+'\n' for n,h in hashes.items()))
PY
sha256sum -c inputs.sha256 > input-check-before.txt
st aa-start 'fresh actual-base A/A40;13,15/concurrency2; SPSA unchanged0–11'
timeout --signal=TERM --kill-after=15s 30m taskset -c 13,15 nice -n 19 ./sprt -new ./ngn -base ./ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 gate-parser.py gate aa.log --gate-mode aa > aa-summary.json
NEW[1]="EvalFile=$S/candidate.nnue"
st gate-start 'exactly400 paired10+0.1; factorizer only; lower95>0 rule; no extension'
timeout --signal=TERM --kill-after=15s 4h taskset -c 13,15 nice -n 19 ./sprt -new ./ngn -base ./ngn "${COMMON[@]}" -maxgames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 gate-parser.py gate gate.log --gate-mode gate > gate-summary.json
python3 - <<'PY'
import re
from pathlib import Path
assert not re.search(r'PANIC|RECOVERED|FATAL',Path('ngn_crashes.log').read_text())
PY
sha256sum -c inputs.sha256 > input-check-after.txt
sha256sum -c frozen-inputs.sha256 > frozen-check-after.txt
st gate-done "$(cat gate-summary.json); review before promotion"
