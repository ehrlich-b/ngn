#!/usr/bin/env python3
"""Bounded qsearch checks; execution receipts/logs stay outside the checkout."""
import hashlib
import json
import os
from pathlib import Path
import re
import resource
import selectors
import shlex
import socket
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[3]
HERE = Path(__file__).resolve().parent
BASE = 'b704892247bccfa0e236d5119c157e06230b5795'
OUTPUT = ROOT.parent / 'evidence/qchecks'
FRAGMENT = ['nice', '-n', '10', 'go', 'test', '-p', '1', '-count=1', './engine',
            '-run', '^TestWave2(QChecks|GivesCheck)', '-timeout', '15s', '-v']


def environment():
    env = {k: v for k, v in os.environ.items()
           if not re.search(r'credential|token|secret|password', k, re.I)}
    env.update(GOCACHE='/work/cache/go-build', GOMODCACHE='/work/cache/go-mod',
               GOPATH='/work/cache/go', GOMAXPROCS='2',
               npm_config_cache='/work/cache/npm', UV_CACHE_DIR='/work/cache/uv',
               UV_PYTHON_INSTALL_DIR='/work/cache/uv-python',
               TMPDIR=str(ROOT.parent / 'tmp'), OMP_NUM_THREADS='2',
               OPENBLAS_NUM_THREADS='2', MKL_NUM_THREADS='2',
               NUMEXPR_NUM_THREADS='2', CUDA_VISIBLE_DEVICES='',
               PLAYWRIGHT_BROWSERS_PATH='/work/cache/ms-playwright',
               PYTHONDONTWRITEBYTECODE='1')
    env['PATH'] = '/usr/local/go/bin:' + env.get('PATH', os.defpath)
    Path(env['TMPDIR']).mkdir(exist_ok=True)
    return env


def git(*args, root=ROOT):
    return subprocess.check_output(['git', *args], cwd=root,
                                   env=environment(), text=True).strip()


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def receipt(label, args, root, log, started, rc, usage):
    data = dict(command=shlex.join(args), cwd=str(root), head=git('rev-parse', 'HEAD', root=root),
                exit_code=rc, elapsed_seconds=time.monotonic()-started,
                user_seconds=usage.ru_utime, system_seconds=usage.ru_stime,
                peak_rss_kib=usage.ru_maxrss, log_sha256=digest(log),
                host=socket.gethostname(), affinity=sorted(os.sched_getaffinity(0)),
                GOMAXPROCS=2, tensor_threads=2, gpu=False)
    (OUTPUT / (label + '.json')).write_text(json.dumps(data, indent=2)+'\n')
    return data


def run(label, args, root=ROOT):
    if label == 'G2' and socket.gethostname().startswith('forge'):
        print('BLOCKED: the full short suite includes kernel tests; '
              'run '+shlex.join(args)+' on a suitable non-forge host.', flush=True)
        return 75
    OUTPUT.mkdir(parents=True, exist_ok=True)
    log = OUTPUT / (label + '.log')
    started = time.monotonic()
    with log.open('w') as stream:
        stream.write('$ GOMAXPROCS=2 '+shlex.join(args)+'\n')
        stream.flush()
        p = subprocess.Popen(args, cwd=root, env=environment(), stdout=stream,
                             stderr=subprocess.STDOUT)
        _, status, usage = os.wait4(p.pid, 0)
        rc = os.waitstatus_to_exitcode(status)
        p.returncode = rc
    data = receipt(label, args, root, log, started, rc, usage)
    print(json.dumps(data, sort_keys=True), flush=True)
    # Full output is retained, while terminal output stays small on success.
    if rc:
        print(log.read_text()[-12000:], flush=True)
    return rc


def cpu_limit():
    resource.setrlimit(resource.RLIMIT_CPU, (10, 10))


