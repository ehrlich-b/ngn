#!/usr/bin/env bash
# Stage both declared jobs in one WSL entry, without repeating any started job.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
P=/mnt/c/Users/ehrli/ngn-session35
F=$R/session35-factorizer
A=$R/session35-ngnn4-pilot
mkdir -p "$P/staged"
tar -xf "$P/session35-payload.tar" -C "$P/staged"
if [[ ! -e "$F/controller.started" ]]; then
 mkdir -p "$F"
 cp "$P/staged/base.tar" "$F/base.tar"
 cp "$P/staged/factorizer.sh" "$F/controller.sh"
 cp "$P/staged/factorizer.md" "$F/manifest.md"
 cp "$R/kb8h512/best.nnue" "$F/base.nnue"
 cp "$R/session9b/candidate/best.nnue" "$F/candidate.nnue"
 cp "$R/session32-key/sprt" "$R/session32-key/openings.txt" "$R/session32-key/gate-parser.py" "$F/"
 cd "$F"
 sha256sum base.tar controller.sh manifest.md base.nnue candidate.nnue sprt openings.txt gate-parser.py > frozen-inputs.sha256
 setsid nohup bash controller.sh < /dev/null > controller.log 2>&1 &
 echo $! > controller.pid
fi
if [[ ! -e "$A/controller.started" ]]; then
 mkdir -p "$A"
 cp "$P/staged/pilot.sh" "$A/controller.sh"
 cp "$P/staged/pilot.py" "$A/pilot.py"
 cp "$P/staged/pilot.md" "$A/manifest.md"
 cd "$A"
 sha256sum controller.sh pilot.py manifest.md > inputs.sha256
 setsid nohup bash controller.sh < /dev/null > controller.log 2>&1 &
 echo $! > controller.pid
fi
python3 - "$F" "$A" <<'PY'
import datetime,json,sys
from pathlib import Path
j=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),jobs=[])
for folder,cpus in zip(sys.argv[1:],[[13,15],[12,14]]):
 r=Path(folder);j['jobs'].append(dict(root=folder,pid=int((r/'controller.pid').read_text()),cpus=cpus))
Path('/home/ehrli/ngn-data/night-20261007/session35-launch.json').write_text(json.dumps(j,indent=2)+'\n')
print(json.dumps(j))
PY
