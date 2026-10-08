#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session12
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3
export GOMAXPROCS=2 GOCACHE=$S/go-cache PYTHONDONTWRITEBYTECODE=1
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        stage='spsa-exact-resume',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
  local rc=$?
  printf '%s\n' "$rc" > "$S/controller.exit"
  date -Is > "$S/controller.done"
  st "$( [[ $rc == 0 ]] && echo done || echo fail )" "rc $rc; isolated future tuner only; live jobs untouched"
}
trap finish EXIT
cd "$S"
[[ ! -e controller.started ]]
sha256sum -c preparation-inputs.sha256 > input-check-before.txt
[[ $(df -B1 --output=avail /mnt/c | tail -n 1 | tr -d ' ') -ge 26843545600 ]]
date -Is > controller.started
st start "pid $$; CPU 13,15; exact-resume synthetic controls; no live tuner changes or games"
mkdir source
tar -xf source.tar -C source
tar -xf overlay.tar -C source
cd source
taskset -c 13,15 nice -n 19 gofmt -w cmd/spsa/*.go
find . -type f -not -path './.git/*' -print0 | sort -z | xargs -0 sha256sum > "$S/source-inputs.sha256"
check() {
  local name=$1; shift
  local rc=0
  taskset -c 13,15 nice -n 19 "$@" > "$S/$name.log" 2>&1 || rc=$?
  printf '%s\n' "$rc" > "$S/$name.exit"
  [[ $rc == 0 ]]
}
[[ $(go version) == 'go version go1.25.5 linux/amd64' ]]
check spsa-tests go test -short ./cmd/spsa -count=1 -v
check spsa-race go test -short -race ./cmd/spsa -count=1
check engine-short go test -short ./engine -count=1
check engine-race go test -short -race ./engine -count=1
check full-short go test -short ./... -count=1
check full-race go test -short -race ./... -count=1
check vet go vet ./...
check build go build -p 2 -o "$S/future-spsa" ./cmd/spsa
go version -m "$S/future-spsa" > "$S/buildinfo.txt"
sha256sum -c "$S/source-inputs.sha256" > "$S/source-check-after.txt"
cd "$S"
taskset -c 13,15 nice -n 19 python3 - <<'PY'
import datetime,hashlib,json,os,subprocess
from pathlib import Path
s=Path.cwd(); digest=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
stub=s/'must-not-start'
stub.write_text('#!/bin/sh\ntouch "'+str(s/'engine-started')+'"\nexit 1\n'); stub.chmod(0o755)
legacy=s/'legacy.jsonl'; legacy.write_text('{"iter":25,"theta":{"NullMoveR":3}}\n')
empty=s/'empty.jsonl'; empty.write_text('')
corrupt=s/'corrupt.jsonl'; corrupt.write_text('{\n')
cases=[('legacy',['-resume','-out',str(legacy)]),
       ('missing',['-resume','-out',str(s/'missing.jsonl')]),
       ('empty',['-resume','-out',str(empty)]),
       ('corrupt',['-resume','-out',str(corrupt)]),
       ('mixed-fresh',['-out',str(legacy)]),
       ('bad-report',['-report','0','-out',str(s/'bad-report.jsonl')]),
       ('bad-iters',['-iters','0','-out',str(s/'bad-iters.jsonl')]),
       ('bad-pairs',['-pairs','0','-out',str(s/'bad-pairs.jsonl')]),
       ('bad-rend',['-rend','NaN','-out',str(s/'bad-rend.jsonl')]),
       ('bad-afrac',['-afrac','Inf','-out',str(s/'bad-afrac.jsonl')]),
       ('selftest-resume',['-selftest','-resume','-out',str(legacy)])]
receipts=[]
for name,opts in cases:
    argv=['taskset','-c','13,15','nice','-n','19',str(s/'future-spsa'),'-engine',str(stub),'-lowpower=false']+opts
    p=subprocess.run(argv,text=True,capture_output=True,timeout=30)
    assert p.returncode!=0 and not (s/'engine-started').exists(),name
    (s/(name+'.stdout')).write_text(p.stdout); (s/(name+'.stderr')).write_text(p.stderr)
    receipts.append(dict(case=name,argv=argv,returncode=p.returncode,engine_started=False,stderr=p.stderr))
(s/'cli-receipt.json').write_text(json.dumps(receipts,indent=2)+'\n')
names=['spsa-tests','spsa-race','engine-short','engine-race','full-short','full-race','vet','build']
assert all((s/(n+'.exit')).read_text().strip()=='0' for n in names)
testlog=(s/'spsa-tests.log').read_text()
assert 'PASS' in testlog and 'FAIL' not in testlog
receipt=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
    cpu=[13,15],nice=19,synthetic_only=True,live_tuner_replaced=False,
    checks={n:dict(exit=0,log_sha256=digest(s/(n+'.log'))) for n in names},
    cli_refusals=len(receipts),future_tuner_sha256=digest(s/'future-spsa'),
    source_input_list_sha256=digest(s/'source-inputs.sha256'),
    source_files={str(p.relative_to(s/'source')):digest(p) for p in (s/'source/cmd/spsa').glob('*.go')})
(s/'test-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
PY
sha256sum -c preparation-inputs.sha256 > input-check-after.txt
