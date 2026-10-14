#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session81/profile
finish() { local rc=$?; echo "$rc" > "$D/controller.exit"; date -Is > "$D/controller.done"; }
trap finish EXIT
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off GOCACHE=$R/session17f/go-cache
cd "$D/source"
GOMAXPROCS=2 taskset -c 13,15 go test -p 1 -c -o "$D/engine.test" ./engine > "$D/build.log" 2>&1
export NGN_S81_PROFILE=$D NGN_CAMPAIGN_NET=$R/session79-admission-see/base.nnue
for cell in 4:1 4:4 1:1; do
 export NGN_S81_THREADS=${cell%:*} GOMAXPROCS=${cell#*:}
 taskset -c 0,2,4,6 "$D/engine.test" -test.run '^TestS81SMPProfile$' -test.v -test.timeout 5m > "$D/t${NGN_S81_THREADS}-p${GOMAXPROCS}.log" 2>&1
 for kind in cpu mutex block; do
  GOMAXPROCS=2 taskset -c 13,15 go tool pprof -top -nodecount=20 "$D/engine.test" "$D/t${NGN_S81_THREADS}-p${GOMAXPROCS}-f0-s20.$kind.pprof" > "$D/t${NGN_S81_THREADS}-p${GOMAXPROCS}.$kind.txt" 2>&1
 done
done
python3 - <<'PY'
import datetime,json,pathlib
r=pathlib.Path('/home/ehrli/ngn-data/night-20261007');d=r/'session81/profile'
rows=[json.loads(line.removeprefix('S81PROFILE ')) for p in d.glob('t*-p*.log') for line in p.read_text().splitlines() if line.startswith('S81PROFILE ')]
(d/'result.json').write_text(json.dumps(dict(rows=rows,completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),class_='diagnostic',no_elo_verdict=True),indent=2)+'\n')
with (r/'status.jsonl').open('a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session81-profile',event='result',detail={'rows':len(rows),'result':str(d/'result.json')}))+'\n')
PY
