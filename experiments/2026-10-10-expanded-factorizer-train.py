"""Run the one-flag expanded-data factorizer trial; postprocess on forge."""
import datetime
import json
import math
import os
from pathlib import Path
import signal
import subprocess
import sys

s = Path(sys.argv[1]).resolve()
manifest = json.loads((s / 'training-manifest.json').read_text())
out = s / 'trained'
assert not out.exists()
with (s / 'train.log').open('w') as log:
    child = subprocess.Popen(manifest['argv'], cwd=manifest['trainer_source'],
                             stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
    (s / 'trainer.pid').write_text(str(child.pid) + '\n')
    try:
        while True:
            disk = os.statvfs('/mnt/c')
            if disk.f_bavail * disk.f_frsize < 25.25 * 1024**3:
                raise RuntimeError('C: reserve below25.25GiB; preserve training, no automatic restart')
            try:
                rc = child.wait(timeout=60)
                break
            except subprocess.TimeoutExpired:
                pass
    except BaseException:
        if child.poll() is None:
            os.killpg(child.pid, signal.SIGTERM)
            try:
                child.wait(timeout=15)
            except subprocess.TimeoutExpired:
                os.killpg(child.pid, signal.SIGKILL)
                child.wait()
        raise
assert rc == 0, f'trainer exit {rc}'
history = json.loads((out / 'history.json').read_text())
assert [row['epoch'] for row in history] == list(range(1, 59))
assert all(row['positions'] == 256000000 and math.isfinite(row['validation_loss']) for row in history)
config = json.loads((out / 'config.json').read_text())
expected = json.loads((s / 'baseline-config.json').read_text())
expected.update(king_factorizer=True, out=str(out))
assert config == expected, [key for key in config if config[key] != expected.get(key)]
terminal = dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                full58=True, epochs=58, total_positions=sum(row['positions'] for row in history),
                king_factorizer=True, postprocessing_machine='forge', automatic_games=False)
(s / 'training-terminal.json').write_text(json.dumps(terminal, indent=2) + '\n')
print(json.dumps(terminal))
