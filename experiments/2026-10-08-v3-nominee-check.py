"""WSL-only checks for a clean candidate engine with the unchanged v3 tool."""
import copy
import hashlib
import json
import os
from pathlib import Path
import queue
import re
import shutil
import subprocess
import sys
import threading
import time

ROOT = Path('/home/ehrli/ngn-data/night-20261007')
TOOLING = Path('/home/ehrli/ngn-data/claim-v3-20261008/tooling')
TOOL_SHA = '4cdf39aa3ae964ccbc7b0c7b7a65a4fe71f32992ee35a8326c1f7c0163d41e7c'
TOOL_COMMIT = '376671f842aad7841cd2c743c834c61f702eb0ae'
NET_SHA = 'be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
PGO_SHA = '616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0'
FENS = [
    'r1bq1rk1/pp2bppp/2n2n2/2pp4/3P4/2N1PN2/PPQ1BPPP/R1B2RK1 w - - 0 10',
    'r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1',
    'r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10',
    '8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1',
    '8/5k2/2p2p2/3p4/3P1P2/2P2K2/8/8 w - - 0 1',
    '6k1/5pp1/7p/3q4/8/3Q2P1/5P1P/6K1 w - - 0 1',
]


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def read(path):
    return json.loads(Path(path).read_text())


def write(path, value):
    with Path(path).open('x') as f:
        json.dump(value, f, indent=2)
        f.write('\n')


def clean(path, commit):
    assert subprocess.check_output(['git', '-C', str(path), 'rev-parse', 'HEAD'], text=True).strip() == commit
    assert not subprocess.check_output(['git', '-C', str(path), 'status', '--porcelain', '--untracked-files=all'], text=True).strip()


def source_inputs(s, commit):
    clean(s/'source', commit)
    clean(TOOLING/'source', TOOL_COMMIT)
    assert digest(TOOLING/'frozen/claimtool') == TOOL_SHA
    evidence = read(s/'engine-source-evidence.json')
    files = subprocess.check_output(['git', '-C', str(s/'source'), 'ls-files'], text=True).splitlines()
    actual = {p: digest(s/'source'/p) for p in files
              if p.endswith(('.go', '.s', '.h', '.c', '.nnue', '.pgo')) or p in ('go.mod', 'go.sum')}
    assert len(actual) == len(evidence['inputs']) == 230
    assert actual == {p: v['candidate'] for p, v in evidence['inputs'].items()}
    assert digest(s/'source/default.pgo') == PGO_SHA
    logs = {}
    for name in ('engine-short', 'engine-race', 'full-short', 'full-race', 'vet'):
        p = ROOT/f'session6/{name}.log'
        assert digest(p) == evidence['tests'][name]['sha256'], name
        if name != 'vet':
            assert 'ok  ' in p.read_text() and '\nFAIL' not in p.read_text()
        logs[name] = dict(path=str(p), sha256=digest(p))
    return dict(commit=commit, inputs=actual, inherited_tests=logs,
                evidence_sha256=digest(s/'engine-source-evidence.json'), passed=True)


