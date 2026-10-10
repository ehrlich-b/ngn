#!/usr/bin/env bash
# Integration checks after the provisionally accepted board-key replacement.
set -euo pipefail
S=/work/lanes/ngn-s39-key-integration
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
cd "$S"
st() {
 python3 - "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open('status.jsonl','a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session39-key-integration',event=sys.argv[1],detail=sys.argv[2]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; compatibility only, no architecture adoption"; }
trap finish EXIT
date -Is > controller.started
sha256sum -c inputs.sha256 > input-check-before.txt
mkdir baseline source pre-batch
tar -xf base.tar -C baseline
tar -xf base.tar -C source
tar -xf pre-batch.tar -C pre-batch
(cd source && git apply ../architecture.patch)
st checks-start 'actual accepted key source; unchanged NGNN4/trainer; own 2.5GiB forge scope8-11'
cd source
for tag in avx2 purego; do
 flags=(); [[ $tag != purego ]] || flags=(-tags purego)
 timeout 15m taskset -c 8-11 nice -n 10 go test -p 2 "${flags[@]}" -run '^TestNGNN' -count=1 -v ./engine > "$S/differential-$tag.log" 2>&1
done
for mode in full-short full-race; do
 flags=(); [[ $mode != *race ]] || flags=(-race)
 timeout 30m taskset -c 8-11 nice -n 10 go test -p 2 -short "${flags[@]}" ./... -count=1 > "$S/$mode.log" 2>&1
done
timeout 15m taskset -c 8-11 nice -n 10 go vet -p 2 ./... > "$S/vet.log" 2>&1
timeout 15m taskset -c 8-11 nice -n 10 go build -p 2 -pgo="$S/source/default.pgo" -o "$S/candidate-ngn" . > "$S/candidate-build.log" 2>&1
timeout 15m taskset -c 8-11 nice -n 10 go build -p 2 -pgo="$S/source/default.pgo" -o "$S/candidate-probe" ./cmd/speedprobe > "$S/probe-build.log" 2>&1
cd "$S/baseline"
timeout 15m taskset -c 8-11 nice -n 10 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/baseline-probe" ./cmd/speedprobe > "$S/baseline-probe-build.log" 2>&1
cd "$S/pre-batch"
timeout 15m taskset -c 8-11 nice -n 10 go build -p 2 -pgo="$S/pre-batch/default.pgo" -o "$S/pre-batch-ngn" . > "$S/pre-batch-build.log" 2>&1
cd "$S"
export GOMAXPROCS=1
python3 - <<'PY'
import datetime,hashlib,json,os,platform,subprocess
from pathlib import Path
p=Path.cwd();sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
identity={}
for mode,net,depth in [('ngnn3',str(p/'base.nnue'),14),('ngnn2',str(p/'source/engine/nets/ngn-h512-c6c12796.nnue'),14),('hce','',11)]:
 rows={}
 for kind in ['baseline','candidate']:
  argv=['taskset','-c','8-11','nice','-n','10',str(p/(kind+'-probe')),'-net',net,'-mode','identity','-depth',str(depth)]
  run=subprocess.run(argv,check=True,text=True,capture_output=True,timeout=900)
  (p/f'identity-{mode}-{kind}.jsonl').write_text(run.stdout)
  assert not run.stderr
  rows[kind]=[{k:v for k,v in json.loads(line).items() if k not in ['seconds','nps']} for line in run.stdout.splitlines()]
 assert len(rows['baseline'])==12 and rows['baseline']==rows['candidate'],mode
 identity[mode]=dict(cases=12,depth=depth,equal=True)
files=['candidate-ngn','candidate-probe','baseline-probe','pre-batch-ngn','architecture.patch','base.tar','pre-batch.tar','source/default.pgo','source/engine/hash.go','source/engine/uci_process_test.go']
changed=[line.split(' b/',1)[1] for line in Path('architecture.patch').read_text().splitlines() if line.startswith('diff --git ')]
j=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),base_source='6c0fc26a17e359442098802f9f33ee673c118c90',architecture_parent='2acf834945d48378209853464b86cefe72587ccc',checks=['AVX2/purego NGNN tests','full-short ./...','full-race ./...','go vet ./...','explicit PGO UCI/probes','36 old-format exact identities on current keys'],identity=identity,machine='forge / Ryzen6900HX Debian13 VM',os=platform.platform(),go_version=subprocess.check_output(['go','version'],text=True).strip(),cpus=[8,9,10,11],nice=10,memory_max=2684354560,files_sha256={n:sha(p/n) for n in files},architecture_source_sha256={n:sha(p/'source'/n) for n in changed},trainer_unchanged=True,training_or_games=False,playing_adoption=False)
(p/'receipt.json').write_text(json.dumps(j,indent=2)+'\n')
PY
sha256sum -c inputs.sha256 > input-check-after.txt
st checks-pass '36 current-key old-format identities/full checks; unchanged trainer; games remain required'
