#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session15h
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
export PATH=/usr/local/go/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$S/go-cache
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 CUDA_VISIBLE_DEVICES=''
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='ngnn4-cpu',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
cleanup() {
  local rc=$?
  echo "$rc" > "$S/controller.exit"
  date -Is > "$S/controller.done"
  st done "rc $rc; topic candidate only; no net or runtime installed"
}
trap cleanup EXIT
date -Is > "$S/controller.started"
cd "$S"
sha256sum -c frozen-inputs.sha256 > input-check-before.txt
st wait 'native pidfd 598985; do not overlap the deferred fixed speed sample'
python3 - <<'PY'
import os,select
try:
    fd=os.pidfd_open(598985)
except ProcessLookupError:
    pass
else:
    poll=select.poll();poll.register(fd,select.POLLIN);poll.poll();os.close(fd)
PY
st start "pid $$; CPU 13,15; CPU-only hidden-layer tests and parity"
mkdir source
tar -xf base.tar -C source
cd source
git apply ../candidate.patch
export PYTHONPATH=$S/source
taskset -c 13,15 nice -n 19 "$PY" -m trainer.make_random_net --format 4 --hidden 16 --hidden2 16 --buckets 8 --seed 20261008 --out testdata/ngnn4/random_h16_l16_k8_b8.nnue --evals testdata/ngnn4/random_h16_l16_k8_b8_evals.json > "$S/fixture.json" 2> "$S/fixture.err"
taskset -c 13,15 nice -n 19 "$PY" -m unittest trainer.tests.test_ngnn4 -v > "$S/trainer-narrow.log" 2>&1
taskset -c 13,15 nice -n 19 "$PY" -m unittest discover -s trainer/tests -v > "$S/trainer-full.log" 2>&1
st trainer-pass 'new hidden layer and full CPU trainer suite; CUDA checks intentionally skipped'
taskset -c 13,15 nice -n 19 go test -p 2 ./engine -run '^TestNGNN4' -count=1 -v > "$S/ngnn4-test.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -tags purego ./engine -run '^TestNGNN4' -count=1 -v > "$S/ngnn4-purego.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short ./engine -count=1 > "$S/engine-short.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short -race ./engine -count=1 > "$S/engine-race.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short ./... -count=1 > "$S/full-short.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 -short -race ./... -count=1 > "$S/full-race.log" 2>&1
taskset -c 13,15 nice -n 19 go vet -p 2 ./engine > "$S/vet.log" 2>&1
taskset -c 13,15 nice -n 19 go test -p 2 ./engine -run '^$' -bench '^BenchmarkNGNN(1|4)Output$' -benchtime 1s -count=3 > "$S/inference-bench.log" 2>&1
taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$S/source/default.pgo" -o "$S/ngn-hidden" . > "$S/build.log" 2>&1
sha256sum -c "$S/frozen-inputs.sha256" > "$S/input-check-after.txt"
st tests-pass 'new AVX2/Go/scalar parity, engine/full short/race, vet and build; no strength verdict'
python3 - "$S" <<'PY'
import hashlib,json,sys
from pathlib import Path
s=Path(sys.argv[1]); digest=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
(s/'receipt.json').write_text(json.dumps(dict(tests_pass=True,cuda_executed=False,
    patch_sha256=digest(s/'candidate.patch'),binary_sha256=digest(s/'ngn-hidden'),
    fixture=json.loads((s/'fixture.json').read_text()),cpu=[13,15],nice=19,
    candidate_accepted=False,training_or_games_queued=False),indent=2)+'\n')
PY
