set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session44-expanded-data
python3 - "$S" "$R/status.jsonl" <<'PY'
import datetime,json,os,select,signal,sys
from pathlib import Path
s=Path(sys.argv[1])
assert b'trainer.train' in Path('/proc/13201/cmdline').read_bytes()
fd=os.pidfd_open(13201);poll=select.poll();poll.register(fd,select.POLLIN)
assert poll.poll(12*3600*1000),'trainer completion wait expired';os.close(fd)
history_path=s/'trained/history.json'
history=json.loads(history_path.read_text()) if history_path.exists() else []
j=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),event='training-terminal',epochs=len(history),full58=[r['epoch'] for r in history]==list(range(1,59)),postprocessing_machine='forge',original_parent_stopped_then_terminated=True,automatic_games=False)
(s/'training-terminal-session47.json').write_text(json.dumps(j,indent=2)+'\n')
with open(sys.argv[2],'a') as f:f.write(json.dumps(dict(stage='session47-training-handoff',**j))+'\n')
# The unchanged trainer has exited. Retire only its stopped postprocessor.
# Do not resume it: it contains a now-forbidden rig Go parity command.
assert b'trial.py' in Path('/proc/13172/cmdline').read_bytes()
os.kill(13172,signal.SIGKILL)
PY
