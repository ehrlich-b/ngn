#!/usr/bin/env bash
set -euo pipefail
s=$1
cd "$s"
st() { printf '{"utc":"%s","stage":"session69-clock-diagnostic","event":"%s","detail":"%s"}\n' "$(date -u +%FT%TZ)" "$1" "$2" >> "$s/status.jsonl"; }
finish() { local rc=$?; echo "$rc" > "$s/controller.exit"; date -Is > "$s/controller.done"; st terminal "exit $rc"; }
trap finish EXIT
date -Is > controller.started
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/mod
export GOMAXPROCS=2 GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off PYTHONDONTWRITEBYTECODE=1
st checks-start 'tool-only TC flags; forge4-11/3GiB/swap0/nice10'
cd source
gofmt -w cmd/pairmatch/main.go cmd/pairmatch/main_test.go
go version > ../environment.txt
timeout 10m go test -p 2 -short ./internal/uci ./cmd/pairmatch -count=1 > ../short.log 2>&1
timeout 10m go test -p 2 -short -race ./internal/uci ./cmd/pairmatch -count=1 > ../race.log 2>&1
timeout 10m go test -p 2 -short ./... -count=1 > ../full-short.log 2>&1
timeout 10m go test -p 2 -short -race ./... -count=1 > ../full-race.log 2>&1
timeout 10m go vet -p 2 ./internal/uci ./cmd/pairmatch > ../vet.log 2>&1
timeout 10m go build -p 2 -pgo=default.pgo -o ../pairmatch ./cmd/pairmatch > ../build.log 2>&1
cd "$s"
sha256sum pairmatch source/cmd/pairmatch/main.go source/cmd/pairmatch/main_test.go > checks-inputs.sha256
date -Is > checks.done
st checks-done 'short/race/full-short/full-race/vet/PGO'
python3 -m py_compile clock-report.py
python3 - <<'PY'
import datetime, hashlib, json, os, platform, subprocess
from pathlib import Path
p=Path.cwd()
files=['baseline-ngn','base.nnue','openings.txt','pairmatch','controller.sh','clock-report.py','manifest.md','nomination.json']
pins={name:hashlib.sha256((p/name).read_bytes()).hexdigest() for name in files}
nom=json.loads((p/'nomination.json').read_text())
for name,pin in nom['binary_inputs_sha256'].items(): assert pins[name]==pin, name
argv=['taskset','-c','4-11',str(p/'pairmatch'),'-ngn',str(p/'baseline-ngn'),'-opponent',str(p/'baseline-ngn'),'-games','24','-time-ms','10000','-increment-ms','100','-concurrency','2','-openings',str(p/'openings.txt'),'-out',str(p/'pairs.jsonl')]
for side in ['ngn','opponent']:
    for option in [f'EvalFile={p}/base.nnue','UseNNUE=true','Hash=64','Threads=1','OwnBook=false']:
        argv += ['-'+side+'option',option]
manifest=dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),nomination=nom,argv=argv,machine=platform.platform(),cpus='4-11',concurrency=2,nice=10,memory_max_bytes=3*1024**3,memory_swap_max=0,environment=dict(GOMAXPROCS='1',GOAMD64='v3'),host_pinning='unverified after reboot',files_sha256=pins,timeout_seconds=1200,decision='DIAGNOSTIC ONLY; no Elo/adoption/retry/extension/automatic reopening')
(p/'game-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
pins['game-manifest.json']=hashlib.sha256((p/'game-manifest.json').read_bytes()).hexdigest()
(p/'inputs.sha256').write_text(''.join(pin+'  '+name+'\n' for name,pin in pins.items()))
with (p/'background-before.txt').open('w') as f:
    subprocess.run(['ps','-eo','pid,ppid,ni,psr,pcpu,rss,comm'],stdout=f,check=True)
with (p/'topology.txt').open('w') as f:
    subprocess.run(['lscpu'],stdout=f,check=True)
PY
sha256sum -c inputs.sha256 > input-check-before.txt
st diagnostic-start 'fixed target24/12 at10+0.1; diagnostic only, no automatic strength gate'
set +e
GOMAXPROCS=1 python3 - <<'PY' > summary.json 2> diagnostic-stderr.log
import json,os,subprocess
from pathlib import Path
argv=json.loads(Path('game-manifest.json').read_text())['argv']
r=subprocess.run(['timeout','--signal=TERM','--kill-after=15s','1200',*argv],env=dict(os.environ,GOMAXPROCS='1'))
Path('diagnostic.exit').write_text(str(r.returncode)+'\n')
PY
set -e
sha256sum -c inputs.sha256 > input-check-after.txt
python3 clock-report.py > clock-report.log 2>&1
st diagnostic-done 'clock-review.json; no strength verdict or lane admission'
