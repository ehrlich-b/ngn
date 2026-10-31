#!/usr/bin/env bash
# Combined TT/board validation, then the independent key-order gate.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session21
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session21-provenance',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > "$S/controller.exit"; date -Is > "$S/controller.done"; st end "rc $rc; review before source adoption"; }
trap finish EXIT
date -Is > controller.started
st capacity-wait 'C: must recover to32GiB; silent five-minute capacity checks,12h bound; no WSL shutdown or hopper interruption'
python3 - <<'PY'
import os,time
deadline=time.monotonic()+12*3600
while True:
 d=os.statvfs('/mnt/c')
 if d.f_bavail*d.f_frsize>=32*1024**3:break
 assert time.monotonic()<deadline,'C: capacity wait expired'
 time.sleep(300)
PY
st board-start 'TT-only base b49ab5a vs board39eb128; full short/race/vet and exact search identities on13,15'
mkdir -p board/baseline board/candidate key/candidate
tar -xf tt-base.tar -C board/baseline
tar -xf base.tar -C board/candidate
cp board-nodecheck.py board/nodecheck.py
for mode in full-short full-race; do
 flags=(); [[ $mode != *race ]] || flags=(-race)
 (cd board/candidate && taskset -c 13,15 nice -n 19 go test -p 2 -short "${flags[@]}" ./... -count=1) > "board/$mode.log" 2>&1
