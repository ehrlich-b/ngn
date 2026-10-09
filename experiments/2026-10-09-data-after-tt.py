"""Keep large corpus conversion off the TT game's CPU13,15 window."""
import datetime
import json
import os
from pathlib import Path
import select
import signal

root = Path('/home/ehrli/ngn-data/night-20261007')
record = root / 'session19-data-resume'
record.mkdir(exist_ok=True)
data_pid = 655799
gate_pid = int((root / 'session19-tt-gate/controller.pid').read_text())
assert os.getpgid(data_pid) == data_pid
assert os.readlink(f'/proc/{data_pid}/cwd') == str(root / 'session19-data')
assert (root / 'session19-data/preparation.done').exists()
assert not (root / 'session19-data/scatter-t76').exists()


def status(event):
    with (root / 'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                               stage='data-after-tt', event=event,
                               detail=f'data group{data_pid}; TT gate{gate_pid}; native dependency')) + '\n')


os.killpg(data_pid, signal.SIGSTOP)
status('hold')
try:
    if not (root / 'session19-tt-gate/controller.done').exists():
        fd = os.pidfd_open(gate_pid)
        poll = select.poll()
        poll.register(fd, select.POLLIN)
        assert poll.poll(6 * 60 * 60 * 1000), 'TT gate wait timeout'
        os.close(fd)
    assert (root / 'session19-tt-gate/controller.done').exists()
    (record / 'controller.exit').write_text('0\n')
finally:
    os.killpg(data_pid, signal.SIGCONT)
    status('release')
    (record / 'controller.done').write_text(datetime.datetime.now(datetime.timezone.utc).isoformat() + '\n')
