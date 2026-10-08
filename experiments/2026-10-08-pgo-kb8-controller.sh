#!/usr/bin/env bash
# Recheck the existing speed profile after the screen and anchor preparation.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session1
SP=$R/spsa/kb8
A=/home/ehrli/ngn-data/claim-v3-20261008/anchors/berserk12
PAUSED=0
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='pgo-kb8',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
cleanup() {
  local rc=$?
  if [[ $PAUSED == 1 ]]; then
    cd "$SP"
    setsid nohup bash "$S/spsa-resume.cmd" >> "$SP/spsa.log" 2>&1 < /dev/null &
    echo $! > "$SP/spsa.pid"
    st spsa-resume "pid $(cat "$SP/spsa.pid")"
  fi
  [[ $rc == 0 ]] || st fail "rc $rc; no PGO merge or retry"
}
trap cleanup EXIT
st wait "pid $$; screen and anchor build must finish first"
ready=0
for ((i=0; i<24; i++)); do
  if [[ -s $S/train.pid ]]; then
    # The build publishes its receipt just before its final date/exit; wait for exit too.
    apid=$(cat "$A/build.pid")
    astate=$(ps -o stat= -p "$apid" 2>/dev/null || true)
    if [[ -z $astate || $astate == Z* ]]; then ready=1; break; fi
  else
    kill -0 "$(cat "$S/controller.pid")" 2>/dev/null || exit 1
  fi
  sleep 300
done
[[ $ready == 1 ]] || { st fail 'dependency wait exhausted'; exit 1; }
python3 - "$R" <<'PY'
import hashlib,sys
from pathlib import Path
r=Path(sys.argv[1])
for line in (r/'session1/pgo-source-blobs.txt').read_text().splitlines():
    expected,relative=line.split(' ',1); data=(r/'pgo/src'/relative).read_bytes()
    assert hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest()==expected,relative
PY
pid=$(cat "$SP/spsa.pid")
pgid=$(ps -o pgid= -p "$pid" | tr -d ' ')
[[ $pgid == "$pid" ]]
kill -TERM -- "-$pid"
for ((i=0; i<30; i++)); do
  if ! ps -eo pgid=,stat= | awk -v p="$pid" '$1==p && $2 !~ /^Z/ { found=1 } END { exit !found }'; then break; fi
  sleep 1
done
if ps -eo pgid=,stat= | awk -v p="$pid" '$1==p && $2 !~ /^Z/ { found=1 } END { exit !found }'; then
  st fail 'tuner did not drain; no timing run'; exit 1
fi
PAUSED=1
st start "SPSA paused; fixed identity and ten interleaved timing blocks on CPU 4"
python3 "$S/pgo-kb8-check.py" > "$S/pgo-kb8.log" 2>&1
st done "see $S/pgo-kb8/receipt.json; review before merge"
