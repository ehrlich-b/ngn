#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
finish() { local rc=$?; echo "$rc" > "$R/session80/queue.exit"; date -Is > "$R/session80/queue.done"; }
trap finish EXIT
export GOMAXPROCS=1 PYTHONDONTWRITEBYTECODE=1
python3 "$R/session80/campaign_queue.py" "$R/session80/queue.json"
# Continue the same queue unit after the completed width measurement.
# A low own-width result holds for the mandated profile; no live job is killed.
python3 - <<'PY'
import json,pathlib,re,subprocess
r=pathlib.Path('/home/ehrli/ngn-data/night-20261007');d=r/'session80'
width=json.loads((d/'ngn-width/result.json').read_text())
if width['verdict']=='MEASURED' and width['exact_elo_ci95'][0]>=50:
    queue=json.loads((d/'followup-queue-prospective.json').read_text())
    smoke=(d/'rodent-clock-smoke.log').read_text() if (d/'rodent-clock-smoke.log').exists() else ''
    smoke_ok=(d/'rodent-clock-smoke.exit').exists() and (d/'rodent-clock-smoke.exit').read_text().strip()=='0' and 'Games: 4' in smoke and not re.search(r'WATCHDOG|panic|fatal|no-move|illegal',smoke,re.I)
    if not smoke_ok:
        queue['jobs']=[job for job in queue['jobs'] if job['name']!='rodent-width']
    (d/'followup-queue.json').write_text(json.dumps(queue,indent=2)+'\n')
    subprocess.run(['python3',str(d/'campaign_queue-next.py'),str(d/'followup-queue.json')],check=True)
else:
    (d/'width-profile-required').write_text('Own-width point below50: run4-thread5-20s CPU profile before policy code. No automatic successor.\n')
PY
