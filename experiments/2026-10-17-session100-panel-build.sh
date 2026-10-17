#!/usr/bin/env bash
set -euo pipefail
S=/home/ehrli/ngn-data/night-20261007/session100-speed
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=/home/ehrli/ngn-data/night-20261007/session17f/go-cache
cd "$S"
python3 - <<'PY'
import os,select
from pathlib import Path
s=Path.cwd()
if not (s/'checks.done').exists():
    try: fd=os.pidfd_open(int((s/'checks.pid').read_text()))
    except ProcessLookupError: pass
    else:
        try: select.select([fd],[],[])
        finally: os.close(fd)
assert (s/'checks.exit').read_text().strip()=='0', 'candidate checks failed; inspect before building panel'
PY
finish() { local rc=$?; echo "$rc" > panel-build.exit; date -Is > panel-build.done; }
trap finish EXIT
mkdir profile-source
tar xf base.tar -C profile-source
cp profile.go.txt profile-source/session100_profile.go
python3 - <<'PY'
from pathlib import Path
p=Path('profile-source/main.go');text=p.read_text()
old='os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))'
assert text.count(old)==1
p.write_text(text.replace(old,'os.Exit(profiledRun())'))
PY
cd profile-source
gofmt -w session100_profile.go
go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/profile-ngn" . > "$S/profile-build.log" 2>&1
cd "$S"
python3 panel.py --smoke --out "$S/panel-smoke" > panel-smoke.log 2>&1
