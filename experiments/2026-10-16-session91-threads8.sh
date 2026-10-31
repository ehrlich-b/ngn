#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session91/threads8
echo $$ > "$D/controller.pid"
finish(){
  rc=$?
  echo "$rc" > "$D/controller.exit"
  date -Is > "$D/controller.done"
  python3 - "$rc" <<'PY'
import datetime,json,sys
from pathlib import Path
r=Path('/home/ehrli/ngn-data/night-20261007')
d=r/'session91/threads8'
result=dict(state='PASS' if sys.argv[1]=='0' else 'FAILED', exit=int(sys.argv[1]),
            finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            manifest=str(d/'manifest.json'),behavior_changed=False,
            interpretation='Correctness only on CPUs13,15; no strength or eight-core throughput verdict')
(d/'result.json').write_text(json.dumps(result,indent=2)+'\n')
with (r/'status.jsonl').open('a') as f:
    f.write(json.dumps(dict(utc=result['finished_utc'],stage='session91-threads8',event='complete',detail=result))+'\n')
PY
}
trap finish EXIT
systemd-run --user --wait --collect --unit=ngn-s91-threads8 \
  -p AllowedCPUs=13,15 -p CPUQuota=200% -p MemoryMax=3G -p MemorySwapMax=0 \
  nice -n 19 taskset -c 13,15 bash "$D/work.sh"
