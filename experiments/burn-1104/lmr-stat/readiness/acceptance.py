#!/usr/bin/env python3
"""Small acceptance audits; execution outcomes are emitted by the readiness runner."""
import json
from pathlib import Path
import re
import subprocess
import sys

base = '6d3168b8c5f15532cc7a352d05465a77ab50f361'
experiment = Path('experiments/burn-1104/lmr-stat')

def git(*args):
    return subprocess.check_output(['git', *args], text=True).strip()

def smoke(log, pgn):
    text = Path(log).read_text()
    assert 'setoption name T1104LMRFractional value 1' in text
    assert 'base options: []' in text and 'mode: nodes 1000/move' in text
    assert re.search(r'Games: 2\s+W-D-L: \d+-\d+-\d+', text)
    terms = re.findall(r'\[Termination "([^"]+)"\]', Path(pgn).read_text())
    assert len(terms) == 2 and all(t in ('checkmate', 'stalemate', 'draw-rule', 'max-moves') for t in terms), terms
    assert sorted(re.findall(r'\[White "([^"]+)"\]', Path(pgn).read_text())) == ['base', 'candidate']
    assert len(re.findall(r'^1\. ', Path(pgn).read_text(), re.M)) == 2
    print('Two legal color-swapped fixed-node smoke games; no strength inference.')

mode = sys.argv[1]
if mode == 'host':
    changed = git('diff', '--name-only', base, 'HEAD').splitlines()
    assert set(changed) <= {'engine/search.go', 'engine/lmr_fractional_test.go'} | {str(experiment / f) for f in ('identity.json', 'smoke.txt', 'smoke.pgn', 'smoke-openings.txt', 'nomination.json', 'PRESCREEN.md')}, changed
    assert all(not p.startswith(('kernel/', 'drivers/')) for p in changed)
    print('Only Go chess-search code, tests, and experiment records changed; no Linux kernel/device/root checks apply. Product short suite is gate G2.')
elif mode == 'artifacts':
    required = ('identity.json', 'smoke.txt', 'smoke.pgn', 'smoke-openings.txt', 'nomination.json', 'PRESCREEN.md')
    tracked = set(git('ls-files', str(experiment)).splitlines())
    assert all(str(experiment / f) in tracked for f in required)
    identity = json.loads((experiment / 'identity.json').read_text())
    assert identity['depth'] == 10 and identity['cpu'] == 3
    for backend in ('hce', 'nnue'):
        assert identity['identity'][backend]['equal'] is True
        assert len(identity['identity'][backend]['searches']) == 12
    nomination = json.loads((experiment / 'nomination.json').read_text())
    assert nomination['option'] == 'T1104LMRFractional=1' and nomination['verdict'] == 'DEFERRED'
    assert nomination['games'] == 0 and nomination['penta_elo'] is None and nomination['interval'] is None
    assert all(nomination[k] == 0 for k in ('W', 'D', 'L'))
    assert nomination['pentanomial_counts'] == [0]*5
    requirement = next(line[2:] for line in Path('../BRIEF.md').read_text().splitlines() if line.startswith('- Commit experiments/'))
    assert requirement in (experiment / 'PRESCREEN.md').read_text()
    smoke(experiment / 'smoke.txt', experiment / 'smoke.pgn')
    print('Committed identity, smoke, verbatim rig procedure, and DEFERRED composite nomination verified.')
elif mode == 'smoke':
    smoke(*sys.argv[2:])
elif mode == 'authorship':
    commits = git('log', '--format=%H|%an|%ae|%cn|%ce|%s', base+'..HEAD').splitlines()
    assert commits
    for commit in commits:
        fields = commit.split('|')
        assert fields[1:5] == ['Bryan Ehrlich', 'ehrlich.bryan@gmail.com']*2, commit
        assert fields[5].endswith('.') and '\n' not in fields[5]
        print(commit)
elif mode == 'headless':
    decisions = json.loads(Path('../evidence/decisions.json').read_text())
    assert decisions['mode'] == 'headless' and decisions['questions'] == 0
    assert decisions['prescreen'] == 'deferred-by-brief'
    for name in ('../identity.sh', '../smoke.sh', '../proof.py', '../acceptance.py'):
        assert not re.search(r'\binput\s*\(', Path(name).read_text())
    print('Headless checks use fixed fixture, 1000-node two-game smoke, and no interactive inputs; rig pre-screen deferred as required.')
else:
    raise SystemExit('unknown acceptance mode')
