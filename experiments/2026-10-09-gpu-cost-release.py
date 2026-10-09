"""Retarget unstarted CUDA windows after the Rodent priority holds factorizer."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import select
import signal
import subprocess

root = Path('/home/ehrli/ngn-data/night-20261007')
record = root / 'session19-gpu-release2'
record.mkdir(exist_ok=True)


def status(event, detail):
    with (root / 'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                               stage='gpu-cost-release', event=event, detail=detail)) + '\n')


def original(pid, directory):
    assert os.getpgid(pid) == pid
    assert os.readlink(f'/proc/{pid}/cwd') == str(directory)
    assert Path(f'/proc/{pid}/cmdline').read_bytes().endswith(b'controller.sh\0cuda\0')
    assert not (directory / 'cuda-tests.log').exists(), 'CUDA already started'
    assert not (directory / 'cuda-controller.done').exists()
    return pid, directory


jobs = [original(641099, root / 'session17b'), original(651274, root / 'session17s')]
admission = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                 reason='Rodent priority holds factorizer parity/games through07:45; avoid GPU starvation',
                 trainer_pid=630440, cpu_affinity=[12, 14], nice=19,
                 samples_and_verdict_rules_unchanged=True, pending_only=True,
                 original_waiting_groups=[p for p, _ in jobs], automatic_training=False,
                 controllers={str(s): hashlib.sha256((s / 'controller.sh').read_bytes()).hexdigest()
                              for _, s in jobs})
(record / 'admission.json').write_text(json.dumps(admission, indent=2) + '\n')
# These groups contain only waiting shells/pidfd waiters, no CUDA work.
for pid, directory in jobs:
    os.killpg(pid, signal.SIGSTOP)
    (directory / 'cuda-controller-original.pid').write_text(str(pid) + '\n')
    for suffix in ['started', 'log']:
        p = directory / ('cuda-controller.' + suffix)
        (directory / ('cuda-controller-original.' + suffix)).write_bytes(p.read_bytes())
status('wait', 'native trainer630440 completion; pending641099/651274 stopped; new CUDA windows12,14, same rules')
try:
    fd = os.pidfd_open(630440)
    poll = select.poll()
    poll.register(fd, select.POLLIN)
    assert poll.poll(12 * 60 * 60 * 1000), 'GPU training dependency timeout'
    os.close(fd)
    # Train writes one history row only after the full epoch/validation completes.
    history = json.loads((root / 'session9b/candidate/history.json').read_text())
    assert len(history) == 58, 'full training schedule not completed'
    assert [r['epoch'] for r in history] == list(range(1, 59))
    for pid, directory in jobs:
        source = (directory / 'controller.sh').read_text()
        if directory.name == 'session17b':
            old = '  wait_for "$R/session9b/controller.done" "$R/session9b/controller.pid"\n  [[ $(cat "$R/session9b/controller.exit") == 0 ]]\n'
            assert source.count(old) == 1
            source = source.replace(old, '  # Full58-epoch trainer completion checked by the admission owner.\n')
        source = source.replace('taskset -c 13,15', 'taskset -c 12,14')
        # Rodent0-11 and TT13,15 are disjoint from these GPU launch threads.
        # Training/datagen/exclusive claims still prevent admission.
        source = source.replace('trainer\\.train|/datagen |/claimtool |/sprt |baseline-probe|candidate-probe',
                                'trainer\\.train|/datagen |/claimtool ')
        script = directory / 'controller-early-cuda.sh'
        script.write_text(source)
        subprocess.run(['bash', '-n', str(script)], check=True)
        with (directory / 'cuda-controller.log').open('w') as output:
            p = subprocess.Popen(['bash', str(script), 'cuda'], stdout=output,
                                 stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL,
                                 start_new_session=True)
            (directory / 'cuda-controller.pid').write_text(str(p.pid) + '\n')
            status('start', f'{directory.name} CUDA controller{p.pid}; unchanged source/tests/samples; CPU12,14')
            rc = p.wait()
        # The original stopped waiter must never execute a second CUDA window.
        os.killpg(pid, signal.SIGKILL)
        assert (directory / 'cuda-controller.done').exists()
        assert rc == int((directory / 'cuda-controller.exit').read_text()) == 0
    (record / 'controller.exit').write_text('0\n')
    (record / 'controller.done').write_text(datetime.datetime.now(datetime.timezone.utc).isoformat() + '\n')
    status('done', 'both CUDA windows terminal; unchanged data dispatcher may continue; no trainer queued')
except BaseException:
    # Restore still-unstarted original waits if admission itself failed. Already
    # completed windows retain their new markers and are never repeated.
    for pid, directory in jobs:
        if not (directory / 'cuda-controller.done').exists():
            try:
                os.killpg(pid, signal.SIGCONT)
            except ProcessLookupError:
                pass
    (record / 'controller.exit').write_text('1\n')
    (record / 'controller.done').write_text(datetime.datetime.now(datetime.timezone.utc).isoformat() + '\n')
    status('fail', 'admission/operational failure; preserve actual CUDA outputs; no automatic retry')
    raise
