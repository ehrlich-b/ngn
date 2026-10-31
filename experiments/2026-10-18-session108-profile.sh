#!/usr/bin/env bash
# Bounded accepted-base profile on the spare campaign check CPUs, not a gate.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session108-speed
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache
finish() { local rc=$?; echo "$rc" > "$D/profile.exit"; date -Is > "$D/profile.done"; }
trap finish EXIT
echo $$ > "$D/profile.pid"
# The preceding checks are short and already on their final candidate.
while [ ! -f "$R/session107/checks.done" ]; do sleep 60; done
test "$(cat "$R/session107/checks.exit")" = 0
mkdir "$D/profile-source"
tar xf "$R/session107/base.tar" -C "$D/profile-source"
cp "$R/session100-speed/profile.go.txt" "$D/profile-source/session108_profile.go"
python3 - "$D" <<'PY'
from pathlib import Path
import sys
d=Path(sys.argv[1]);p=d/'profile-source/main.go';s=p.read_text()
old='os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))'
assert s.count(old)==1
p.write_text(s.replace(old,'os.Exit(profiledRun())'))
PY
cd "$D/profile-source"
gofmt -w session108_profile.go
go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/profile-ngn" . > "$D/profile-build.log" 2>&1
cd /home/ehrli
python3 - "$D" <<'PY'
import datetime,hashlib,importlib.util,json,sys
from pathlib import Path
d=Path(sys.argv[1]);r=d.parent
spec=importlib.util.spec_from_file_location('panel',r/'session100-speed/panel.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
frames=json.loads((r/'session77-smp-panel/panel.json').read_text())
manifest=dict(base='fe92ced/81215229/compiled final14',purpose='bounded hotspot diagnosis only',
    width=1,hash_mib=256,cpu=13,nice=19,frames=frames,milliseconds=5000,
    binary_sha256=hashlib.sha256((d/'profile-ngn').read_bytes()).hexdigest(),
    started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
(d/'profile-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
e=m.UCI(d/'profile-ngn',1,256,d,prefix=d/'accepted',smoke=True)
try:
    rows=[e.search(frame,5000) for frame in frames]
finally:
    e.close()
(d/'profile-searches.json').write_text(json.dumps(rows,indent=2)+'\n')
PY
go tool pprof -source_path=/home/ehrli -list 'alphaBetaPV|quiescenceWithDepth' "$D/profile-ngn" "$D/accepted.cpu.pprof" > "$D/search-list.txt" 2>&1
go tool pprof -top -nodecount=60 "$D/profile-ngn" "$D/accepted.cpu.pprof" > "$D/top.txt" 2>&1
