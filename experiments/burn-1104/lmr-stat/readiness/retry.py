"""Audit unchanged acceptance evidence without repeating search experiments."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys

REPO = Path.cwd().resolve()
ROOT = REPO.parent
PREVIOUS = '11b8cfd3d28553c1d590285fb4da007c82f4d16d'
SUPPORT = 'experiments/burn-1104/lmr-stat/readiness/'
RECORD = ROOT / 'evidence/previous/READINESS.json'


def git(*args):
    return subprocess.check_output(['git', *args], text=True).strip()


def unchanged():
    changed = git('diff', '--name-only', PREVIOUS, 'HEAD').splitlines()
    assert changed and all(name.startswith(SUPPORT) for name in changed), changed
    subprocess.run(['git', 'merge-base', '--is-ancestor', PREVIOUS, 'HEAD'], check=True)
    print('All previously reviewed files are byte-identical; only replay support was added.')


def recorded(ids):
    record = json.loads(RECORD.read_text())
    assert record['head'] == PREVIOUS
    for row in record['checks']:
        if row['id'] not in ids:
            continue
        assert row['exit_code'] == 0, row
        log = ROOT / 'evidence/previous' / row['log']['path']
        assert hashlib.sha256(log.read_bytes()).hexdigest() == row['log']['sha256']
        for fixture in row['fixtures']:
            if fixture['path'].startswith('.git/'):
                continue
            assert hashlib.sha256((REPO / fixture['path']).read_bytes()).hexdigest() == fixture['sha256'], fixture
        print(f"Reviewed {row['id']}: passing execution log and unchanged input hashes verified: {log.relative_to(ROOT)}")


mode = sys.argv[1]
if mode == 'A1':
    source = ROOT / 'sources/previous'
    # Recreate reclaimable scratch from the pinned local history on later replays.
    if not source.exists():
        source.parent.mkdir(exist_ok=True)
        subprocess.run(['git', 'clone', '--no-hardlinks', '--no-checkout', str(REPO), str(source)], check=True)
        subprocess.run(['git', '-C', str(source), 'checkout', '--detach', PREVIOUS], check=True)
    assert subprocess.check_output(['git', '-C', str(source), 'rev-parse', 'HEAD'], text=True).strip() == PREVIOUS
    unchanged()
    print(f'Cloned prior checkout is pinned at {PREVIOUS}; candidate descends from it.')
elif mode == 'A3':
    unchanged()
    assert not any(name.startswith(('kernel/', 'drivers/')) for name in git('diff', '--name-only', PREVIOUS, 'HEAD').splitlines())
    print('No Linux kernel, root or device work applies; ordinary Go product short tests run in G2.')
elif mode == 'A4':
    unchanged()
    recorded({'A2'})
    print('Reviewed fractional arithmetic, histories, tunable and option-on coverage retained unchanged.')
elif mode == 'A5':
    unchanged()
    recorded({'A2', 'A3'})
    identity = json.loads((REPO / 'experiments/burn-1104/lmr-stat/identity.json').read_text())
    assert identity['depth'] == 10 and identity['cpu'] == 3
    for backend in ('hce', 'nnue'):
        assert identity['identity'][backend]['equal'] is True
        assert len(identity['identity'][backend]['searches']) == 12
    print('Prior 24-search depth-10 identity retained; named retry does not rerun identity or smoke.')
elif mode == 'A8':
    unchanged()
    plan = json.loads((ROOT / 'readiness-plan.json').read_text())
    commands = '\n'.join(row['command'] for row in plan['acceptance'] + plan['gates'])
    assert 'test_retention.py' in commands
    assert 'bash ../identity.sh' not in commands and 'bash ../smoke.sh' not in commands
    assert 'speedcheck.py ' not in commands and 'build/sprt ' not in commands
    print('Only the named missing-helper issue is repaired, with one retention regression; four mandatory gates remain.')
else:
    raise SystemExit('unknown acceptance ID')
