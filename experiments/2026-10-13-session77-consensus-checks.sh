#!/usr/bin/env bash
set -euo pipefail
D=/home/ehrli/ngn-data/night-20261007/session77-consensus
cd "$D/source"
finish(){ rc=$?; echo "$rc" > "$D/checks.exit"; date -Is > "$D/checks.done"; }
trap finish EXIT
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOTOOLCHAIN=local GOMAXPROCS=2
export GOCACHE=/home/ehrli/ngn-data/night-20261007/session17f/go-cache
export NGN_CAMPAIGN_NET=/home/ehrli/ngn-data/night-20261007/session77-see-rig/base.nnue
# Diagnostic checks serialize natively with this use of the spare two cores.
python3 - <<'PY'
import os,select,pathlib
p=pathlib.Path('/home/ehrli/ngn-data/night-20261007/session77-smp-panel')
if not (p/'checks.done').exists() or (p/'checks.exit').read_text().strip()!='0':
 try:
  fd=os.pidfd_open(5730); poll=select.poll();poll.register(fd,select.POLLIN);assert poll.poll(3600000);os.close(fd)
 except ProcessLookupError: pass
assert (p/'checks.exit').read_text().strip()=='0','panel checks failed'
PY
gofmt -w engine/search_smp.go engine/search_smp_consensus.go engine/search_smp_consensus_test.go
go test -p 1 -short ./engine -run '^(TestSMPConsensus|TestLazySMP)' -count=1 -v > "$D/focused.log" 2>&1
go test -p 1 -short ./engine -count=1 > "$D/engine-short.log" 2>&1
for width in 2 4; do
 GOMAXPROCS=$width go test -p 1 -short -race ./... -count=1 > "$D/full-race-$width.log" 2>&1
done
go vet -p 1 ./... > "$D/vet.log" 2>&1
go build -p 1 -pgo=default.pgo -o "$D/candidate-ngn" . > "$D/build.log" 2>&1
sha256sum "$D/candidate-ngn" > "$D/candidate.sha256"

python3 "$D/identity.py" > "$D/identity.log" 2>&1