def artifacts(s, commit):
    for name in ('ngn', 'speedprobe'):
        info = (s/f'frozen/{name}-buildinfo.txt').read_text()
        for required in (f'vcs.revision={commit}', 'vcs.modified=false', 'GOAMD64=v3', 'GOARCH=amd64', 'GOOS=linux', 'go1.25.5', '-pgo='):
            assert required in info, (name, required)
        assert not re.search(r'^\s*dep\s', info, re.M)
        target = '.' if name == 'ngn' else './cmd/speedprobe'
        write(s/f'frozen/{name}-build.json', ['/usr/local/go/bin/go', 'build', '-buildvcs=true', '-p', '2', '-o', str(s/'frozen'/name), target])
        (s/'frozen'/name).chmod(0o555)
    shutil.copyfile(ROOT/'kb8h512/best.nnue', s/'frozen/development-be34532c.nnue')
    assert digest(s/'frozen/development-be34532c.nnue') == NET_SHA
    prior = read(TOOLING/'frozen/development-identity.json')
    write(s/'frozen/provenance.json', prior['Provenance'])
    for a in prior['Provenance'].values():
        assert digest(a['Path']) == a['SHA256']
    common = read(ROOT/'session9b/build-receipt.json')
    assert common['source_reconciled'] and common['engine_inputs'] == 230
    assert digest(ROOT/'session9b/frozen/ngn') == common['binary_sha256']['ngn']
    saved = ROOT/'session6/timing'
    speed = read(saved/'receipt.json')
    assert speed['repeatable_gain'] and sum(v['cases'] for v in speed['identity'].values()) == 36
    assert digest(ROOT/'session6/candidate-probe') == speed['binary_sha256']['candidate']
    paths = [s/'frozen/ngn', s/'frozen/speedprobe', s/'frozen/development-be34532c.nnue',
             s/'frozen/provenance.json', TOOLING/'frozen/claimtool', TOOLING/'frozen/claimtool-build.json',
             ROOT/'session9b/frozen/ngn', ROOT/'session9b/build-receipt.json', saved/'receipt.json']
    paths += [saved/f'identity-{name}-candidate.jsonl' for name in ('ngnn3', 'ngnn2', 'hce')]
    paths += [Path(a['Path']) for a in prior['Provenance'].values()]
    write(s/'checks/artifact-pins.json', {str(p): digest(p) for p in paths})


def pins(s):
    for p, expected in read(s/'checks/artifact-pins.json').items():
        assert digest(p) == expected, p


def uci(s, binary, net, depth, fen, name):
    # A fresh process per case prevents histories/TT from carrying across cases.
    env = dict(os.environ, GOMAXPROCS='1')
    argv = ['taskset', '-c', '13', 'nice', '-n', '19', str(binary)]
    q = queue.Queue()
    lines, commands = [], []
    with (s/f'checks/{name}.stderr').open('w') as err:
        p = subprocess.Popen(argv, cwd='/tmp', env=env, stdin=subprocess.PIPE,
                             stdout=subprocess.PIPE, stderr=err, text=True, bufsize=1)
        def drain():
            for line in p.stdout:
                q.put(line.rstrip('\n'))
            q.put(None)
        thread = threading.Thread(target=drain, daemon=True)
        thread.start()
        def send(line):
            commands.append(line)
            p.stdin.write(line+'\n')
            p.stdin.flush()
        def until(prefix):
            deadline = time.monotonic()+120
            while True:
                remaining = deadline-time.monotonic()
                assert remaining > 0, (name, 'UCI timeout')
                line = q.get(timeout=remaining)
                assert line is not None, (name, 'premature UCI exit')
                lines.append(line)
                assert not re.search(r'(?:error|failed|unknown option)', line, re.I), (name, line)
                if line.startswith(prefix):
                    return line
        try:
            send('uci')
            until('uciok')
            options = dict(Threads='1', Hash='64', OwnBook='false', **{'Move Overhead': '100'})
            if net:
                options['EvalFile'] = str(net)
            options['UseNNUE'] = str(bool(net)).lower()
            for key, value in options.items():
                assert any(l.startswith(f'option name {key} type ') for l in lines), key
                send(f'setoption name {key} value {value}')
            send('isready')
            until('readyok')
            send('ucinewgame')
            send('position fen '+fen)
            send(f'go depth {depth}')
            best = until('bestmove ').split()[1]
            final = [l for l in lines if l.startswith(f'info depth {depth} ')][-1]
            match = re.search(r'info depth (\d+).*? score (cp|mate) (-?\d+) nodes (\d+).*? pv (.*)', final)
            assert match, (name, final)
            result = dict(depth=int(match[1]), score_kind=match[2], score=int(match[3]),
                          nodes=int(match[4]), pv=match[5].split(), best=best)
            assert result['nodes'] > 0 and result['pv'] and result['pv'][0] == best
            send('quit')
            p.wait(timeout=10)
            thread.join(timeout=5)
            assert p.returncode == 0 and not thread.is_alive()
        finally:
            if p.poll() is None:
                p.kill()
                p.wait()
            while not q.empty():
                tail = q.get_nowait()
                if tail is not None:
                    lines.append(tail)
            (s/f'checks/{name}.stdout').write_text('\n'.join(lines)+'\n')
    assert sum(l.startswith('bestmove ') for l in lines) == 1
    assert not re.search(r'(?:panic|fatal|failed|error)', (s/f'checks/{name}.stderr').read_text(), re.I)
    return dict(result=result, commands=commands, argv=argv, environment={'GOMAXPROCS': '1'})


