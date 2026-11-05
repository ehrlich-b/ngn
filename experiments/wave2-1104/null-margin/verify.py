#!/usr/bin/env python3
"""Reproduce the fixed null-margin candidate; raw evidence stays outside the tree."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[3]
FIXTURE = Path(__file__).resolve().parent
PARENT = 'fdd3227fa50b50dea7cdbbeaab7b7c4a4f6f071c'
COMMAND = ['nice', '-n', '10', 'go', 'test', '-p', '1', '-count=1', './engine',
           '-run', '^TestWave2Null(Margin|Verify)', '-timeout', '15s', '-v']


def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT, text=True).strip()


def fixtures():
    config = json.loads((FIXTURE / 'candidate.json').read_text())
    assert config == {'version': 1, 'parent': PARENT,
                      'candidate': {'W2NullVerify': 1, 'W2NullMargin': 1},
                      'default': {'W2NullVerify': 0, 'W2NullMargin': 0},
                      'surplus_cp_per_ply': 200, 'max_extra_plies': 2,
                      'rig_gate': 'DEFERRED', 'elo_claim': False, 'parameter_sweep': False}
    subprocess.run(['git', 'merge-base', '--is-ancestor', PARENT, 'HEAD'], cwd=ROOT, check=True)
    rows = json.loads((FIXTURE / 'parent-trace.json').read_text())
    assert len(rows) == 4 and all(row['depth'] == 4 and not row['stopped'] for row in rows)
    assert [row['fen'] for row in rows] == json.loads((ROOT / 'experiments/wave2-1104/null-verify/fens.json').read_text())
    print('PASS: required parent, four-case trace and predeclared verify=1 margin=1 candidate; rig DEFERRED; no Elo claim')


def fragment(capture=False):
    fixtures()
    os.environ.update(OMP_NUM_THREADS='2', OPENBLAS_NUM_THREADS='2', MKL_NUM_THREADS='2',
                      NUMEXPR_NUM_THREADS='2', CUDA_VISIBLE_DEVICES='', PYTHONDONTWRITEBYTECODE='1',
                      TMPDIR=str(ROOT.parent / 'tmp'))
    Path(os.environ['TMPDIR']).mkdir(exist_ok=True)
    evidence = ROOT.parent / 'evidence'
    evidence.mkdir(exist_ok=True)
    spec = importlib.util.spec_from_file_location('time_v', ROOT / 'experiments/burn-1104/h768/time_v.py')
    timer = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(timer)
    log = Path(tempfile.mkdtemp(prefix='null-margin-', dir=evidence)) / 'fragment.log'
    status = timer.run(COMMAND, log)
    output = log.read_text()
    print(output, end='')
    assert status == 0, f'fragment failed: {log}'
    result = {
        'status': 'PASS',
        'tests': re.findall(r'^--- PASS: (TestWave2Null\S+)', output, re.M),
        'witnesses': re.findall(r'^\s+wave2_null(?:_margin)?_test.go:\d+: (.+)$', output, re.M),
        'parent_trace_sha256': hashlib.sha256((FIXTURE / 'parent-trace.json').read_bytes()).hexdigest(),
        'candidate': {'W2NullVerify': 1, 'W2NullMargin': 1},
        'rig_gate': 'DEFERRED', 'elo_claim': False,
    }
    assert len(result['tests']) == 10, result['tests']
    assert sum(s.startswith('parent trace ') for s in result['witnesses']) == 4
    assert any('actual_child=0 verify_fail_low=1 ordinary=true restored=true' in s for s in result['witnesses'])
    if capture:
        (FIXTURE / 'result.json').write_text(json.dumps(result, indent=2) + '\n')
    else:
        assert result == json.loads((FIXTURE / 'result.json').read_text()), 'deterministic proof result changed'
    print('PASS: exact null-margin proof reproduced; raw log:', log)


def metadata():
    fixtures()
    expected = 'Bryan Ehrlich <ehrlich.bryan@gmail.com>'
    commits = git('log', '--format=%an <%ae>|%cn <%ce>', PARENT + '..HEAD').splitlines()
    assert commits and all(line == expected + '|' + expected for line in commits)
    source = (ROOT / 'engine/wave2_null.go').read_text()
    registry = (ROOT / 'engine/search.go').read_text()
    for option in ('W2NullVerify', 'W2NullMargin'):
        assert re.search(r'^var ' + option + r' = 0$', source, re.M)
        assert '{"' + option + '", &' + option + ', 0, 0, 1}' in registry
    assert not git('status', '--porcelain')
    print('PASS: clean committed candidate, Bryan author/committer, both defaults 0; no adoption or Elo claim')
    print('branch=' + git('branch', '--show-current') + ' head=' + git('rev-parse', 'HEAD'))


if __name__ == '__main__':
    os.chdir(ROOT)
    {'fixtures': fixtures, 'fragment': fragment, 'capture': lambda: fragment(True),
     'metadata': metadata}[sys.argv[1]]()
