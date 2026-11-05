#!/usr/bin/env python3
"""Replay the bounded capture-ordering experiment; all run evidence is external."""
import hashlib
import json
import os
from pathlib import Path
import re
import resource
import shlex
import socket
import subprocess
import sys
import tarfile
import tempfile
import time

REPO = Path(__file__).resolve().parents[3]
FOLDER = Path(__file__).resolve().parent
LANE = REPO.parent
BASE = '3517a353a99bbe627a505ccc5463aadce047431b'
NET = 'engine/nets/ngn-h768-2dcb6d93.nnue'
AUTHOR = 'Bryan Ehrlich <ehrlich.bryan@gmail.com>'
SOURCE = ['engine/search.go', 'engine/wave2_qcapture.go', 'engine/wave2_qcapture_test.go']


def env():
    clean = {k: v for k, v in os.environ.items() if not re.search('credential|token|secret|password', k, re.I)}
    clean.update(GOCACHE='/work/cache/go-build', GOMODCACHE='/work/cache/go-mod',
                 GOPATH='/work/cache/go', GOMAXPROCS='2', TMPDIR=str(LANE/'tmp'),
                 npm_config_cache='/work/cache/npm', UV_CACHE_DIR='/work/cache/uv',
                 UV_PYTHON_INSTALL_DIR='/work/cache/uv-python',
                 OMP_NUM_THREADS='2', MKL_NUM_THREADS='2', OPENBLAS_NUM_THREADS='2',
                 NUMEXPR_NUM_THREADS='2', CUDA_VISIBLE_DEVICES='', PYTHONDONTWRITEBYTECODE='1')
    clean['PATH'] = '/usr/local/go/bin:' + clean.get('PATH', os.defpath)
    return clean


def git(*args):
    return subprocess.check_output(['git', *args], cwd=REPO, env=env(), text=True).strip()


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def execute(args, cwd=REPO, cpu_limit=False):
    log = EVIDENCE / ('%02d.log' % (len(list(EVIDENCE.glob('*.log'))) + 1))
    command = ['nice', '-n', '10', *map(str, args)]
    print('$ ' + shlex.join(command), flush=True)
    start = time.monotonic()
    def limits():
        resource.setrlimit(resource.RLIMIT_CPU, (10, 10))
    with log.open('wb') as output:
        p = subprocess.Popen(command, cwd=cwd, env=env(), stdout=output,
                             stderr=subprocess.STDOUT, preexec_fn=limits if cpu_limit else None)
        _, status, usage = os.wait4(p.pid, 0)
        p.returncode = os.waitstatus_to_exitcode(status)
    stats = dict(command=command, exit_code=p.returncode, elapsed_seconds=time.monotonic()-start,
                 user_seconds=usage.ru_utime, system_seconds=usage.ru_stime, peak_rss_kib=usage.ru_maxrss)
    log.with_suffix('.json').write_text(json.dumps(stats, indent=2)+'\n')
    output = log.read_text()
    print(output, end='')
    print(json.dumps(stats), flush=True)
    if p.returncode:
        raise RuntimeError('command failed; evidence: ' + str(log))
    return output


def focused():
    execute(['go', 'test', '-p', '1', '-count=1', './engine', '-run', '^TestWave2QCapture', '-timeout', '15s', '-v'])


def identities():
    return dict(option_name='W2QCapture', default=0, verdict='DEFERRED',
                baseline_head=BASE, baseline_tree=git('rev-parse', BASE+'^{tree}'),
                baseline_engine_tree=git('rev-parse', BASE+':engine'),
                candidate_engine_tree=git('write-tree', '--prefix=engine/'),
                source_sha256={name: digest(REPO/name) for name in SOURCE},
                net_path=NET, net_sha256=digest(REPO/NET),
                fixture_sha256=digest(FOLDER/'fens.json'), probe_sha256=digest(FOLDER/'probe.go.txt'),
                depth=4, threads=1, hash_mb=16, cpu_limit_per_case_seconds=10,
                claim='Exact off parity only; no speed or Elo claim; rig gate remains undone.')


