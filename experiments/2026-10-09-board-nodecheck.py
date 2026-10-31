"""Linux UCI version of the canonical nodecheck, using the current paired base."""
import json
import re
import subprocess
from pathlib import Path

root = Path('/home/ehrli/ngn-data/night-20261007/session19-board2')
net = root.parent / 'session18d/base.nnue'
cases = [
    ('kiwipete', 'r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1', 12),
    ('mid', 'r1bq1rk1/pp2bppp/2n2n2/2pp4/3P4/2N1PN2/PPQ1BPPP/R1B2RK1 w - - 0 10', 12),
    ('end', '8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1', 16),
]


def run(kind, name, fen, depth):
    p = subprocess.Popen([str(root / (kind + '-ngn'))], stdin=subprocess.PIPE,
                         stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    lines = []

    def send(s):
        p.stdin.write(s + '\n')
        p.stdin.flush()

    def read_until(prefix):
        while True:
            s = p.stdout.readline()
            assert s, (kind, name, 'engine ended before', prefix)
            lines.append(s.rstrip())
            if s.startswith(prefix):
                return s.rstrip()

    try:
        send('uci')
        read_until('uciok')
        for n, v in [('Threads', '1'), ('Hash', '64'), ('OwnBook', 'false'),
                     ('UseNNUE', 'true'), ('EvalFile', str(net))]:
            send(f'setoption name {n} value {v}')
        send('isready')
        read_until('readyok')
        send('debug on')
        send('position fen ' + fen)
        send(f'go depth {depth}')
        best = read_until('bestmove')
        stats = [s for s in lines if s.startswith('info string stats')]
        assert stats, (kind, name, 'missing cumulative node stats')
        nodes = int(re.search(r'\bnodes (\d+)', stats[-1]).group(1))
        info = [s for s in lines if s.startswith(f'info depth {depth} ')]
        assert info, (kind, name, 'requested depth not completed')
        score = re.search(r'\bscore (?:cp|mate) -?\d+', info[-1]).group(0)
        pv = info[-1].split(' pv ', 1)[1]
        send('quit')
        assert p.wait(timeout=10) == 0
        (root / (kind + '-nodecheck-' + name + '.log')).write_text('\n'.join(lines) + '\n')
        return dict(name=name, depth=depth, nodes=nodes, score=score, pv=pv, best=best)
    finally:
        if p.poll() is None:
            p.kill()
            p.wait()


rows = {k: [run(k, *c) for c in cases] for k in ('baseline', 'candidate')}
print(json.dumps(dict(identical=rows['baseline'] == rows['candidate'], rows=rows), indent=2))
