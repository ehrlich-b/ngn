#!/usr/bin/env bash
# Fixed operational samples only; never launches a strength screen or claim.
set -euo pipefail
[[ ${1:-} == --dry-run || $# == 0 ]] || exit 2
R=/home/ehrli/ngn-data/night-20261007
S=/home/ehrli/ngn-data/claim-v3-20261008/admission-session3
V=/home/ehrli/ngn-data/claim-v3-20261008/tooling
F=$V/frozen
SP=$R/spsa/kb8
PAUSED=0
export PATH=/usr/local/go/bin:$PATH GOMAXPROCS=1
cd "$S"
st() {
  python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                           stage='v3-operational-admission',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
check_inputs() {
  sha256sum -c "$S/frozen-inputs.sha256" > "$S/input-check-${1}.txt"
  python3 - "$V" <<'PY'
import hashlib,json,subprocess,sys
from pathlib import Path
v=Path(sys.argv[1]); f=v/'frozen'; digest=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
assert digest(v/'build-receipt.json')=='b3ceb67ed2c4781465e264d19936022d7759e944de38db3cb8691afc6d41f04a'
receipt=json.loads((v/'build-receipt.json').read_text())
for item in receipt['artifacts'].values():
    assert digest(Path(item['path']))==item['sha256'],item['path']
assert digest(f/'development-identity.json')=='2509127a6eefedde73b7f2cbc648dfed96d1be6c75c1f2a8d66736e0c3aabf31'
assert subprocess.check_output(['git','-C',str(v/'source'),'rev-parse','HEAD'],text=True).strip()=='376671f842aad7841cd2c743c834c61f702eb0ae'
assert not subprocess.check_output(['git','-C',str(v/'source'),'status','--porcelain','--untracked-files=all'],text=True)
PY
}
if [[ ${1:-} == --dry-run ]]; then
  check_inputs dry
  for mode in tooling-check stability-screen stability-confirmation; do
    taskset -c 13,15 nice -n 19 "$F/claimtool" prepare --mode "$mode" \
      --candidate "$F/development-identity.json" --source "$V/source" \
      --tool-build-argv "$F/claimtool-build.json" --dry-run --out "$S/dry-$mode" > "$S/dry-$mode.log" 2>&1
  done
  exit 0
fi
cleanup() {
  local rc=$?
  if [[ $PAUSED == 1 ]]; then
    cd "$SP"
    setsid nohup bash "$R/session1/spsa-resume.cmd" >> "$SP/spsa.log" 2>&1 < /dev/null &
    echo $! > "$SP/spsa.pid"
    st spsa-resume "pid $(cat "$SP/spsa.pid")"
  fi
  if [[ $rc != 0 ]]; then st fail "rc $rc; preserve partial sample, no retry"; fi
  date -Is > "$S/controller.done"
}
trap cleanup EXIT
check_inputs before
[[ -s $S/manifest-commit.txt ]]
st wait "pid $$; native completion wait for session1 controller 522940, bounded at 14h"
python3 - "$R/session1/controller.pid" "$S" <<'PY'
import datetime,os,select,sys
from pathlib import Path
pid=int(Path(sys.argv[1]).read_text()); out=Path(sys.argv[2]); assert pid==522940
cmd=Path(f'/proc/{pid}/cmdline').read_bytes()
assert b'/home/ehrli/ngn-data/night-20261007/session1/controller.sh' in cmd
fd=os.pidfd_open(pid)
poller=select.poll();poller.register(fd,select.POLLIN)
(out/'waiter-ready.json').write_text(__import__('json').dumps(dict(pid=pid,cmdline=cmd.decode().replace('\0',' '),mechanism='pidfd',timeout_hours=14))+'\n')
assert poller.poll(14*60*60*1000),'dependency wait expired; no restart'
os.close(fd)
(out/'dependency-completed.txt').write_text(datetime.datetime.now(datetime.timezone.utc).isoformat()+'\n')
PY
python3 - "$R" <<'PY'
import json,sys
from pathlib import Path
r=Path(sys.argv[1]); events=[json.loads(s) for s in (r/'status.jsonl').read_text().splitlines()]
assert any(e['stage']=='session1' and e['event']=='complete' for e in events),'session1 did not complete normally'
checks=(r/'session1/post-gate-input-check.log').read_text().splitlines()
assert checks and all(line.endswith(': OK') for line in checks)
PY
pid=$(cat "$SP/spsa.pid")
state=$(ps -p "$pid" -o stat= 2>/dev/null || true)
if [[ -n $state && $state != Z* ]]; then
  [[ $(ps -p "$pid" -o pgid= | tr -d ' ') == "$pid" ]]
  ps -p "$pid" -o args= | rg -F "$R/session1/spsa-resume.cmd" > "$S/tuner-before.txt"
  cp "$SP/spsa.jsonl" "$S/tuner-before.jsonl"
  PAUSED=1
  kill -TERM -- "-$pid"
  for ((i=0; i<30; i++)); do
    if ! ps -eo pgid=,stat= | awk -v p="$pid" '$1==p && $2 !~ /^Z/ { found=1 } END { exit !found }'; then break; fi
    sleep 1
  done
  if ps -eo pgid=,stat= | awk -v p="$pid" '$1==p && $2 !~ /^Z/ { found=1 } END { exit !found }'; then
    PAUSED=0
    st fail 'tuner did not drain; no games or duplicate tuner'; exit 1
  fi
  st spsa-pause "group $pid drained; resume from last retained checkpoint"
fi
if pgrep -af 'trainer\.train|datagen|spsa -engine|/sprt |cutechess|fastchess|baseline-probe|candidate-probe' > "$S/blocked-processes.txt"; then
  st fail 'foreign evidence-producing process remains; no live admission'; exit 1
fi
free_bytes=$(df -B1 --output=avail /mnt/c | tail -n 1 | tr -d ' ')
[[ $free_bytes =~ ^[0-9]+$ && $free_bytes -ge 26843545600 ]]
printf '%s\n' "$free_bytes" > "$S/host-c-free-before.txt"
check_inputs pre-games
export NGN_CLAIM_DEADLINE_UTC=$(date -u -d '+45 minutes' '+%Y-%m-%dT%H:%M:%SZ')
for mode in tooling-check stability-screen stability-confirmation; do
  out=$S/run-$mode
  [[ ! -e $out ]]
  st start "$mode; fixed four games; excluded from all claims"
  taskset -c 0-15 nice -n 19 "$F/claimtool" prepare --mode "$mode" \
    --candidate "$F/development-identity.json" --source "$V/source" \
    --tool-build-argv "$F/claimtool-build.json" --out "$out" > "$S/run-$mode.log" 2>&1
  taskset -c 0-15 nice -n 19 "$F/claimtool" run --receipt "$out/receipt.json" --games "$out/games" --out "$out/run-status.json" >> "$S/run-$mode.log" 2>&1
  taskset -c 0-15 nice -n 19 "$F/claimtool" audit --receipt "$out/receipt.json" --games "$out/games" --out "$out/audit.json" >> "$S/run-$mode.log" 2>&1
  taskset -c 0-15 nice -n 19 "$F/claimtool" analyze --receipt "$out/receipt.json" --games "$out/games" --audit "$out/audit.json" --out "$out/analysis.json" >> "$S/run-$mode.log" 2>&1
  taskset -c 0-15 nice -n 19 "$F/claimtool" recheck --receipt "$out/receipt.json" --out "$out/post-run-hashes.json" >> "$S/run-$mode.log" 2>&1
  python3 - "$out" "$mode" <<'PY'
import json,sys
from pathlib import Path
r=Path(sys.argv[1]); mode=sys.argv[2]
receipt=json.loads((r/'receipt.json').read_text()); audit=json.loads((r/'audit.json').read_text()); a=json.loads((r/'analysis.json').read_text())
assert audit['Admitted'] and audit['Exclusions']==0
assert a['Schema']=='NGN-CLAIM-3' and a['ClaimThreshold']==3600 and a['Mode']==mode
assert not a['ScreenEligible'] and not a['ClaimEligible'] and a['Exclusions']==0
assert len(a['Games'])==4 and sum(x['Counts']['N'] for x in a['Anchors'])==4
assert receipt['Plan']['Count']==2 and len(receipt['ClockCheck'])==2*receipt['Plan']['Concurrency']
if mode.startswith('stability-'):
    assert receipt['Selected']['Berserk 12']=={'Threads':'1','Hash':'64','SyzygyPath':'<empty>','MultiPV':'1','Ponder':'false','UCI_ShowWDL':'true','UCI_Chess960':'false','MoveOverhead':'300','Contempt':'0','EvalFile':'<empty>'}
PY
  st done-stage "$mode; four games independently admitted and hashes rechecked"
done
check_inputs after
python3 - "$S" <<'PY'
import datetime,hashlib,json,sys
from pathlib import Path
r=Path(sys.argv[1]); modes=['tooling-check','stability-screen','stability-confirmation']
files={str(p.relative_to(r)):hashlib.sha256(p.read_bytes()).hexdigest() for mode in modes for p in (r/f'run-{mode}').iterdir() if p.is_file()}
receipt=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),modes=modes,games=12,admitted=True,
             claim_eligible=False,candidate='frozen development control be34532c; no final nomination',files=files)
(r/'admission-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
PY
st done 'all 12 operational games admitted; no strength screen/claim queued; review admission-receipt.json'