done
(cd board/candidate && taskset -c 13,15 nice -n 19 go vet -p 2 ./...) > board/vet.log 2>&1
for kind in baseline candidate; do
 (cd "board/$kind" && taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$S/board/baseline/default.pgo" -o "$S/board/$kind-probe" ./cmd/speedprobe) > "board/$kind-probe-build.log" 2>&1
 (cd "board/$kind" && taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$S/board/baseline/default.pgo" -o "$S/board/$kind-ngn" .) > "board/$kind-build.log" 2>&1
 GOMAXPROCS=1 taskset -c 13 nice -n 19 "$S/board/$kind-probe" -mode identity -depth 14 -net "$R/kb8h512/best.nnue" > "board/$kind-identity.jsonl"
 (cd "board/$kind" && GOMAXPROCS=1 taskset -c 13 nice -n 19 go test -p 1 -run '^TestPerftReferenceSuite$' -count=1 -v ./engine) > "board/$kind-perft.log" 2>&1
done
GOMAXPROCS=1 taskset -c 13 nice -n 19 python3 board/nodecheck.py > board/nodecheck.json
python3 - <<'PY'
import datetime,json,hashlib
from pathlib import Path
p=Path('board');rows={k:[json.loads(x) for x in (p/(k+'-identity.jsonl')).read_text().splitlines()] for k in ['baseline','candidate']}
fields=['position','kind','depth','nodes','score','best','pv']
assert len(rows['baseline'])==len(rows['candidate'])==12
assert all(all(a[f]==b[f] for f in fields) for a,b in zip(rows['baseline'],rows['candidate']))
assert json.loads((p/'nodecheck.json').read_text())['identical']
j=dict(passed=True,completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),base='b49ab5a',candidate='39eb128',fixed_depth_identities=12,nodecheck_identities=3,perft_positions=6,binaries={k:hashlib.sha256((p/(k+'-ngn')).read_bytes()).hexdigest() for k in ['baseline','candidate']})
(p/'receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
echo 0 > board.exit
date -Is > board.done
st board-pass '12+3 exact identities and full checks; neutral representation adopted over accepted TT'
tar -xf base.tar -C key/candidate
(cd key/candidate && git apply ../../key.patch)
st key-tests 'only own key seed/draw order changes; full checks on13,15'
for mode in full-short full-race; do
 flags=(); [[ $mode != *race ]] || flags=(-race)
 (cd key/candidate && taskset -c 13,15 nice -n 19 go test -p 2 -short "${flags[@]}" ./... -count=1) > "key/$mode.log" 2>&1
done
(cd key/candidate && taskset -c 13,15 nice -n 19 go vet -p 2 ./...) > key/vet.log 2>&1
(cd key/candidate && taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$S/board/baseline/default.pgo" -o "$S/key/candidate-ngn" .) > key/build.log 2>&1
cd key
cp ../board/candidate-ngn baseline-ngn
cp "$R/session19-tt-gate/sprt" sprt
cp "$R/session19-tt-gate/openings.txt" openings.txt
cp "$R/kb8h512/best.nnue" base.nnue
cp ../gate-parser.py gate-parser.py
export GOMAXPROCS=1
COMMON=(-tc 10+0.1 -concurrency 2 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$S/key/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/key/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/key/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
python3 - <<'PY'
import datetime,hashlib,json
from pathlib import Path
p=Path.cwd();sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
names=['baseline-ngn','candidate-ngn','sprt','openings.txt','base.nnue','gate-parser.py']
hashes={n:sha(p/n) for n in names};hashes['candidate/engine/hash.go']=sha(p/'candidate/engine/hash.go')
common=['-tc','10+0.1','-concurrency','2','-mingames','1000000','-maxmoves','400','-gametimeout','180','-openings',str(p/'openings.txt'),'-lowpower=false']
options=[]
for side in ['new','base']:
 for n,v in [('EvalFile',str(p/'base.nnue')),('UseNNUE','true'),('Hash','64'),('Threads','1'),('OwnBook','false')]:options+=['-'+side+'option',n+'='+v]
def argv(candidate,games):return ['taskset','-c','13,15','nice','-n','19',str(p/'sprt'),'-new',str(p/candidate),'-base',str(p/'baseline-ngn'),*common,'-maxgames',str(games),*options]
j=dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),record='experiments/2026-10-09-session21.md',base_source='39eb128',base_net=hashes['base.nnue'],candidate_patch=sha(p.parent/'key.patch'),tc='10+0.1',concurrency=2,cpus=[13,15],nice=19,environment={'GOMAXPROCS':'1','GOAMD64':'v3'},aa_argv=argv('baseline-ngn',40),gate_argv=argv('candidate-ngn',400),files_sha256=hashes,decision='400 games/200 pairs, upper95>0 and zero operational failures/intact inputs: provisional provenance keep; later combined batch SPRT [0,+6]; otherwise shelve',automatic_promotion=False)
(p/'game-manifest.json').write_text(json.dumps(j,indent=2)+'\n');hashes['game-manifest.json']=sha(p/'game-manifest.json')
(p/'inputs.sha256').write_text(''.join(h+'  '+n+'\n' for n,h in hashes.items()))
PY
sha256sum -c inputs.sha256 > input-check-before.txt
st key-aa-start 'A/A40 then400 paired games; CPU13,15; upper95>0 provenance rule; no automatic adoption'
timeout --signal=TERM --kill-after=15s 30m taskset -c 13,15 nice -n 19 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 gate-parser.py gate aa.log --gate-mode aa > aa-summary.json
st key-gate-start '400 paired games vs immediate39eb128/be34532c'
timeout --signal=TERM --kill-after=15s 3h taskset -c 13,15 nice -n 19 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 gate-parser.py gate gate.log --gate-mode gate > original-parser-summary.json
sha256sum -c inputs.sha256 > input-check-after.txt
python3 - <<'PY'
import json,re
from pathlib import Path
p=Path.cwd();j=json.loads((p/'original-parser-summary.json').read_text())
assert not re.search(r'PANIC|RECOVERED|FATAL',(p/'ngn_crashes.log').read_text())
j['verdict']='provisional-provenance-keep' if j['ci95'][1]>0 and j['exact_elo_ci95'][2]>0 else 'shelved'
(p/'gate-summary.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st key-done "$(cat gate-summary.json); review before adoption; SEE games remain unlaunched"
