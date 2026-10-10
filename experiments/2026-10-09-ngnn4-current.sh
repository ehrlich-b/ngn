#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session34-architecture
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 PYTHONDONTWRITEBYTECODE=1 CUDA_VISIBLE_DEVICES=''
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session34-architecture',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; compatibility only, no training or playing adoption"; }
trap finish EXIT
date -Is > controller.started
sha256sum -c inputs.sha256 > input-check-before.txt
mkdir source
tar -xf base.tar -C source
(cd source && git apply ../candidate.patch)
export PYTHONPATH=$S/source
st checks-start 'retained512 + current playing null frames + reviewed batched trainer;12,14/nice19'
cd source
taskset -c 12,14 nice -n 19 "$PY" -m unittest trainer.tests.test_ngnn4 trainer.tests.test_ngnn4_batched -v > "$S/trainer-narrow.log" 2>&1
taskset -c 12,14 nice -n 19 "$PY" -m unittest discover -s trainer/tests -v > "$S/trainer-full.log" 2>&1
for tag in avx2 purego; do
 flags=(); [[ $tag != purego ]] || flags=(-tags purego)
 taskset -c 12,14 nice -n 19 go test -p 2 "${flags[@]}" -run '^TestNGNN' -count=1 -v ./engine > "$S/differential-$tag.log" 2>&1
done
for mode in full-short full-race; do
 flags=(); [[ $mode != *race ]] || flags=(-race)
 taskset -c 12,14 nice -n 19 go test -p 2 -short "${flags[@]}" ./... -count=1 > "$S/$mode.log" 2>&1
done
taskset -c 12,14 nice -n 19 go vet -p 2 ./... > "$S/vet.log" 2>&1
taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/source/default.pgo" -o "$S/candidate-ngn" . > "$S/build.log" 2>&1
taskset -c 12,14 nice -n 19 go build -p 2 -pgo="$S/source/default.pgo" -o "$S/candidate-probe" ./cmd/speedprobe > "$S/probe-build.log" 2>&1
cd "$S"
export GOMAXPROCS=1
python3 - <<'PY'
import hashlib,json,os,subprocess
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
assert sha(r/'baseline-probe')=='9a47183e79f585fd973ed324b00fa1a0c6e0e25fc14d68875a1b659bb897ad40'
identity={}
strip=lambda row:{k:v for k,v in row.items() if k not in ['seconds','nps']}
for mode,net,depth in [('ngnn3',str(r/'base.nnue'),14),('ngnn2',str(r/'source/engine/nets/ngn-h512-c6c12796.nnue'),14),('hce','',11)]:
 rows={}
 for kind in ['baseline','candidate']:
  argv=['taskset','-c','14','nice','-n','19',str(r/f'{kind}-probe'),'-net',net,'-mode','identity','-depth',str(depth)]
  p=subprocess.run(argv,check=True,text=True,capture_output=True,timeout=600)
  (r/f'identity-{mode}-{kind}.jsonl').write_text(p.stdout)
  assert not p.stderr
  rows[kind]=[strip(json.loads(line)) for line in p.stdout.splitlines()]
 assert len(rows['baseline'])==12 and rows['baseline']==rows['candidate'],mode
 identity[mode]=dict(cases=12,depth=depth,equal=True)
(r/'identity.json').write_text(json.dumps(identity,indent=2)+'\n')
PY
st identity-pass '36 old-format exact identities vs ac658e2 checked playing probe; no repeated timing sample'
unset CUDA_VISIBLE_DEVICES
cd source
taskset -c 12,14 nice -n 19 "$PY" -m unittest \
 trainer.tests.test_ngnn4.NGNN4Tests.test_cuda_fp32_bf16_gradients_and_fused_step \
 trainer.tests.test_ngnn4_batched.NGNN4BatchedTests.test_cuda_fp32_bf16_forward_backward_all_bucket_counts -v > "$S/cuda.log" 2>&1
cd "$S"
python3 - <<'PY'
import datetime,hashlib,json,platform,subprocess
from pathlib import Path
r=Path.cwd();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
log=(r/'cuda.log').read_text();assert 'OK' in log and 'skipped' not in log
files=[line.split(' b/',1)[1] for line in (r/'candidate.patch').read_text().splitlines() if line.startswith('+++ b/')]
j=dict(base_source='ac658e2',architecture_source='e0f4fb9',tests_pass=True,cuda_no_skips=True,
 utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),go_version=subprocess.check_output(['go','version'],text=True).strip(),
 os=platform.platform(),GOAMD64='v3',cpu=[12,14],nice=19,source_sha256={p:sha(r/'source'/p) for p in files},
 binary_sha256={p:sha(r/p) for p in ['baseline-probe','candidate-probe','candidate-ngn']},
 old_format_identity=json.loads((r/'identity.json').read_text()),trainer='reviewed batched-affine; dense transformer',
 training_or_games_queued=False,playing_adoption=False)
(r/'receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
sha256sum -c inputs.sha256 > input-check-after.txt
st checks-pass 'engine/trainer/CUDA + old-format integration complete; isolated topic only'
