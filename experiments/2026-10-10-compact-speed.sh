#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session46-compact
export GOMAXPROCS=1
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session46-compact',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; review after earlier time gate; no automatic integration or extension"; }
trap finish EXIT
date -Is > controller.started
python3 - <<'PY'
import os
s=os.statvfs('/mnt/c');assert s.f_bavail*s.f_frsize>=25.5*1024**3
PY
st speed-start 'forge compact checks passed; free rig CPU15 fixed36 identities/40 blocks; prior time gate settled'
if taskset -c 13,15 nice -n 19 python3 check.py > check.log 2>&1; then
 st speed-keep 'complete nominated compact sample passes; session review before integration'
else
 rc=$?
 if [[ $rc == 3 ]]; then st speed-shelve 'complete current-base sample fails; no extension'; else exit "$rc"; fi
fi