def identities(s):
    pins(s)
    nets = [('ngnn3', s/'frozen/development-be34532c.nnue', 14),
            ('ngnn2', s/'source/engine/nets/ngn-h512-c6c12796.nnue', 14), ('hce', None, 11)]
    checks, commands = {}, []
    for name, net, depth in nets:
        argv = ['taskset', '-c', '13', 'nice', '-n', '19', str(s/'frozen/speedprobe'),
                '-net', str(net) if net else '', '-mode', 'identity', '-depth', str(depth)]
        commands.append(argv)
        path = s/f'checks/speedprobe-{name}.jsonl'
        with path.open('x') as out, (s/f'checks/speedprobe-{name}.stderr').open('x') as err:
            subprocess.run(argv, env=dict(os.environ, GOMAXPROCS='1'), stdout=out, stderr=err,
                           check=True, timeout=300)
        assert not (s/f'checks/speedprobe-{name}.stderr').stat().st_size
        rows = [json.loads(l) for l in path.read_text().splitlines()]
        prior = [json.loads(l) for l in (ROOT/f'session6/timing/identity-{name}-candidate.jsonl').read_text().splitlines()]
        strip = lambda row: {k: v for k, v in row.items() if k not in ('seconds', 'nps')}
        assert len(rows) == len(prior) == 12
        assert all(r['nodes'] > 0 for r in rows)
        assert [strip(r) for r in rows] == [strip(r) for r in prior], name
        checks[name] = dict(cases=12, depth=depth, exact=True)
    uci_checks = []
    for name, net, depth in nets:
        for index, fen in enumerate(FENS):
            results = {kind: uci(s, binary, net, depth, fen, f'uci-{name}-{index}-{kind}')
                       for kind, binary in [('common', ROOT/'session9b/frozen/ngn'), ('clean', s/'frozen/ngn')]}
            assert results['common']['result'] == results['clean']['result'], (name, index)
            uci_checks.append(dict(evaluator=name, position=index, fen=fen, exact=True, results=results))
    pins(s)
    write(s/'checks/identity-receipt.json', dict(passed=True, speedprobe_cases=36, uci_cases=18,
                                              speedprobe=checks, commands=commands, uci=uci_checks))


