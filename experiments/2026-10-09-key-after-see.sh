#!/usr/bin/env bash
# The existing key-order change, on a newly accepted SEE base only.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session32-key
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session32-key',event=sys.argv[2],detail=sys.argv[3]))+'\n')
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
assert j['see_reviewed'] and j['see_verdict']=='provisional-provenance-keep'
assert j['base_source']!='39eb128' and j['candidate_change']=='existing-key-order-only'
d=os.statvfs('/mnt/c'); assert d.f_bavail*d.f_frsize>=27*1024**3
PY
mkdir candidate
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch)
st tests-start 'changed immediate SEE base; existing key patch only; full short/race/vet on13,15'
for mode in full-short full-race; do
 flags=(); [[ $mode != *race ]] || flags=(-race)
 (cd candidate && taskset -c 13,15 nice -n 19 go test -p 2 -short "${flags[@]}" ./... -count=1) > "$mode.log" 2>&1
done
(cd candidate && taskset -c 13,15 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
(cd candidate && taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$S/candidate/default.pgo" -o "$S/candidate-ngn" .) > build.log 2>&1
export GOMAXPROCS=1
COMMON=(-tc 10+0.1 -concurrency 2 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
p=Path.cwd();sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
nom=json.loads((p/'nomination.json').read_text())
names=['baseline-ngn','candidate-ngn','sprt','openings.txt','base.nnue','gate-parser.py','candidate.patch','base.tar','nomination.json','controller.sh']
hashes={n:sha(p/n) for n in names}
sources=['engine/hash.go','engine/piece.go','engine/see_test.go','engine/independent_piece_test.go']
source_hashes={f:sha(p/'candidate'/f) for f in sources}
assert hashes['baseline-ngn']==nom['baseline_binary_sha256']
assert hashes['base.nnue']==nom['base_net_sha256']
assert hashes['candidate.patch']==nom['candidate_patch_sha256']
common=['-tc','10+0.1','-concurrency','2','-mingames','1000000','-maxmoves','400','-gametimeout','180','-openings',str(p/'openings.txt'),'-lowpower=false']
options=[]
for side in ['new','base']:
 for n,v in [('EvalFile',str(p/'base.nnue')),('UseNNUE','true'),('Hash','64'),('Threads','1'),('OwnBook','false')]: options+=['-'+side+'option',n+'='+v]
def argv(candidate,games): return ['taskset','-c','13,15','nice','-n','19',str(p/'sprt'),'-new',str(p/candidate),'-base',str(p/'baseline-ngn'),*common,'-maxgames',str(games),*options]
j=dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),record='experiments/2026-10-09-key-after-see.md',base_source=nom['base_source'],candidate_topic=nom['candidate_topic'],candidate_source_sha256=source_hashes,tc='10+0.1',concurrency=2,cpus=[13,15],nice=19,environment={'GOMAXPROCS':'1','GOAMD64':'v3'},machine='Ryzen9800X3D / WSL2',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),profile_sha256=sha(p/'candidate/default.pgo'),aa_argv=argv('baseline-ngn',40),gate_argv=argv('candidate-ngn',400),files_sha256=hashes,decision='A/A40 complete20 pairs, printed/recomputed paired CI contains0 and zero faults; then exactly400 games/200 pairs, both upper95>0 and zero flags/crashes/illegal/no-move/intact inputs: provisional provenance keep; otherwise shelve; combined batch SPRT [0,+6] remains required; no retries/extensions',automatic_promotion=False)
(p/'game-manifest.json').write_text(json.dumps(j,indent=2)+'\n')
hashes['game-manifest.json']=sha(p/'game-manifest.json')
(p/'inputs.sha256').write_text(''.join(h+'  '+n+'\n' for n,h in hashes.items()))
PY
sha256sum -c inputs.sha256 > input-check-before.txt
st aa-start 'new SEE immediate base; one frozen A/A40 on13,15; original failed key preflight preserved'
timeout --signal=TERM --kill-after=15s 30m taskset -c 13,15 nice -n 19 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 gate-parser.py gate aa.log --gate-mode aa > aa-summary.json
st gate-start 'exactly400 paired10+0.1 games vs immediate SEE base; upper95>0 provenance rule'
timeout --signal=TERM --kill-after=15s 3h taskset -c 13,15 nice -n 19 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 gate-parser.py gate gate.log --gate-mode gate > original-parser-summary.json
sha256sum -c inputs.sha256 > input-check-after.txt
sha256sum -c frozen-inputs.sha256 > frozen-check-after.txt
python3 - <<'PY'
import json,re
from pathlib import Path
j=json.loads(Path('original-parser-summary.json').read_text())
assert not re.search(r'PANIC|RECOVERED|FATAL',Path('ngn_crashes.log').read_text())
j['verdict']='provisional-provenance-keep' if j['ci95'][1]>0 and j['exact_elo_ci95'][2]>0 else 'shelved'
Path('gate-summary.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st gate-done "$(cat gate-summary.json); no automatic adoption; combined provenance certification remains required"
