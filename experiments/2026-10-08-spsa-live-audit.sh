#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session11
cd "$S"
finish() {
  local rc=$?
  printf '%s\n' "$rc" > controller.exit
  date -Is > controller.done
  python3 - "$R/status.jsonl" "$rc" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        stage='spsa-live-audit',event='done' if sys.argv[2]=='0' else 'fail',
        detail='rc '+sys.argv[2]+'; diagnostic only; training and tuner untouched'))+'\n')
PY
}
trap finish EXIT
sha256sum -c audit-inputs.sha256 > input-check-before.txt
[[ $(df -B1 --output=avail /mnt/c | tail -n 1 | tr -d ' ') -ge 26843545600 ]]
kill -0 522940
kill -0 541007
kill -0 577984
[[ ! -d /home/ehrli/ngn-data/claim-v3-20261008/admission-session3/run-tooling-check ]]
python3 - "$R/status.jsonl" <<'PY'
import datetime,json,os,sys
with open(sys.argv[1],'a') as f:
    f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        stage='spsa-live-audit',event='start',
        detail='CPU 13,15; 180 bounded searches; no tuner pause, GPU or games'))+'\n')
PY
taskset -c 13,15 nice -n 19 python3 audit.py "$S" > audit.log 2>&1
sha256sum -c audit-inputs.sha256 > input-check-after.txt
