#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session66-clock-ledger
cd "$s"
finish() { local rc=$?; echo "$rc" > "$s/checks.exit"; date -Is > "$s/checks.done"; }
trap finish EXIT
python3 - <<'PY'
import ctypes,os,select,time
from pathlib import Path
p=Path('/home/ehrli/ngn-data/night-20261007/session66-correctness')
lib=ctypes.CDLL(None);fd=lib.inotify_init1(os.O_NONBLOCK)
assert fd>=0 and lib.inotify_add_watch(fd,os.fsencode(p),0x8|0x80)>=0
end=time.monotonic()+3600
while not (p/'checks.done').exists():
    remaining=end-time.monotonic();assert remaining>0
    assert select.select([fd],[],[],remaining)[0]
    os.read(fd,65536)
os.close(fd)
PY
export PATH=/home/ehrli/go/bin:/usr/local/go/bin:$PATH
export GOMAXPROCS=2 GOAMD64=v3
cd source
date -Is > "$s/checks.started"
go test -p 2 -short ./internal/uci -run '^TestTournamentClockLedger$' -count=1 -v > "$s/clock-test.log" 2>&1
go test -p 2 -short ./internal/uci ./cmd/pairmatch -count=1 > "$s/short.log" 2>&1
go test -p 2 -short -race ./internal/uci ./cmd/pairmatch -count=1 > "$s/race.log" 2>&1
go vet -p 2 ./internal/uci ./cmd/pairmatch > "$s/vet.log" 2>&1
go build -p 2 -pgo=default.pgo -o "$s/pairmatch" ./cmd/pairmatch > "$s/build.log" 2>&1
sha256sum "$s/pairmatch" internal/uci/uci.go internal/uci/clock_test.go cmd/pairmatch/main.go > "$s/inputs.sha256"
python=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
"$python" -m py_compile "$s/confirmation.py" "$s/clock-audit-check.py"
"$python" "$s/clock-audit-check.py" > "$s/clock-audit-check.log" 2>&1
