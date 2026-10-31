import datetime
import json
from pathlib import Path
import subprocess
R=Path('/home/ehrli/ngn-data/night-20261007')
D=R/'session88'
D.mkdir(exist_ok=True)
result=json.loads((R/'session83/provenance/result.json').read_text())
assert result['verdict']=='BISECT' and not result['faults']
# No scored gate was ever admitted: only the zero-search occupancy probe ran.
assert not any((R/'session87'/name/'manifest.json').exists() for name in ['aa-true-width','true-width','net-final174'])
queue=json.loads((R/'session87/queue.json').read_text())
queue.pop('after')
queue['preflight_argv']=['python3',str(D/'width-preflight.py')]
(D/'queue.json').write_text(json.dumps(queue,indent=2)+'\n')
probe=(D/'width-probe-source.py').read_text().replace("D = R/'session87'","D = R/'session88'")
(D/'width-probe.py').write_text(probe)
(D/'width-probe.json').write_bytes((R/'session87/width-probe.json').read_bytes())
preflight=(R/'session87/width-preflight.py').read_text().replace("D = R/'session87'","D = R/'session88'")
(D/'width-preflight.py').write_text(preflight)
(D/'campaign_queue.py').write_bytes((R/'session87/campaign_queue.py').read_bytes())
(D/'queue.sh').write_text('''#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
echo $$ > "$R/session88/queue-controller.pid"
trap 'rc=$?; echo "$rc" > "$R/session88/queue-controller.exit"; date -Is > "$R/session88/queue-controller.done"' EXIT
systemd-run --user --wait --collect --unit=ngn-s88-queue -p AllowedCPUs=0-15 -p CPUQuota=800% -p MemoryMax=3G -p MemorySwapMax=0 nice -n 19 taskset -c 0-15 python3 "$R/session88/campaign_queue.py" "$R/session88/queue.json"
''')
with (R/'status.jsonl').open('a') as stream:
    stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session88',event='queue-recovery',detail='Provenance natural H0/3237/zero faults; original probe maxmoves2 below six book plies searched zero moves; corrected unscored probe leaves two searched plies; original failures retained; no scored gate restarted'))+'\n')
print('Prepared corrected occupancy probe and unchanged width400/Final1741000 queue')
