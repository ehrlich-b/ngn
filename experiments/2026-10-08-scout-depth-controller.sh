#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session18d
export PATH=/usr/local/go/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=$R/session17f/go-cache
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='scout-depth',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
  local rc=$?
  echo "$rc" > "$S/controller.exit"
  date -Is > "$S/controller.done"
  if [[ $rc != 0 ]]; then st fail "rc $rc; preserve results; no retry or verdict"; fi
}
trap finish EXIT
cd "$S"
date -Is > controller.started
st tests-start "pid $$; CPU13,15; candidate short/race suites; live jobs untouched"
sha256sum -c frozen-inputs.sha256 > input-check-before.txt
cd candidate
taskset -c 13,15 nice -n 19 go test -p 2 -short ./engine -count=1 > "$S/engine-short.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short -race ./engine -count=1 > "$S/engine-race.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short ./... -count=1 > "$S/full-short.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short -race ./... -count=1 > "$S/full-race.log" 2>&1
taskset -c 13,15 nice -n 19 go vet -p 2 ./engine > "$S/vet.log" 2>&1
for kind in baseline candidate; do
  cd "$S/$kind"
  taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-ngn" . > "$S/$kind-build.log" 2>&1
done
cd "$S"
export GOMAXPROCS=1
COMMON=(-tc 10+0.1 -concurrency 2 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$S/openings.txt" -lowpower=false)
NEW_OPTS=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE_OPTS=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
python3 - <<'PY'
import datetime,hashlib,json,subprocess
from pathlib import Path
r=Path.cwd();digest=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
assert digest(r/'base.nnue')=='be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
assert digest(r/'openings.txt')=='974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'
assert digest(r/'sprt')=='9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a'
assert digest(r/'baseline/default.pgo')=='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0'
opts=lambda k:sum(([f'-{k}option',o] for o in [f'EvalFile={r}/base.nnue','UseNNUE=true','Hash=64','Threads=1','OwnBook=false']),[])
common=['-tc','10+0.1','-concurrency','2','-mingames','1000000','-maxmoves','400','-gametimeout','180','-openings',str(r/'openings.txt'),'-lowpower=false']
argv=lambda new,n:['taskset','-c','13,15','nice','-n','19',str(r/'sprt'),'-new',str(r/f'{new}-ngn'),'-base',str(r/'baseline-ngn'),*common,'-maxgames',str(n),*opts('new'),*opts('base')]
files=['base.tar','candidate.patch','baseline-ngn','candidate-ngn','base.nnue','openings.txt','sprt','baseline/default.pgo','controller.sh','gate-parser.py']
manifest=dict(predeclared_record='experiments/2026-10-08-scout-depth.md',created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),accepted_runtime='5b882e2',source_commit=(r/'base-commit.txt').read_text().strip(),binary_sha256={k:digest(r/f'{k}-ngn') for k in ('baseline','candidate')},files_sha256={p:digest(r/p) for p in files},tc='10+0.1',concurrency=2,cpus=[13,15],nice=19,environment={'GOMAXPROCS':'1','GOAMD64':'v3'},go_version=subprocess.check_output(['go','version'],text=True).strip(),uname=subprocess.check_output(['uname','-a'],text=True).strip(),build_argv=['go','build','-p','2',f'-pgo={r}/baseline/default.pgo','-o','OUTPUT','.'],aa_argv=argv('baseline',40),gate_argv=argv('candidate',400),tests_pass=True,decision='complete printed/recomputed penta lower95>0, zero operational failures, intact inputs; otherwise shelve without extension')
(r/'game-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
(r/'game-inputs.sha256').write_text(''.join(f'{digest(r/p)}  {p}\n' for p in files+['game-manifest.json']))
PY
st aa-start '40 A/A games; CPU13,15 / nice19 / concurrency2; manifest frozen'
timeout --signal=TERM --kill-after=10s 30m taskset -c 13,15 nice -n 19 "$S/sprt" -new "$S/baseline-ngn" -base "$S/baseline-ngn" "${COMMON[@]}" -maxgames 40 "${NEW_OPTS[@]}" "${BASE_OPTS[@]}" > aa.log 2>&1
python3 gate-parser.py gate aa.log --gate-mode aa > aa-summary.json
st gate-start '400 paired games; accept only complete lower95>0 and zero operational failures'
timeout --signal=TERM --kill-after=10s 150m taskset -c 13,15 nice -n 19 "$S/sprt" -new "$S/candidate-ngn" -base "$S/baseline-ngn" "${COMMON[@]}" -maxgames 400 "${NEW_OPTS[@]}" "${BASE_OPTS[@]}" > gate.log 2>&1
python3 gate-parser.py gate gate.log --gate-mode gate > gate-summary.json
sha256sum -c game-inputs.sha256 > input-check-after.txt
st done "$(cat gate-summary.json); review before any source promotion; no automatic merge"
