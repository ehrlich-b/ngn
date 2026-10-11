#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session61-dead-pst
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
cd "$S"
st() { python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session61-dead-pst',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > "$S/controller.exit"; date -Is > "$S/controller.done"; st end "rc$rc; review required; no automatic adoption"; }
trap finish EXIT
date -Is > controller.started
# The preceding finite check job has its exact handle; pidfd is a native wait.
python3 - <<'PY'
import os,select
from pathlib import Path
p=Path('/home/ehrli/ngn-data/night-20261007/session61-correctness')
if not (p/'checks.done').exists():
 fd=os.pidfd_open(int((p/'checks.pid').read_text()));poll=select.poll();poll.register(fd,select.POLLIN)
 assert poll.poll(3600000),'correctness checks wait expired'
 os.close(fd)
assert (p/'checks.exit').read_text().strip()=='0'
PY
st checks-start 'lower-priority exact speed on spare13,15; separately complete correctness fullrace first'
for c in see singular; do
 (cd "$R/session61-correctness/$c" && timeout 20m go test -p 2 -short -race ./... -count=1) > "$R/session61-correctness/$c-full-race.log" 2>&1
 done
date -Is > "$R/session61-correctness/full-race.done"
mkdir baseline candidate
tar xf "$R/session61-correctness/base.tar" -C baseline
tar xf "$R/session61-correctness/base.tar" -C candidate
(cd candidate && git apply ../candidate.patch && gofmt -w engine/bitboard.go engine/eval_accumulator_test.go engine/evaluation_core_test.go engine/private_board_proof_test.go)
(cd candidate && NGN_PRIVATE_BOARD_DUMP="$S/board-invariants.bin" timeout 10m go test -p 2 -short -run '^Test(PrivateBoard|EvalAccumulator|Evaluation|RebuildEquivalence|HCE|NGNN)' -count=1 -v ./engine) > narrow.log 2>&1
(cd candidate && timeout 20m go test -p 2 -short ./engine -count=1) > engine-short.log 2>&1
(cd candidate && timeout 20m go test -p 2 -short -race ./engine -count=1) > engine-race.log 2>&1
(cd candidate && timeout 20m go test -p 2 -short ./... -count=1) > full-short.log 2>&1
(cd candidate && timeout 20m go test -p 2 -short -race ./... -count=1) > full-race.log 2>&1
(cd candidate && timeout 10m go vet -p 2 ./...) > vet.log 2>&1
for kind in baseline candidate; do
 (cd "$kind" && timeout 10m go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-probe" ./cmd/speedprobe) > "$kind-build.log" 2>&1
 done
(cd candidate && timeout 10m go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/candidate-ngn" .) > candidate-ngn-build.log 2>&1
cp "$R/session56-time-tt-games/base.nnue" base.nnue
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
files=['engine/bitboard.go','engine/eval_accumulator_test.go','engine/evaluation_core_test.go','engine/private_board_proof_test.go']
j=dict(base_source='ed34480',utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),source_archive_sha256=sha(r.parent/'session61-correctness/base.tar'),patch_sha256=sha(r/'candidate.patch'),source_sha256={f:sha(r/'candidate'/f) for f in files},binary_sha256={k:sha(r/f'{k}-probe') for k in ['baseline','candidate']},uci_sha256=sha(r/'candidate-ngn'),profile_sha256=sha(r/'baseline/default.pgo'),old_net_sha256=sha(r/'baseline/engine/nets/ngn-h512-c6c12796.nnue'),net_sha256=sha(r/'base.nnue'),go_version=subprocess.check_output(['go','version'],text=True).strip(),goamd64='v3',os=platform.platform(),machine='rig WSL2/Ryzen9800X3D',build_test_cpus=[13,15],nice=19,tests_pass=True,memory_max=2147483648,checks=['private board invariant walk','phase/make-unmake/HCE/NNUE controls','engine/full short/race','vet','explicit accepted-PGO builds'],strength_games=0)
(r/'build-receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st timing-start '36 exact identities then fixed40 balanced blocks; cpu15/currentnet; preserve all outcomes'
set +e
python3 speed.py > speed.log 2>&1
rc=$?
set -e
if [ "$rc" = 3 ]; then st speed-shelved 'fixed speed rule missed; no extension'; elif [ "$rc" != 0 ]; then exit "$rc"; else st speed-eligible 'manual complete evidence review required'; fi
