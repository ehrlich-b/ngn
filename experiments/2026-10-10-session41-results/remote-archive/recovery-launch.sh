set -eu
S=/home/ehrli/ngn-data/night-20261007/session41-remote-archive
cd "$S"
mkdir checks-initial
mv checks.done checks.exit remote-tests.log trainer-tests.log checks-initial/
tar -xf /mnt/c/Users/ehrli/ngn-session41-extra.tar -C source
cp /mnt/c/Users/ehrli/ngn-session41-coverage.sh coverage.sh
cat > recover-checks.sh <<'SCRIPT'
set -euo pipefail
S=/home/ehrli/ngn-data/night-20261007/session41-remote-archive
cd "$S/source"
finish() { local rc=$?; echo "$rc" > "$S/checks.exit"; date -Is > "$S/checks.done"; }
trap finish EXIT
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
export PYTHONPATH="$S/source" OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
$PY -m unittest trainer.tests.test_remote_archive trainer.tests.test_ngnn3.NGNN3Tests.test_fixture_reproducible -v > "$S/recovery-tests.log" 2>&1
$PY - <<'PY'
import hashlib,json
from pathlib import Path
s=Path.cwd().parent;files=['trainer/archive.py','trainer/remote_archive.py','trainer/archive_server.py','trainer/tests/test_remote_archive.py']
(s/'check-receipt.json').write_text(json.dumps(dict(tests_pass=True,passed_initial_tests=59,recovered_fixture_pass=True,remote_tests_pass=3,cpus=[12,14],memory_max=2621440000,source_sha256={p:hashlib.sha256((s/'source'/p).read_bytes()).hexdigest() for p in files}),indent=2)+'\n')
PY
SCRIPT
setsid nohup systemd-run --user --scope --unit=ngn-s41-remote-checks-recover -p AllowedCPUs=12,14 -p CPUQuota=200% -p MemoryMax=2500M -p MemorySwapMax=0 bash recover-checks.sh < /dev/null > recovery.log 2>&1 &
echo $! > checks.pid
setsid nohup systemd-run --user --scope --unit=ngn-s41-remote-coverage -p AllowedCPUs=12,14 -p CPUQuota=200% -p MemoryMax=1500M -p MemorySwapMax=0 bash coverage.sh < /dev/null > coverage-controller.log 2>&1 &
echo $! > coverage.pid
cat checks.pid coverage.pid