def uci_case(binary, case, label, root, option):
    """One engine per case, one thread, 10 CPU-second hard cap; no speed claim."""
    OUTPUT.mkdir(parents=True, exist_ok=True)
    log = OUTPUT / (label + '.log')
    args = ['nice', '-n', '10', str(binary), '-own-book=false']
    started = time.monotonic()
    p = subprocess.Popen(args, cwd=root, env=environment(), stdin=subprocess.PIPE,
                         stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                         preexec_fn=cpu_limit)
    selector = selectors.DefaultSelector()
    selector.register(p.stdout, selectors.EVENT_READ)
    transcript = bytearray()
    deadline = started + 30

    def send(command):
        p.stdin.write((command+'\n').encode())
        p.stdin.flush()

    def until(pattern):
        while not re.search(pattern, transcript):
            if time.monotonic() >= deadline:
                raise TimeoutError(label)
            if not selector.select(min(1, deadline-time.monotonic())):
                continue
            chunk = os.read(p.stdout.fileno(), 65536)
            if not chunk:
                raise RuntimeError('engine exited before '+str(pattern))
            transcript.extend(chunk)

    error = None
    try:
        send('uci')
        until(rb'(?m)^uciok\r?$')
        assert b'option name UseNNUE type check default true' in transcript
        assert b'option name EvalFile type string default <embedded>' in transcript
        if option:
            assert b'option name W2QChecks type spin default 0 min 0 max 1' in transcript
            send('setoption name W2QChecks value 0')
        send('setoption name Hash value 16')
        send('setoption name Threads value 1')
        send('setoption name OwnBook value false')
        send('ucinewgame')
        send('isready')
        until(rb'(?m)^readyok\r?$')
        send('position fen '+case['fen'])
        send('go depth 4')
        until(rb'(?m)^bestmove [a-h][1-8][a-h][1-8][qrbn]?(?: .*)?\r?$')
        send('quit')
    except BaseException as exc:
        error = exc
        p.kill()
    _, status, usage = os.wait4(p.pid, 0)
    rc = os.waitstatus_to_exitcode(status)
    p.returncode = rc
    selector.close()
    transcript.extend(p.stdout.read())
    p.stdout.close()
    p.stdin.close()
    log.write_bytes(transcript)
    data = receipt(label, args, root, log, started, rc, usage)
    data['cpu_limit_seconds'] = 10
    (OUTPUT / (label+'.json')).write_text(json.dumps(data, indent=2)+'\n')
    if error:
        raise error
    assert rc == 0, data
    output = transcript.decode()
    depths = [line for line in output.splitlines() if line.startswith('info depth 4 ')]
    assert len(depths) == 1, output
    line = depths[0]
    score = re.search(r' score (cp|mate) (-?\d+)', line)
    nodes = re.search(r' nodes (\d+)', line)
    pv = re.search(r' pv (.*)', line)
    best = re.search(r'(?m)^bestmove (\S+)', output)
    result = dict(name=case['name'], fen=case['fen'], score_kind=score[1],
                  score=int(score[2]), nodes=int(nodes[1]), pv=pv[1].split(), bestmove=best[1])
    print(json.dumps(result, sort_keys=True), flush=True)
    return result


def off_cases(label, root, option):
    binary = Path(environment()['TMPDIR']) / ('qchecks-'+label)
    rc = run(label+'-build', ['nice', '-n', '10', 'go', 'build', '-p', '1',
                             '-o', str(binary), '.'], root)
    assert rc == 0
    cases = json.loads((HERE/'off-cases.json').read_text())
    assert len(cases) == 4
    results = [uci_case(binary, case, label+'-'+case['name'], root, option) for case in cases]
    (OUTPUT/(label+'-results.json')).write_text(json.dumps(results, indent=2)+'\n')
    return results


def policy():
    assert git('merge-base', BASE, 'HEAD') == BASE
    src = (ROOT/'engine/search.go').read_text()
    helper = (ROOT/'engine/wave2_qchecks.go').read_text()
    assert re.search(r'W2QChecks\s+= 0', src)
    assert '{"W2QChecks", &W2QChecks, 0, 0, 1}' in src
    for token in ['W2QChecks == 1 && qDepth == 0 && !inCheck', 'var captures [256]Move',
                  'info.QChecksAdmitted +=', 'i < captureCount && standPat+gain+DELTA_MARGIN',
                  'i < captureCount && !seeNonLosingByMVV', 'ttHash ^= wave2QChecksTTKey']:
        assert token in src, token
    assert 'wave2GivesCheck(&pos.Board, move)' in helper
    assert 'move.PromoType() != NoType' in helper
    print('PASS: required base, default-off registry, first-layer policy, disjoint full-capacity suffix, pruning bypass and TT isolation')
    args = FRAGMENT.copy()
    args[args.index('-run')+1] = '^TestWave2QChecks(Production|TTIsolation|OptionTransition)$'
    return run('policy', args)


