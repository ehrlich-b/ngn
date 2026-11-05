#!/usr/bin/env python3
"""Reproduce the bounded null-verification proof; logs stay outside the tree."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[3]
FIXTURE = Path(__file__).resolve().parent
BASE = '3517a353a99bbe627a505ccc5463aadce047431b'


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT, text=True).strip()


def manifest():
    data = json.loads((FIXTURE / 'manifest.json').read_text())
    assert data['starting_head'] == BASE
    assert data['default'] == 0 and data['rig_gate']['status'] == 'DEFERRED'
    assert data['elo_claim'] is False
    for name, digest in data['sha256'].items():
        assert sha(ROOT / name) == digest, name
    baseline = json.loads((FIXTURE / 'baseline.json').read_text())
    assert len(baseline) == 4 and all(r['depth'] == 4 and not r['stopped'] for r in baseline)
    assert all(r['net_sha256'] == data['net_sha256'] for r in baseline)
    assert [r['fen'] for r in baseline] == json.loads((FIXTURE / 'fens.json').read_text())
    print('PASS: fixture hashes, four depth4 H768 baselines, default0 and DEFERRED rig gate')


def host():
    name = platform.node()
    # Only an exact hostname match designates the restricted host.
    restricted = name == 'forge'
    print(json.dumps({'hostname': name, 'forge_kernel_tests_blocked': restricted,
                      'ordinary_go_checks': 'permitted', 'privileged_device_checks': 'not requested or run'}))
    manifest()


def fragment(capture=False):
    manifest()
    os.environ.update(OMP_NUM_THREADS='2', OPENBLAS_NUM_THREADS='2', MKL_NUM_THREADS='2',
                      NUMEXPR_NUM_THREADS='2', CUDA_VISIBLE_DEVICES='', PYTHONDONTWRITEBYTECODE='1')
    os.environ['TMPDIR'] = str(ROOT.parent / 'tmp')
    Path(os.environ['TMPDIR']).mkdir(exist_ok=True)
    evidence = ROOT.parent / 'evidence'
    evidence.mkdir(exist_ok=True)
    spec = importlib.util.spec_from_file_location('time_v', ROOT / 'experiments/burn-1104/h768/time_v.py')
    timer = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(timer)
    log = Path(tempfile.mkdtemp(prefix='null-verify-', dir=evidence)) / 'fragment.log'
    command = ['nice', '-n', '10', 'go', 'test', '-p', '1', '-count=1', './engine',
               '-run', '^TestWave2NullVerify', '-timeout', '15s', '-v']
    status = timer.run(command, log)
    output = log.read_text()
    print(output, end='')
    assert status == 0, f'fragment failed: {log}'
    result = {
        'status': 'PASS',
        'tests': re.findall(r'^--- PASS: (TestWave2NullVerify\S+)', output, re.M),
        'witnesses': re.findall(r'^\s+wave2_null_test.go:\d+: (.+)$', output, re.M),
        'baseline_sha256': sha(FIXTURE / 'baseline.json'),
        'rig_gate': 'DEFERRED',
        'elo_claim': False,
    }
    assert len(result['tests']) == 6 and any('actual sparse H768' in s for s in result['witnesses'])
    if capture:
        (FIXTURE / 'result.json').write_text(json.dumps(result, indent=2) + '\n')
    else:
        assert result == json.loads((FIXTURE / 'result.json').read_text()), 'proof result changed'
    print('PASS: exact bounded proof result reproduced; raw log:', log)


def metadata():
    manifest()
    commits = git('log', '--format=%an <%ae>|%cn <%ce>', BASE + '..HEAD').splitlines()
    expected = 'Bryan Ehrlich <ehrlich.bryan@gmail.com>'
    assert commits and all(line == expected + '|' + expected for line in commits)
    assert re.search(r'^var W2NullVerify = 0$', (ROOT / 'engine/wave2_null.go').read_text(), re.M)
    assert '{"W2NullVerify", &W2NullVerify, 0, 0, 1}' in (ROOT / 'engine/search.go').read_text()
    print('PASS: committed candidate author/committer, default0; strength adoption DEFERRED; no Elo claim')
    print('branch=' + git('branch', '--show-current') + ' head=' + git('rev-parse', 'HEAD'))


if __name__ == '__main__':
    os.chdir(ROOT)
    mode = sys.argv[1]
    {'host': host, 'fixtures': manifest, 'fragment': fragment,
     'capture': lambda: fragment(True), 'metadata': metadata}[mode]()