def parity(record=False):
    scratch = Path(tempfile.mkdtemp(prefix='qcapture.', dir=LANE/'tmp'))
    baseline = scratch/'base'
    baseline.mkdir()
    archive = scratch/'base.tar'
    with archive.open('wb') as output:
        subprocess.run(['git', 'archive', BASE, 'engine', 'internal', 'go.mod'],
                       cwd=REPO, env=env(), stdout=output, check=True)
    with tarfile.open(archive) as source:
        source.extractall(baseline, filter='data')
    probe = scratch/'probe.go'
    probe.write_bytes((FOLDER/'probe.go.txt').read_bytes())
    fens = json.loads((FOLDER/'fens.json').read_text())
    assert len(fens) == 4 and len(set(fens)) == 4
    rows = {}
    for label, root in [('baseline', baseline), ('off', REPO)]:
        binary = scratch/label
        execute(['go', 'build', '-p', '1', '-trimpath', '-o', binary, probe], cwd=root)
        rows[label] = []
        for fen in fens:
            stdout = execute([binary, root/NET, fen], cwd=root, cpu_limit=True)
            row = json.loads(stdout)
            assert row['W2QCapture'] == ('absent' if label == 'baseline' else '0')
            assert (row['depth'], row['threads'], row['hash_mb'], row['backend']) == (4, 1, 16, 'ngnn3')
            rows[label].append(row)
    for base, off in zip(rows['baseline'], rows['off']):
        assert {k:v for k,v in base.items() if k != 'W2QCapture'} == {k:v for k,v in off.items() if k != 'W2QCapture'}, (base, off)
    result = dict(identities=identities(), exact_off_parity=True, results=rows)
    (EVIDENCE/'result.json').write_text(json.dumps(result, indent=2)+'\n')
    if record:
        (FOLDER/'result.json').write_text(json.dumps(result, indent=2)+'\n')
    else:
        assert result == json.loads((FOLDER/'result.json').read_text()), 'replay differs from committed result'
    print('PASS: four FENs, depth4, Threads1 Hash16, explicit H768, exact nodes/score/bestmove; DEFERRED')


def policy():
    assert env()['GOMAXPROCS'] == '2' and env()['OMP_NUM_THREADS'] == '2'
    assert Path(git('rev-parse', '--show-toplevel')) == REPO
    expected = subprocess.check_output(['git', 'show', BASE+':engine/search.go'], cwd=REPO, env=env(), text=True)
    expected = expected.replace('var TunableSearchParams = []TunableParam{',
        'var TunableSearchParams = []TunableParam{\n\t{"W2QCapture", &W2QCapture, 0, 0, 1},')
    expected = expected.replace('capScore[i] = 8*int(captures[i].CapturedPiece().seeWeight()) - int(captures[i].MovingPiece().seeWeight())\n\t\tif captures[i] == ttMove {\n\t\t\tcapScore[i] = 1 << 30\n\t\t}',
        'capScore[i] = info.history.wave2QCaptureScore(captures[i], ttMove)')
    assert (REPO/'engine/search.go').read_text() == expected, 'search eligibility/pruning/learning or ties changed'
    assert not git('diff', BASE, '--', 'engine/moveorder.go', 'engine/movegen.go')
    allowed = {*SOURCE, *(str(p.relative_to(REPO)) for p in FOLDER.iterdir() if p.is_file())}
    changed = set(git('diff', '--name-only', BASE, 'HEAD').splitlines())
    assert changed <= allowed, changed - allowed
    execute(['git', 'diff', '--check', BASE, 'HEAD'])
    result = json.loads((FOLDER/'result.json').read_text())
    assert result['exact_off_parity'] and result['identities']['verdict'] == 'DEFERRED'
    assert identities() == result['identities']
    print('PASS: assigned source, bounded clean CPU environment, unchanged search policy, committed reproducible fixtures/results.')


def provenance():
    rows = git('log', '--format=%an <%ae>|%cn <%ce>|%s', BASE+'..HEAD').splitlines()
    assert rows, 'candidate must be committed'
    for row in rows:
        author, committer, subject = row.split('|', 2)
        assert author == committer == AUTHOR, row
        assert subject.endswith('.') and '\n' not in subject
    assert 'var W2QCapture = 0' in (REPO/'engine/wave2_qcapture.go').read_text()
    assert identities() == json.loads((FOLDER/'result.json').read_text())['identities']
    print('PASS: all candidate commits authored/committed by '+AUTHOR+'; default off; verdict DEFERRED, no Elo claim.')


def execute_on_host(args):
    if args == ['go', 'test', '-p', '2', '-short', '-count=1', './...'] and socket.gethostname() == 'forge':
        print('BLOCKED: kernel tests are prohibited on the exact host forge. Run nice -n 10 go test -p 2 -short -count=1 ./... on a suitable CPU host such as forge2-vm or the rig.', flush=True)
        sys.exit(77)
    execute(args)


if __name__ == '__main__':
    (LANE/'tmp').mkdir(exist_ok=True)
    (LANE/'evidence').mkdir(exist_ok=True)
    EVIDENCE = Path(tempfile.mkdtemp(prefix='qcapture-', dir=LANE/'evidence'))
    print('Evidence: '+str(EVIDENCE), flush=True)
    mode = sys.argv[1]
    if mode == 'focused': focused()
    elif mode in ('record', 'parity'): parity(record=mode == 'record')
    elif mode == 'policy': policy()
    elif mode == 'provenance': provenance()
    elif mode == 'exec':
        execute_on_host(sys.argv[2:])
    else: raise ValueError('unknown mode: '+mode)
