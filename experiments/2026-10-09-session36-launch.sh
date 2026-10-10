#!/usr/bin/env bash
# Recover the existing two jobs once, then queue isolated time checks on their CPU lane.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
P=/mnt/c/Users/ehrli/ngn-session36
bash /mnt/c/Users/ehrli/ngn-session35/2026-10-09-session35-launch.sh
S=$R/session36-time-checks
if [[ ! -e "$S/controller.started" ]]; then
 mkdir -p "$S"
 cp "$P/base.tar" "$S/base.tar"
 cp "$P/candidate.patch" "$S/candidate.patch"
 cp "$P/horizon.patch" "$S/horizon.patch"
 cp "$P/controller.sh" "$S/controller.sh"
 cp "$P/manifest.md" "$S/manifest.md"
 cd "$S"
 sha256sum base.tar candidate.patch horizon.patch controller.sh manifest.md > inputs.sha256
 setsid nohup bash controller.sh < /dev/null > controller.log 2>&1 &
 echo $! > controller.pid
fi
python3 - "$S" <<'PY'
import datetime,json,sys
from pathlib import Path
p=Path(sys.argv[1]);j=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),root=str(p),pid=int((p/'controller.pid').read_text()),cpus=[12,14],stage='time code checks only')
Path('/home/ehrli/ngn-data/night-20261007/session36-launch.json').write_text(json.dumps(j,indent=2)+'\n')
print(json.dumps(j))
PY
