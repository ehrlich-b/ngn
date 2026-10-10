#!/usr/bin/env bash
# Pipe this script to rig's WSL only after reviewing the capacity recovery.
# Choose training or capture explicitly; no retry loop or automatic adoption.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
MODE=${NGN_SESSION44_STAGE:?Set NGN_SESSION44_STAGE=training or capture}
python3 - <<'PY'
import os
s=os.statvfs('/mnt/c');assert s.f_bavail*s.f_frsize>=25.5*1024**3
PY
case "$MODE" in
 training) S=$R/session44-expanded-data; UNIT=ngn-s44-expanded-data; CPUS=12,14; CAP=4G ;;
 capture) S=$R/session44-capture-quiet; UNIT=ngn-s44-capture-quiet; CPUS=13,15; CAP=2G ;;
 *) exit 2 ;;
esac
cd "$S"
test ! -e controller.pid
setsid nohup systemd-run --user --scope --unit="$UNIT" -p AllowedCPUs="$CPUS" -p CPUQuota=200% -p MemoryMax="$CAP" -p MemorySwapMax=0 nice -n 19 bash controller.sh < /dev/null > controller.log 2>&1 &
echo $! > controller.pid
cat controller.pid
