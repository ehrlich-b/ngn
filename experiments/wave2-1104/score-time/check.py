#!/usr/bin/env python3
"""Replay the non-search acceptance checks without changing the checkout."""
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess
import sys

BASE = '3517a353a99bbe627a505ccc5463aadce047431b'
HERE = Path('experiments/wave2-1104/score-time')


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def git(*args):
    return subprocess.check_output(['git', *args], text=True).strip()


def host():
    # Only an exact hostname of forge has the extra kernel-test restriction.
    hostname = platform.node()
    assert os.geteuid() != 0, 'proof must run without root'
    assert os.environ['GOMAXPROCS'] == '2'
    assert len(os.sched_getaffinity(0)) <= 2
    for name in ('OMP_NUM_THREADS', 'OPENBLAS_NUM_THREADS', 'MKL_NUM_THREADS', 'NUMEXPR_NUM_THREADS'):
        assert os.environ[name] == '2'
    assert os.environ['CUDA_VISIBLE_DEVICES'] == ''
    print(json.dumps({'hostname': hostname, 'exact_forge': hostname == 'forge',
                      'ordinary_go_checks_permitted': True, 'euid': os.geteuid(),
                      'cpu_affinity': sorted(os.sched_getaffinity(0)),
                      'scope': 'named Go checks only; no devices, modules or privileged operations'}))


def artifacts():
    result = json.loads((HERE / 'result.json').read_text())
    for name, digest in result['input_sha256'].items():
        assert sha(Path(name)) == digest, 'stale captured result input: ' + name
    assert sha(HERE / 'result.log') == result['log_sha256']
    assert result['execution']['exit_code'] == 0
    text = (HERE / 'result.log').read_text()
    for marker in ('TRACE stable_move_loses_100cp', 'TRACE improving_score', 'TRACE near_hard_deadline',
                   'TRACE new_search', 'TRACE positive_mate', 'TRACE negative_mate',
                   'CALLBACK run=1 depth=1', 'ASPIRATION completed_depth=', 'FIXED_NODES off/on'):
        assert marker in text, 'missing trace: ' + marker
    assert '\nPASS\n' in text and '\nFAIL' not in text
    manifest = json.loads((HERE / 'timed-match.json').read_text())
    assert manifest['status'] == 'DEFERRED' and manifest['games_run'] == 0
    assert manifest['default_W2ScoreTime'] == 0 and manifest['elo_claim'] is None
    assert manifest['baseline'] == BASE
    subprocess.run(['git', 'diff', '--check', BASE, 'HEAD'], check=True)
    print(json.dumps({'captured_inputs_verified': sorted(result['input_sha256']),
                      'captured_execution': result['execution'],
                      'timed_match': manifest['status'], 'log_sha256': result['log_sha256']}))


def provenance():
    assert git('branch', '--show-current') == 'codex/ngn-w2-score-time-1104-20261105'
    subprocess.run(['git', 'merge-base', '--is-ancestor', BASE, 'HEAD'], check=True)
    commits = git('log', '--format=%H|%an <%ae>|%cn <%ce>', BASE + '..HEAD').splitlines()
    assert commits, 'candidate must be committed'
    for commit in commits:
        _, author, committer = commit.split('|')
        assert author == committer == 'Bryan Ehrlich <ehrlich.bryan@gmail.com>', commit
    manifest = json.loads((HERE / 'timed-match.json').read_text())
    assert manifest['status'] == 'DEFERRED' and manifest['default_W2ScoreTime'] == 0
    print(json.dumps({'head': git('rev-parse', 'HEAD'), 'commits': commits,
                      'default_W2ScoreTime': 0, 'rig_gate': 'DEFERRED', 'elo_claim': None}))


{'host': host, 'artifacts': artifacts, 'provenance': provenance}[sys.argv[1]]()
