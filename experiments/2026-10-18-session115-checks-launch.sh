#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session115-search
python3 - <<'PY'
import json
from pathlib import Path
r=Path('/home/ehrli/ngn-data/night-20261007');d=r/'session115-speed'
p=d/'s5-profile-admission.json';a=json.loads(p.read_text())
a.update(immediate_accepted_base=(r/'session115/accepted-source.txt').read_text().strip()+'/81215229/compiled warm2000',
         admission='After all four search shelves and retargeted checks; one unchanged exclusive timing gate',
         automatic_timing=True,retarget_note='Same S5 patch; source defaults changed only after the completed warm vector KEEP')
p.write_text(json.dumps(a,indent=2)+'\n')
assert json.loads((r/'session115/checks-result.json').read_text())['state']=='DEFAULTS_CHECKS_PASS'
PY
setsid nohup systemd-run --user --scope --unit=ngn-s115-checks \
 -p AllowedCPUs=13,15 -p CPUQuota=200% -p MemoryMax=3G -p MemorySwapMax=0 \
 taskset -c 13,15 nice -n 19 bash "$D/checks.sh" \
 < /dev/null > "$D/checks-controller.log" 2>&1 &
echo $! > "$D/checks-launcher.pid"
python3 - <<'PY'
import datetime,json
from pathlib import Path
r=Path('/home/ehrli/ngn-data/night-20261007');d=r/'session115-search'
pid=int((d/'checks-launcher.pid').read_text())
(d/'successor.json').write_text(json.dumps(dict(checks_pid=pid),indent=2)+'\n')
result=dict(accepted_source=(r/'session115/accepted-source.txt').read_text().strip(),
 checks_pid=pid,checks_scope='ngn-s115-checks.scope',cpus='13,15',
 first_gate='qsearch after own passing checks',
 then='continuation, pawn, ProbCut only on clean shelves; retargeted S5 timing after all four shelves and passing checks',
 automatic_adoption=False,utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
(d/'start.json').write_text(json.dumps(result,indent=2)+'\n')
with (r/'status.jsonl').open('a') as f:f.write(json.dumps(dict(utc=result['utc'],stage='session115',event='accepted-and-retargeted',detail=result))+'\n')
print(json.dumps(result,indent=2))
PY
