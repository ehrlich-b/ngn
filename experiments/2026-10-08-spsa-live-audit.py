#!/usr/bin/env python3
"""Bounded diagnostic of the frozen tuner's actual engine; no games or tuning."""
import concurrent.futures
import datetime
import hashlib
import json
import os
from pathlib import Path
import queue
import re
import subprocess
import sys
import threading
import time

ROOT = Path('/home/ehrli/ngn-data/night-20261007')
BINARY = ROOT / 'kb8bin/ngn-kb8'
SOURCE = ROOT / 'kb8bin/src'
POSITIONS = [
    'r1bq1rk1/pp2bppp/2n2n2/2pp4/3P4/2N1PN2/PPQ1BPPP/R1B2RK1 w - - 0 10',
    'r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1',
    'r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10',
    '8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1',
    '8/5k2/2p2p2/3p4/3P1P2/2P2K2/8/8 w - - 0 1',
    '6k1/5pp1/7p/3q4/8/3Q2P1/5P1P/6K1 w - - 0 1',
]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n')


def registry():
    text = (SOURCE / 'engine/search.go').read_text()
    rows = re.findall(r'\{"(\w+)", &(\w+), (\d+), (\d+), (\d+)\}', text)
    assert len(rows) == 14 and len({r[0] for r in rows}) == 14
    lines = text.splitlines()
    params = []
    for name, variable, default, minimum, maximum in rows:
        # Record executable references, not declarations, registry rows or comments.
        reads = [i for i, line in enumerate(lines, 1)
                 if re.search(r'\b' + variable + r'\b', line.split('//')[0])
                 and not re.search(r'\b' + variable + r'\s*=', line.split('//')[0])
                 and not line.lstrip().startswith('{')]
        assert reads, (name, 'no executable source reference')
        params.append(dict(name=name, variable=variable, default=int(default),
                           minimum=int(minimum), maximum=int(maximum), read_lines=reads))
    return params


def parse_result(lines, best):
    final = [line for line in lines if line.startswith('info depth ')][-1]
    m = re.search(r'info depth (\d+).*? score (cp|mate) (-?\d+) nodes (\d+).*? pv (.*)', final)
    assert m, final
    result = dict(depth=int(m[1]), score_kind=m[2], score=int(m[3]), nodes=int(m[4]),
                  pv=m[5].split(), best=best)
    assert result['nodes'] > 0 and result['pv'] and result['pv'][0] == best
    return result


def probe(out, params, name, override, cpu):
    commands, lines, results = [], [], []
    q = queue.Queue()
    argv = ['taskset', '-c', str(cpu), 'nice', '-n', '19', str(BINARY)]
    with (out / (name + '.stderr')).open('w') as err:
        p = subprocess.Popen(argv, cwd=out, env=dict(os.environ, GOMAXPROCS='1'),
                             stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=err,
                             text=True, bufsize=1)
        def drain():
            for line in p.stdout:
                q.put(line.rstrip('\n'))
            q.put(None)
        thread = threading.Thread(target=drain, daemon=True)
        thread.start()
        def send(line):
            commands.append(line)
            p.stdin.write(line + '\n')
            p.stdin.flush()
        def until(prefix):
            deadline = time.monotonic() + 30
            while True:
                remaining = deadline - time.monotonic()
                assert remaining > 0, (name, 'UCI timeout')
                line = q.get(timeout=remaining)
                assert line is not None, (name, 'premature exit')
                lines.append(line)
                assert not re.search(r'error|failed|unknown option', line, re.I), (name, line)
                if line.startswith(prefix):
                    return line
        try:
            send('uci')
            until('uciok')
            for param in params:
                assert (f"option name {param['name']} type spin default {param['default']} "
                        f"min {param['minimum']} max {param['maximum']}") in lines
            for option in ('OwnBook type check default false', 'UseNNUE type check default true',
                           'Threads type spin default 1 min 1 max 64',
                           'Hash type spin default 128 min 1 max 1024'):
                assert 'option name ' + option in lines, option
            # Preserve the actual tuning process defaults, including its embedded net.
            if override:
                send(f'setoption name {override[0]} value {override[1]}')
            send('isready')
            until('readyok')
            for fen in POSITIONS:
                send('ucinewgame')
                send('isready')
                until('readyok')
                send('position fen ' + fen)
                start = len(lines)
                send('go depth 14 nodes 1000000')
                best = until('bestmove ').split()[1]
                results.append(parse_result(lines[start:], best))
            send('quit')
            p.wait(timeout=10)
            thread.join(timeout=5)
            assert p.returncode == 0 and not thread.is_alive()
        finally:
            if p.poll() is None:
                p.kill()
                p.wait(timeout=10)
            write(out / (name + '.json'), dict(argv=argv, commands=commands,
                                               stdout=lines, returncode=p.returncode,
                                               override=override, results=results))
    assert (out / (name + '.stderr')).stat().st_size == 0, name
    return results


def main(out):
    params = registry()
    paths = [BINARY, ROOT / 'spsa/spsa', ROOT / 'session1/spsa-resume.cmd']
    paths += [SOURCE / p for p in ('engine/search.go', 'engine/uci.go', 'engine/default_net.go',
                                  'internal/uci/uci.go', 'cmd/spsa/main.go')]
    # The recorded default filename is discovered from frozen source, never guessed.
    default_text = (SOURCE / 'engine/default_net.go').read_text()
    net_name = re.search(r'//go:embed (nets/\S+\.nnue)', default_text)[1]
    source_net = SOURCE / 'engine' / net_name
    assert digest(source_net) == 'be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
    paths.append(source_net)
    before = {str(p): digest(p) for p in paths}
    write(out / 'source-pins.json', before)
    write(out / 'registry.json', params)
    first = probe(out, params, 'defaults-before', None, 13)
    jobs = [(p['name'] + '-' + side, (p['name'], p[side]), 13 if side == 'minimum' else 15)
            for p in params for side in ('minimum', 'maximum')]
    # At most two bounded engines, one per preparation CPU; no matches or trainer changes.
    results = {}
    with concurrent.futures.ThreadPoolExecutor(max_workers=1) as left, \
            concurrent.futures.ThreadPoolExecutor(max_workers=1) as right:
        pools = {13: left, 15: right}
        futures = {pools[cpu].submit(probe, out, params, name, override, cpu): name
                   for name, override, cpu in jobs}
        for future in concurrent.futures.as_completed(futures):
            results[futures[future]] = future.result()
    last = probe(out, params, 'defaults-after', None, 15)
    assert first == last, 'unchanged defaults did not reproduce exactly'
    assert before == {str(p): digest(p) for p in paths}, 'frozen inputs changed'
    witnesses = {}
    for param in params:
        name = param['name']
        witnesses[name] = [dict(side=side, position=i, baseline=first[i], changed=record)
                           for side in ('minimum', 'maximum')
                           for i, record in enumerate(results[name + '-' + side]) if record != first[i]]
    receipt = dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                   kind='live-option diagnostic; no games, convergence or Elo verdict',
                   parameters=14, configurations=30, searches=180, depth_cap=14, node_cap=1000000,
                   cpu=[13, 15], nice=19, gomaxprocs=1, source_pins=before,
                   default_identity=True, advertised_options=True,
                   witnessed=[name for name, w in witnesses.items() if w],
                   unwitnessed=[name for name, w in witnesses.items() if not w], witnesses=witnesses)
    write(out / 'audit-receipt.json', receipt)
    print(json.dumps({k: v for k, v in receipt.items() if k not in ('source_pins', 'witnesses')}))


if __name__ == '__main__':
    main(Path(sys.argv[1]))
