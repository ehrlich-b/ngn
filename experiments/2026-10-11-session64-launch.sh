#!/usr/bin/env bash
set -euo pipefail
s=/work/lanes/ngn-s64/current-spsa
chmod +x "$s/baseline-ngn" "$s/candidate-ngn"
bash -n "$s/controller.sh"
python3 - "$s/games.py" <<'PY'
import sys
from pathlib import Path
p=Path(sys.argv[1]);compile(p.read_text(),str(p),'exec')
PY
setsid nohup systemd-run --user --scope --unit=ngn-s64-current-spsa -p AllowedCPUs=4-11 -p CPUQuota=800% -p MemoryMax=3G -p MemorySwapMax=0 nice -n 10 bash "$s/controller.sh" < /dev/null > "$s/controller.log" 2>&1 &
echo $! > "$s/controller.pid"
cat "$s/controller.pid"
