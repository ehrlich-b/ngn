#!/usr/bin/env bash
# WSL only. No scored games and no change to the live tuner/trainer.
set -Eeuo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session92
V=/home/ehrli/ngn-data/claim-v4-20261016
status() {
  python3 - "$R/status.jsonl" "$1" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        stage='session92-v4-prep',event=sys.argv[2]))+'\n')
PY
}
finish() {
  rc=$?
  trap - EXIT
  python3 - "$D" "$rc" <<'PY'
import datetime,json,pathlib,sys
p=pathlib.Path(sys.argv[1]);rc=int(sys.argv[2])
(p/'checks-result.json').write_text(json.dumps(dict(state='PASS' if rc==0 else 'FAILED',
    exit_code=rc,finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
    games=0,engine_behavior_changed=False),indent=2)+'\n')
PY
  status "complete-exit-$rc"
  exit "$rc"
}
trap finish EXIT
status start
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2
export GOCACHE=$R/session17f/go-cache
cd "$D/source"
gofmt -w internal/claim/*.go
python3 "$V/anchor-smoke.py" > "$D/anchor-smoke.log" 2>&1
go test -p 1 -short ./internal/claim ./cmd/claimtool -count=1 > "$D/claim-tests.log" 2>&1
go test -p 1 -short ./... -count=1 > "$D/full-short.log" 2>&1
go test -p 1 -short -race ./... -count=1 > "$D/full-race.log" 2>&1
go vet -p 1 ./internal/claim ./cmd/claimtool > "$D/vet.log" 2>&1
bash -n scripts/claim/*.sh
