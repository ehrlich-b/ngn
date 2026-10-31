#!/usr/bin/env bash
# One from-scratch factorizer net, then exact parity, fresh A/A40 and gate400.
# Launch only after a committed reviewed nomination; never promotes a net.
set -euo pipefail
[[ $# == 0 || ( $# == 1 && $1 == --dry-run ) ]] || exit 2
R=/home/ehrli/ngn-data/night-20261007
S=$R/session9b
SP=$R/spsa/kb8
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
PAUSED=0
export PYTHONPATH=$S/source PYTHONDONTWRITEBYTECODE=1
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 GOMAXPROCS=1
cd "$S"
sha256sum -c trial-inputs.sha256 > trial-input-check-before.txt
python3 trial.py plan "$S" > validated-plan.json
if [[ ${1:-} == --dry-run ]]; then exit 0; fi
[[ $(cat prepare.exit) == 0 && -s activation-commit.txt ]]
sha256sum -c activation-inputs.sha256 > activation-input-check-before.txt
python3 - <<'PY'
import json,re
from pathlib import Path
r=Path.cwd(); commit=(r/'activation-commit.txt').read_text().strip()
assert re.fullmatch('[0-9a-f]{40}',commit)
assert json.loads((r/'nomination.json').read_text())['review_commit']==commit
PY
exec 9> "$R/factorizer-trial.lock"
flock -n 9
[[ ! -e controller.started && ! -e candidate ]]
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='factorizer-trial',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
capacity() {
  local free_bytes
  free_bytes=$(df -B1 --output=avail /mnt/c | tail -n 1 | tr -d ' ')
  [[ $free_bytes =~ ^[0-9]+$ && $free_bytes -ge 26843545600 ]]
  printf '%s\n' "$free_bytes" > "host-c-free-$1.txt"
}
unoccupied() {
  if pgrep -af 'trainer\.train|/datagen |/claimtool |/sprt |cutechess|fastchess|baseline-probe|candidate-probe|/speedprobe ' > "blocked-$1.txt"; then
    st fail 'another evidence job owns the window; no new work'; return 1
  fi
}
resume() {
  if [[ $PAUSED == 1 ]]; then
    # A completed tune is never restarted; unfinished old-base tuning stays separate.
    if python3 - "$SP/spsa.jsonl" <<'PY'
import json,sys
from pathlib import Path
assert json.loads(Path(sys.argv[1]).read_text().splitlines()[-1])['iter']<2000
PY
    then
      cd "$SP"
      setsid nohup bash "$R/session1/spsa-resume.cmd" >> "$SP/spsa.log" 2>&1 < /dev/null 9>&- &
      echo $! > "$SP/spsa.pid"
      cd "$S"
      st spsa-resume "pid $(cat "$SP/spsa.pid"); frozen old-base tuner; no transfer to a promoted net"
    fi
    PAUSED=0
  fi
}
finish() {
  local rc=$?
  resume
  printf '%s\n' "$rc" > "$S/controller.exit"
  date -Is > "$S/controller.done"
  if [[ $rc != 0 ]]; then st fail "rc $rc; preserve artifacts, no retry or extension"; fi
}
trap finish EXIT
date -Is > controller.started
capacity before-train
unoccupied before-train
unset CUDA_VISIBLE_DEVICES
taskset -c 12,14 nice -n 19 "$PY" -c 'import torch; assert torch.cuda.is_available()' > cuda-available.log 2>&1
st data-check 'verify the immutable T80 archive before from-scratch training'
printf '%s\n' '270f65bdf9f5c948160cae5330030a60f99e94383de2a290b565ae7aad7cde26  /home/ehrli/nnue-owned-k4-20260920/archive-a1/corpus-a1/train-archive.bf' > data-input.sha256
taskset -c 13,15 nice -n 19 sha256sum -c data-input.sha256 > data-input-check.txt
st train-start "pid $$; only --king-factorizer changes the reviewed base recipe; 24h timeout"
timeout --signal=TERM --kill-after=30s 24h taskset -c 12,14 nice -n 19 "$PY" trial.py train "$S" > train.log 2>&1
st train-done 'full declared schedule complete; lowest-validation checkpoint proceeds to parity, no loss-based strength verdict'
export CUDA_VISIBLE_DEVICES=''
taskset -c 13,15 nice -n 19 "$PY" trial.py parity "$S" > parity.json 2> parity.err
NGN_NGNN3_PARITY_NET="$S/candidate/best.nnue" NGN_NGNN3_PARITY_EVALS="$S/parity-evals.json" \
  taskset -c 13,15 nice -n 19 "$S/frozen/engine.test" -test.run '^TestNGNN3TrainerParity$' -test.v > parity.log 2>&1
st parity-done '20 exact engine/reference cases and byte-identical folded reexport'
sha256sum candidate/best.nnue candidate/best.pt parity.json parity-evals.json > candidate-inputs.sha256
sha256sum -c trial-inputs.sha256 > trial-input-check-pre-games.txt
sha256sum -c activation-inputs.sha256 > activation-input-check-pre-games.txt
python3 trial.py plan "$S" > pre-game-plan.json
cmp validated-plan.json pre-game-plan.json
pid=$(cat "$SP/spsa.pid")
state=$(ps -p "$pid" -o stat= 2>/dev/null || true)
if [[ -n $state && $state != Z* ]]; then
  [[ $(ps -p "$pid" -o pgid= | tr -d ' ') == "$pid" ]]
  ps -p "$pid" -o args= | rg -F "$R/session1/spsa-resume.cmd" > tuner-before.txt
  minimum=$(python3 - "$SP/spsa.jsonl" <<'PY'
import json,sys
from pathlib import Path
i=json.loads(Path(sys.argv[1]).read_text().splitlines()[-1])['iter']
print(min(2000,(i//25+1)*25))
PY
)
  st checkpoint-wait "native inotify/pidfd; tuner $pid; iteration >=$minimum; 90min timeout"
  python3 checkpoint-wait.py "$SP/spsa.jsonl" "$pid" "$minimum" 5400 > checkpoint-ready.json
  [[ $(cat "$SP/spsa.pid") == "$pid" ]]
  cp "$SP/spsa.jsonl" tuner-before.jsonl
  PAUSED=1
  kill -TERM -- "-$pid"
  for ((i=0; i<30; i++)); do
    if ! ps -eo pgid=,stat= | awk -v p="$pid" '$1==p && $2 !~ /^Z/ { found=1 } END { exit !found }'; then break; fi
    sleep 1
  done
  if ps -eo pgid=,stat= | awk -v p="$pid" '$1==p && $2 !~ /^Z/ { found=1 } END { exit !found }'; then
    PAUSED=0; st fail 'tuner did not drain; no games or duplicate tuner'; exit 1
  fi
fi
unoccupied pre-games
if pgrep -af '/spsa -engine' > blocked-tuner.txt; then exit 1; fi
capacity pre-games
BASE=$(python3 -c 'import json; print(json.load(open("validated-plan.json"))["base_net"])')
COMMON=(-tc 10+0.1 -concurrency 8 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$S/frozen/openings.txt" -lowpower=false)
NEW_OPTS=(-newoption "EvalFile=$BASE" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE_OPTS=(-baseoption "EvalFile=$BASE" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
st aa-start 'exactly 40 same-base games, CPUs 0-11 / nice 19 / GOMAXPROCS=1'
timeout --signal=TERM --kill-after=30s 30m taskset -c 0-11 nice -n 19 "$S/frozen/sprt" -new "$S/frozen/ngn" -base "$S/frozen/ngn" "${COMMON[@]}" -maxgames 40 "${NEW_OPTS[@]}" "${BASE_OPTS[@]}" > aa.log 2>&1
python3 trial.py gate aa.log --gate-mode aa > aa-summary.json
capacity pre-gate
NEW_OPTS[1]="EvalFile=$S/candidate/best.nnue"
st gate-start 'exactly 400 paired games; lower95 >0 and zero operational failures; no extension'
timeout --signal=TERM --kill-after=30s 2h taskset -c 0-11 nice -n 19 "$S/frozen/sprt" -new "$S/frozen/ngn" -base "$S/frozen/ngn" "${COMMON[@]}" -maxgames 400 "${NEW_OPTS[@]}" "${BASE_OPTS[@]}" > gate.log 2>&1
python3 trial.py gate gate.log --gate-mode gate > gate-summary.json
sha256sum -c candidate-inputs.sha256 > candidate-input-check-after.txt
sha256sum -c trial-inputs.sha256 > trial-input-check-after.txt
sha256sum -c activation-inputs.sha256 > activation-input-check-after.txt
python3 trial.py plan "$S" > post-game-plan.json
cmp validated-plan.json post-game-plan.json
st done "$(cat gate-summary.json); review before promotion; no strength screen or claim queued"
