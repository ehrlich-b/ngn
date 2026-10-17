#!/usr/bin/env bash
set -euo pipefail
S=/home/ehrli/ngn-data/night-20261007/session100-speed
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=/home/ehrli/ngn-data/night-20261007/session17f/go-cache
cd "$S"
finish() { local rc=$?; echo "$rc" > claim-checks.exit; date -Is > claim-checks.done; }
trap finish EXIT
python3 - <<'PY'
import os,select
from pathlib import Path
s=Path.cwd()
if not (s/'panel-build.done').exists():
    try: fd=os.pidfd_open(int((s/'panel-build.pid').read_text()))
    except ProcessLookupError: pass
    else:
        try: select.select([fd],[],[])
        finally: os.close(fd)
assert (s/'panel-build.exit').read_text().strip()=='0'
PY
mkdir claim-source
tar xf base.tar -C claim-source
tar xf claim.tar -C claim-source
cd claim-source
gofmt -w internal/claim/model.go internal/claim/analysis.go internal/claim/receipt.go internal/claim/analysis_test.go internal/claim/receipt_test.go
go test -p 2 -short ./internal/claim ./cmd/claimtool -count=1 > "$S/claim-focused.log" 2>&1
go test -p 2 -short ./... -count=1 > "$S/claim-full-short.log" 2>&1
go test -p 2 -short -race ./... -count=1 > "$S/claim-full-race.log" 2>&1
go vet -p 2 ./... > "$S/claim-vet.log" 2>&1
go build -p 2 -o "$S/claimtool-3555" ./cmd/claimtool > "$S/claim-build.log" 2>&1
cd "$S"
python3 boundary-checks.py > boundary-checks.log 2>&1
python3 -m py_compile panel.py boundary.py identity.py