def plans(s):
    pins(s)
    candidate = s/'frozen/development-identity.json'
    tool = TOOLING/'frozen/claimtool'
    expected = {'screen': (40, 8, 10_000_000_000, 100_000_000, [3317,3359,3557,3713]),
                'stability-screen': (2, 8, 10_000_000_000, 100_000_000, [3713]),
                'stability-confirmation': (2, 8, 120_000_000_000, 1_000_000_000, [3713]),
                'tooling-check': (2, 1, 1_000_000_000, 20_000_000, [3359])}
    def run(mode, identity, out):
        argv = ['taskset', '-c', '13,15', 'nice', '-n', '19', str(tool), 'prepare', '--mode', mode,
                '--candidate', str(identity), '--source', str(TOOLING/'source'),
                '--tool-build-argv', str(TOOLING/'frozen/claimtool-build.json'), '--dry-run', '--out', str(out)]
        result = subprocess.run(argv, capture_output=True, text=True, timeout=120)
        (s/f'checks/{out.name}.log').write_text(result.stdout+result.stderr)
        return result
    for mode, want in expected.items():
        out = s/f'dry-{mode}'
        result = run(mode, candidate, out)
        assert result.returncode == 0, result.stderr
        p = read(out/'dry-run.json')['Plan']
        assert p['Schema'] == 'NGN-CLAIM-3' and p['ClaimThreshold'] == 3600
        assert (p['Count'],p['Concurrency'],p['BaseNS'],p['IncrementNS'],[a['Label'] for a in p['Anchors']]) == want
        assert p['Candidate'] == read(candidate) and p['Runner']['SHA256'] == TOOL_SHA
        assert not (out/'receipt.json').exists() and not (out/'games').exists()
    changes = {
        'engine-digest': ('Engine','SHA256','0'*64,'hash mismatch'),
        'net-digest': ('Net','SHA256','0'*64,'hash mismatch'),
        'source-commit': (None,'SourceCommit','0'*40,'source HEAD mismatch'),
        'go-version': (None,'GoVersion','go1.25.4','Go version mismatch'),
        'build-output': (None,'BuildArgv', ['/usr/local/go/bin/go','build','-buildvcs=true','-p','2','-o',str(s/'wrong-output'),'.'], 'build argv must bind'),
        'provenance-digest': ('Provenance',None,None,'hash mismatch'),
    }
    rejections = {}
    for name, (parent, key, value, error) in changes.items():
        id_copy = copy.deepcopy(read(candidate))
        if name == 'provenance-digest':
            id_copy['Provenance']['parity']['SHA256'] = '0'*64
        else:
            target = id_copy[parent] if parent else id_copy
            target[key] = value
        path = s/f'checks/reject-{name}.json'
        write(path, id_copy)
        out = s/f'rejected-{name}'
        result = run('screen', path, out)
        assert result.returncode != 0 and error in result.stderr, (name, result.stderr)
        assert not out.exists(), name
        rejections[name] = dict(rejected=True, error=result.stderr.strip())
    pins(s)
    write(s/'checks/dry-receipt.json', dict(passed=True, positive_modes=list(expected), rejections=rejections,
                                          tool_sha256=TOOL_SHA, live_games=False, final_nomination=False))


def finish(s, commit):
    assert source_inputs(s, commit) == read(s/'checks/source-receipt.json')
    pins(s)
    assert read(s/'checks/identity-receipt.json')['passed']
    assert read(s/'checks/dry-receipt.json')['passed']
    paths = [p for d in ('frozen','checks') for p in sorted((s/d).rglob('*')) if p.is_file()]
    paths += [s/p for p in ('manifest.md','controller.sh','check.py','source-commit.txt',
                          'preparation-inputs.sha256','engine-source-evidence.json')]
    receipt = dict(passed=True, source_commit=commit, source_clean=True,
                   engine_sha256=digest(s/'frozen/ngn'), profile_sha256=PGO_SHA,
                   original_tool_sha256=TOOL_SHA, original_tool_commit=TOOL_COMMIT,
                   compilation_inputs=230, inherited_tests=read(s/'checks/source-receipt.json')['inherited_tests'],
                   build_argv=read(s/'frozen/ngn-build.json'), build_cpu=[13,15], identity_cpu=13, nice=19,
                   environment=dict(GOAMD64='v3',GOTOOLCHAIN='local',GOPROXY='off',GOMAXPROCS='2',engine_GOMAXPROCS='1'),
                   speedprobe_cases=36, uci_cases=18, dry_modes=4, rejection_controls=6,
                   artifacts={str(p): digest(p) for p in paths}, final_nomination=False, games_queued=False)
    write(s/'build-receipt.json', receipt)
    paths.append(s/'build-receipt.json')
    (s/'frozen-inputs.sha256').write_text(''.join(f'{digest(p)}  {p}\n' for p in paths))
    print(json.dumps({k:v for k,v in receipt.items() if k not in ('artifacts','inherited_tests')}))


if __name__ == '__main__':
    action, root, commit = sys.argv[1:]
    s = Path(root)
    assert sys.platform == 'linux' and s == Path('/home/ehrli/ngn-data/claim-v3-20261008/nominee-session10')
    if action == 'inputs':
        write(s/'checks/source-receipt.json', source_inputs(s, commit))
    elif action == 'artifacts':
        artifacts(s, commit)
    elif action == 'identities':
        identities(s)
    elif action == 'plans':
        plans(s)
    elif action == 'finish':
        finish(s, commit)
    else:
        raise ValueError(action)
