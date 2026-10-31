"""One native terminal wait; bounded provenance allocation and SEE release."""
import ctypes
import datetime as dt
import json
import os
from pathlib import Path
import select
import subprocess
import sys
import time

root = Path('/home/ehrli/ngn-data/night-20261007')
p = root / 'session74-schedule'
predecessor = root / 'session71-provenance-repaired'
cutoff = dt.datetime.fromisoformat('2026-10-12T21:12:00-04:00')

def write(name, data):
    (p / name).write_text(json.dumps(data, indent=2) + '\n')

def status(event, detail):
    with (p / 'status.jsonl').open('a') as out:
        out.write(json.dumps(dict(utc=dt.datetime.now(dt.timezone.utc).isoformat(), event=event, detail=detail)) + '\n')

def wait():
    libc = ctypes.CDLL(None, use_errno=True)
    fd = libc.inotify_init1(os.O_CLOEXEC | os.O_NONBLOCK)
    assert fd >= 0
    try:
        assert libc.inotify_add_watch(fd, os.fsencode(predecessor), 0x8 | 0x80) >= 0
        status('wait', dict(cutoff=cutoff.isoformat(), score_independent=True))
        while not (predecessor / 'controller.done').exists():
            seconds = (cutoff - dt.datetime.now(dt.timezone.utc)).total_seconds()
            if seconds <= 0:
                # Preserve a natural terminal if it raced the timer.
                if (predecessor / 'controller.done').exists():
                    break
                status('administrative-stop', '48-hour plan allocation reached; no score inspection; entire gate unscored')
                subprocess.run(['systemctl', '--user', 'stop', 'ngn-s71-provenance-repaired.service'], check=True)
                # A completed summary racing service stop retains its original natural rule.
                if not (predecessor / 'controller.done').exists() or not (predecessor / 'gate-summary.json').exists():
                    with (p / 'provenance-input-check-after-stop.txt').open('w') as out:
                        checked = subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], cwd=predecessor, stdout=out, stderr=subprocess.STDOUT)
                    receipt = dict(verdict='ADMINISTRATIVE UNSCORED', certification='OUTSTANDING', score_used_for_stop=False,
                                   cutoff=cutoff.isoformat(), inputs_intact=checked.returncode == 0)
                    write('administrative-stop.json', receipt)
                    (predecessor / 'controller.exit').write_text('143\n')
                    (predecessor / 'controller.done').write_text(dt.datetime.now().astimezone().isoformat() + '\n')
                break
            ready, _, _ = select.select([fd], [], [], seconds)
            if ready:
                os.read(fd, 65536)
    finally:
        os.close(fd)
    status('terminal', 'SEE wrapper reviews integrity independently; no automatic source adoption')


def see_release():
    import importlib.util
    lane = root / 'session72-see-rig'
    os.chdir(lane)
    spec = importlib.util.spec_from_file_location('existing_see', lane / 'games.py')
    games = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(games)
    def reviewed_predecessor():
        libc = ctypes.CDLL(None, use_errno=True)
        fd = libc.inotify_init1(os.O_CLOEXEC | os.O_NONBLOCK)
        assert fd >= 0
        try:
            assert libc.inotify_add_watch(fd, os.fsencode(p), 0x8 | 0x80) >= 0
            games.status('amended-provenance-wait', 'session74 plan: natural clean terminal or intact administrative hold; faults block')
            end = time.monotonic() + 7 * 3600
            while not (p / 'controller.done').exists():
                remaining = end - time.monotonic()
                assert remaining > 0
                assert select.select([fd], [], [], remaining)[0]
                os.read(fd, 65536)
        finally:
            os.close(fd)
        admin_path = p / 'administrative-stop.json'
        admin = json.loads(admin_path.read_text()) if admin_path.exists() else {}
        summary_path = predecessor / 'gate-summary.json'
        summary = json.loads(summary_path.read_text()) if summary_path.exists() else {}
        code = (predecessor / 'controller.exit').read_text().strip()
        aa = json.loads((predecessor / 'aa-summary.json').read_text())
        # Clean natural H0/cap is evidence to diagnose, never a restoration veto.
        natural_clean = code == '0' and summary.get('verdict') in ('CERTIFIED', 'REQUIRES DIAGNOSIS')
        administrative_clean = code == '143' and admin.get('inputs_intact') is True
        passed = aa.get('verdict') == 'aa-pass' and (natural_clean or administrative_clean)
        games.status('amended-predecessor-review', dict(exit=code, verdict=summary.get('verdict') or admin.get('verdict'),
                     integrity_resolved=passed, certification_owed=summary.get('verdict') != 'CERTIFIED',
                     plan='astra-48h-2026-10-12.md; no restoration, continuation or pooling'))
        return passed
    games.wait_for_predecessor = reviewed_predecessor
    games.run()

if __name__ == '__main__':
    if sys.argv[1:] == ['see']:
        see_release()
    else:
        wait()