def artifacts():
    expected = json.loads((HERE/'expected-results.json').read_text())
    manifest = json.loads((HERE/'option-manifest.json').read_text())
    pending = json.loads((HERE/'pending-timed-gate.json').read_text())
    assert expected['baseline_head'] == manifest['candidate_base'] == BASE
    assert manifest['default'] == 0 and pending['status'] == 'pending' and not pending['completed']
    assert manifest['network_sha256'] == digest(ROOT/'engine/nets/ngn-h768-2dcb6d93.nnue')
    assert manifest['cpu_seconds_per_case_limit'] == 10 and not manifest['speed_sample']
    assert len(expected['off_cases']) == 4
    for name in ['check.py','witnesses.json','off-cases.json','option-manifest.json',
                 'pending-timed-gate.json','expected-results.json','README.md']:
        git('ls-files','--error-unmatch',str(HERE.relative_to(ROOT)/name))
    assert os.access(__file__, os.X_OK)
    assert len(os.sched_getaffinity(0)) <= 2 and os.geteuid() != 0
    assert all(environment()[k] == '2' for k in ['GOMAXPROCS','OMP_NUM_THREADS','OPENBLAS_NUM_THREADS','MKL_NUM_THREADS','NUMEXPR_NUM_THREADS'])
    assert not any(re.search(r'credential|token|secret|password',k,re.I) for k in environment())
    assert git('status','--porcelain') == ''
    print('PASS: tracked fixtures/executable/expected results/pending gate; affinity <=2, tensor threads=2, no GPU or speed sample, clean committed checkout')
    return 0


def identity():
    wanted = 'Bryan Ehrlich <ehrlich.bryan@gmail.com>'
    commits = git('log','--format=%an <%ae>|%cn <%ce>',BASE+'..HEAD').splitlines()
    assert commits and all(line == wanted+'|'+wanted for line in commits), commits
    assert git('merge-base',BASE,'HEAD') == BASE
    assert git('branch','--show-current') == 'codex/ngn-w2-qchecks-1104-20261105'
    pending = json.loads((HERE/'pending-timed-gate.json').read_text())
    assert not pending['completed'] and pending['option_default'] == 0
    print('PASS: all new commits authored and committed by '+wanted+'; required branch/base; pending rig gate and default 0; no Elo claim')
    return 0


def main():
    os.sched_setaffinity(0, sorted(os.sched_getaffinity(0))[:2])
    mode = sys.argv[1]
    if mode == 'baseline':
        root = Path(sys.argv[2]).resolve()
        assert git('rev-parse','HEAD',root=root) == BASE
        off_cases('baseline', root, False)
        return 0
    if mode == 'freeze':
        # This is a derived expected-value fixture, not an execution receipt.
        # Actual resource/exit records and transcripts remain outside Git.
        data = dict(baseline_head=BASE, depth=4, backend='embedded H768',
                    off_cases=json.loads((OUTPUT/'baseline-results.json').read_text()),
                    tactical=dict(off=0, on=29999, no_pruning_oracle=29999),
                    result_kind='expected fixture assertions; no strength or speed claim')
        (HERE/'expected-results.json').write_text(json.dumps(data, indent=2)+'\n')
        return 0
    if mode == 'off':
        expected = json.loads((HERE/'expected-results.json').read_text())
        assert off_cases('off', ROOT, True) == expected['off_cases']
        print('PASS: option off reproduces all four baseline depth4 H768 scores/nodes/PVs/bestmoves exactly')
        return 0
    if mode == 'fragment':
        return run('fragment', FRAGMENT)
    if mode == 'policy':
        return policy()
    if mode == 'artifacts':
        return artifacts()
    if mode == 'identity':
        return identity()
    if mode == 'run':
        assert sys.argv[3] == '--'
        return run(sys.argv[2], sys.argv[4:])
    raise ValueError(mode)


if __name__ == '__main__':
    sys.exit(main())
