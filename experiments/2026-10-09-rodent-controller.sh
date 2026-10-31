#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session19-rodent3
SP=$R/spsa/kb8
FHELD=0
SPAUSED=0
export GOMAXPROCS=1 GOAMD64=v3
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='rodent-head-to-head',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
resume() {
 if [[ $SPAUSED == 1 ]]; then
  if python3 - "$SP/spsa.jsonl" <<'PY'
import json,sys
from pathlib import Path
assert json.loads(Path(sys.argv[1]).read_text().splitlines()[-1])['iter']<2000
PY
  then
   setsid nohup bash "$R/session1/spsa-resume.cmd" >> "$SP/spsa.log" 2>&1 < /dev/null 9>&- &
   echo $! > "$SP/spsa.pid"
   st spsa-resume "old-base tuner wrapper $(cat "$SP/spsa.pid"); unchanged frozen vector"
  fi
  SPAUSED=0
 fi
 if [[ $FHELD == 1 ]]; then
  kill -CONT 630389
  FHELD=0
  st factorizer-release 'controller630389 resumes parity/A/A/gate; trainer was never stopped'
 fi
}
finish() {
 local rc=$?
 resume
 echo "$rc" > controller.exit
 date -Is > controller.done
 st end "rc $rc; report complete pairs only, no rating or campaign claim"
}
trap finish EXIT
exec 9> "$R/rodent-match.lock"
flock -n 9
[[ ! -e controller.started ]]
date -Is > controller.started
sha256sum -c match-inputs.sha256 > input-check-before.txt
[[ $(df -B1 --output=avail /mnt/c | tail -n 1 | tr -d ' ') -ge 26843545600 ]]
[[ $(readlink /proc/630389/cwd) == "$R/session9b" ]]
[[ $(ps -p 630389 -o args=) == 'bash trial.sh' ]]
[[ $(ps -p 630389 -o stat=) != *T* ]]
FHELD=1
kill -STOP 630389
st factorizer-hold 'only bash controller630389 stopped; active GPU trainer630440 keeps training; parity and games wait for match'
pid=$(cat "$SP/spsa.pid")
state=$(ps -p "$pid" -o stat= 2>/dev/null || true)
if [[ -n $state && $state != Z* ]]; then
 [[ $(ps -p "$pid" -o pgid= | tr -d ' ') == "$pid" ]]
 ps -p "$pid" -o args= | rg -F "$R/session1/spsa-resume.cmd" > tuner-before.txt
 minimum=$(python3 - "$SP/spsa.jsonl" <<'PY'
import json,sys
from pathlib import Path
i=json.loads(Path(sys.argv[1]).read_text().splitlines()[-1])['iter']
print(min(2000,(i//25+1)*25))
PY
 )
 st checkpoint-wait "native inotify/pidfd; tuner $pid at next durable iteration $minimum"
 python3 checkpoint-wait.py "$SP/spsa.jsonl" "$pid" "$minimum" 5400 > checkpoint-ready.json
 [[ $(cat "$SP/spsa.pid") == "$pid" ]]
 cp "$SP/spsa.jsonl" tuner-before.jsonl
 python3 - "$pid" <<'PY'
import os,select,signal,sys,time
pid=int(sys.argv[1]);fd=os.pidfd_open(pid)
os.killpg(pid,signal.SIGTERM)
p=select.poll();p.register(fd,select.POLLIN)
assert p.poll(30000),'SPSA wrapper did not terminate'
os.close(fd)
deadline=time.monotonic()+30
while True:
    alive=[]
    for name in os.listdir('/proc'):
        if not name.isdigit():continue
        try:
            fields=open('/proc/'+name+'/stat').read().split(') ',1)[1].split()
            if int(fields[2])==pid and fields[0]!='Z':alive.append(name)
        except (FileNotFoundError,ProcessLookupError):pass
    if not alive:break
    assert time.monotonic()<deadline,('SPSA descendants did not drain',alive)
    time.sleep(.25)
PY
else
 # First launch drained iteration825, then failed opening its exited pidfd.
 # No games ran. Reuse durable state and restore it after this match.
 cp "$R/session19-rodent/checkpoint-ready.json" checkpoint-ready.json
 cp "$SP/spsa.jsonl" tuner-before.jsonl
 python3 - <<'PY'
import json
from pathlib import Path
r=Path.cwd();a=json.loads((r/'checkpoint-ready.json').read_text())['checkpoint']
b=json.loads((r/'tuner-before.jsonl').read_text().splitlines()[-1])
assert a==b and b['iter']==825
PY
fi
if ps -eo pgid=,stat= | awk -v p="$pid" '$1==p && $2 !~ /^Z/ { found=1 } END { exit !found }'; then exit 1; fi
SPAUSED=1
st spsa-paused "exact group $pid drained at durable checkpoint; complete state saved"
if pgrep -af '/spsa -engine' > blocked-tuner.txt; then exit 1; fi
python3 - <<'PY'
import os
from pathlib import Path
for p in Path('/proc').iterdir():
 if not p.name.isdigit():continue
 try:
  args=(p/'cmdline').read_bytes().replace(b'\0',b' ').decode(errors='replace')
  if '/sprt ' in args or 'cutechess' in args or 'fastchess' in args:
   assert not (os.sched_getaffinity(int(p.name)) & set(range(12))),('other games on0-11',p.name,args)
 except (FileNotFoundError,ProcessLookupError):pass
PY
remaining=$(python3 - <<'PY'
import datetime
d=datetime.datetime(2026,10,9,11,45,tzinfo=datetime.timezone.utc)
s=int((d-datetime.datetime.now(datetime.timezone.utc)).total_seconds())
assert s>1500,'too late to admit a complete pair'
print(s)
PY
)
st match-start '120+1; six paired workers; CPU0-11 nice19; no new pair after07:20 EDT; finish by07:45'
cd "$S/opponent-only/rodent"
timeout --signal=TERM --kill-after=15s "$remaining" taskset -c 0-11 nice -n 19 \
 "$S/pairmatch" -ngn "$S/ngn" -opponent "$S/opponent-only/rodent-v12" \
 -openings "$S/openings.txt" -concurrency 6 -until 2026-10-09T11:45:00Z -out "$S/pairs.jsonl" \
 -ngnoption "EvalFile=$S/base.nnue" -ngnoption UseNNUE=true -ngnoption Hash=64 -ngnoption Threads=1 -ngnoption OwnBook=false \
 -opponentoption Hash=64 -opponentoption Threads=1 -opponentoption OwnBook=false -opponentoption UCI_LimitStrength=false \
 > "$S/summary.json" 2> "$S/match.log"
cd "$S"
sha256sum -c match-inputs.sha256 > input-check-after.txt
st match-done "$(cat summary.json)"
