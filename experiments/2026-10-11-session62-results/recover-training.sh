set -eu
R=/home/ehrli/ngn-data/night-20261007
S=$R/session57-data-step2
C=$R/session62-loader-index
test "$(cat "$C/checks.exit")" = 0
test "$(cat "$S/training-controller-session60.exit")" = 1
/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python - <<'PY'
import datetime,hashlib,json
from pathlib import Path
r=Path('/home/ehrli/ngn-data/night-20261007');s=r/'session57-data-step2';c=r/'session62-loader-index'
assert json.loads((s/'trained/history.json').read_text())==[]
assert {p.name for p in (s/'trained').iterdir()}=={'history.json','config.json'}
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
old=(r/'session44-expanded-data/source/trainer/archive.py').read_text()
new=(c/'source/trainer/archive.py').read_text()
assert new==old.replace('                    records = canonical_records(self.maps[shard][start + selected])', '                    # Sparse offsets fit uint32; absolute rows can exceed 2^32.\n                    records = canonical_records(self.maps[shard][start + selected.astype(np.intp)])')
receipt=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),fault='absolute holdout offset overflow at 2^32; first epoch validation; no durable epoch or checkpoint',before_sha256=sha(r/'session44-expanded-data/source/trainer/archive.py'),after_sha256=sha(c/'source/trainer/archive.py'),regression_test_sha256=sha(c/'source/trainer/tests/test_archive_index.py'),checks='new high-index witness fails before; old archive/local-remote/training-resume suites and new high-index controls pass after',recipe='unchanged',resume='fresh identical seed, no checkpoint available',coverage_ratio=2.0009128203817186)
(c/'repair-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
PY
mkdir "$S/interrupted-session62-loader"
for f in trained train.log training-manifest.json trainer.pid training-controller-session60.started training-controller-session60.exit training-controller-session60.done; do mv "$S/$f" "$S/interrupted-session62-loader/"; done
setsid nohup systemd-run --user --scope --unit=ngn-s62-data-training -p AllowedCPUs=12,14 -p CPUQuota=200% -p MemoryMax=4G -p MemorySwapMax=0 taskset -c 12,14 nice -n 19 bash "$S/training-controller-session62.sh" < /dev/null > "$S/training-controller-session62.log" 2>&1 &
echo "$!" > "$S/training-controller-session62.pid"
cat "$S/training-controller-session62.pid"
