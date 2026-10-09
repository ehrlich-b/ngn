#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session19-clock-games
export PATH=/usr/local/go/bin:$PATH
export GOAMD64=v3 GOMAXPROCS=1 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
export NGN_CAMPAIGN_TRACE_NET=$R/kb8h512/best.nnue NGN_CAMPAIGN_TRACE_OPENINGS=$R/session18d/openings.txt
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='clock-trajectory',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > "$S/controller.exit"; date -Is > "$S/controller.done"; st end "rc $rc; accepted two-game spending diagnostic, no strength verdict"; }
trap finish EXIT
cd "$S"
st wait "pid $$; native wait for architecture cost sample; then two played 120+1 games"
python3 - "$R/session19-arch-cost/controller.pid" <<'PY'
import os,select,sys
from pathlib import Path
pid=int(Path(sys.argv[1]).read_text())
try:fd=os.pidfd_open(pid)
except ProcessLookupError:pass
else:
    p=select.poll();p.register(fd,select.POLLIN);assert p.poll(3*3600*1000),'cost wait timeout';os.close(fd)
PY
mkdir source
tar -xf base.tar -C source
cp clock-games_test.go.txt source/engine/campaign_clock_games_test.go
(cd source && taskset -c 13,15 nice -n 19 go test -p 1 -c -pgo="$S/source/default.pgo" -o "$S/engine.test" ./engine) > build.log 2>&1
sha256sum engine.test clock-games_test.go.txt > diagnostic-inputs.sha256
st start 'accepted source a49ebfc/be34532c; actual bank trajectories; first two pinned openings; CPU13,15/nice19'
timeout --signal=TERM --kill-after=10s 35m taskset -c 13,15 nice -n 19 ./engine.test -test.run '^TestCampaignPlayedLongClockGames$' -test.v -test.timeout 34m > games.log 2>&1
st done 'both accepted-runtime trajectories complete; inspect moves15–40 spending and post-completion tails'
