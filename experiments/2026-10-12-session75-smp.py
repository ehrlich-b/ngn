"""Current-package SMP diagnostics; run only on the rig, with no games."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import queue
import re
import subprocess
import threading
import time

ROOT = Path('/home/ehrli/ngn-data/night-20261007')
LANE = ROOT / 'session75'
ENGINE = ROOT / 'session72-see-rig/baseline-ngn'
NET = ROOT / 'session72-see-rig/base.nnue'
FENS = [
    'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
    'r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1',
    'r4rk1/pp1n1ppp/2pbpn2/q7/2BPP3/2N2N2/PPQ2PPP/R1B2RK1 w - - 0 10',
    '8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1',
    '8/5pk1/6p1/8/5P2/5K2/8/8 w - - 0 1',
    'rnbq1rk1/pp2bppp/2pp1n2/4p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 6',
]


def write(name, value):
    (LANE / name).write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (LANE / 'status.jsonl').open('a') as out:
        out.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(), event=event, detail=detail)) + '\n')


class UCI:
    def __init__(self, binary, width, name):
        self.lines = queue.Queue()
        self.log = (LANE / name).open('w')
        self.process = subprocess.Popen(['taskset', '-c', '0,2,4,6', str(binary)],
            cwd=LANE, env=dict(os.environ, GOMAXPROCS=str(width)),
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1)
        self.reader = threading.Thread(target=self.read, daemon=True)
        self.reader.start()

    def read(self):
        for line in self.process.stdout:
            self.log.write(line)
            self.lines.put(line.strip())
        self.lines.put(None)

    def send(self, text):
        self.process.stdin.write(text + '\n')
        self.process.stdin.flush()

    def until(self, prefix, timeout=120):
        end = time.monotonic() + timeout
        lines = []
        while True:
            line = self.lines.get(timeout=max(.01, end - time.monotonic()))
            assert line is not None, ('unexpected engine exit', self.process.poll())
            lines.append(line)
            if line.startswith(prefix):
                return lines

    def close(self):
        self.send('quit')
        code = self.process.wait(timeout=15)
        self.reader.join(timeout=5)
        self.log.close()
        assert code == 0, code

    def cpu(self):
        fields = Path(f'/proc/{self.process.pid}/stat').read_text().split(') ', 1)[1].split()
        return (int(fields[11]) + int(fields[12])) / os.sysconf('SC_CLK_TCK')


def prepare():
    hashes = {str(path): hashlib.sha256(path.read_bytes()).hexdigest() for path in [ENGINE, NET]}
    assert hashes[str(ENGINE)] == '4f8f631fe29eb010605b6eaec2d8c1fac769e39c416f3ff7f41e997feae7a61d'
    assert hashes[str(NET)] == 'a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86'
    write('smp-manifest.json', dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        playing_source='f1d0a03', final14='compiled defaults', files_sha256=hashes,
        machine='rig Ryzen9800X3D WSL2', cpus='0,2,4,6; four distinct physical cores',
        widths=[1, 2, 4], GOMAXPROCS='explicitly equal to each width', Hash=64,
        depth=16, blocks=6, fens=FENS, cells=108, state='ucinewgame clears TT and all worker learning per cell',
        order='all six permutations of widths, one per block; no competing timing/games/training',
        decision='diagnostic only: complete depths, effective-width receipts, CPU occupancy, time-to-depth and total NPS; no Elo or candidate adoption',
        fixed_nodes='not used: go nodes forces a single worker',
        rodent='existing binary UCI handshake only; no source or net read'))


def run():
    manifest = json.loads((LANE / 'smp-manifest.json').read_text())
    for path, pin in manifest['files_sha256'].items():
        assert hashlib.sha256(Path(path).read_bytes()).hexdigest() == pin
    opponent = ROOT / 'session19-rodent3/opponent-only/rodent-v12'
    rodent = UCI(opponent, 1, 'rodent-uci.log')
    rodent.send('uci')
    handshake = rodent.until('uciok')
    rodent.close()
    write('rodent-threads.json', dict(binary=str(opponent),
        binary_sha256=hashlib.sha256(opponent.read_bytes()).hexdigest(),
        supports_threads=any(re.match(r'option name Threads\s', line, re.I) for line in handshake),
        options=[line for line in handshake if line.startswith('option ')], inspection='uci handshake only'))
    engines = {}
    try:
        for width in [1, 2, 4]:
            engine = UCI(ENGINE, width, f'ngn-threads-{width}.log')
            engines[width] = engine
            engine.send('uci')
            engine.until('uciok')
            for option in [f'EvalFile value {NET}', 'UseNNUE value true', 'Hash value 64',
                           f'Threads value {width}', 'OwnBook value false']:
                engine.send('setoption name ' + option)
            engine.send('isready')
            engine.until('readyok')
        status('scaling-start', dict(depth=16, cells=108, widths=[1, 2, 4]))
        records = []
        orders = [[1, 2, 4], [4, 2, 1], [2, 1, 4], [4, 1, 2], [1, 4, 2], [2, 4, 1]]
        for block, order in enumerate(orders):
            for index, fen in enumerate(FENS):
                for width in order:
                    engine = engines[width]
                    engine.send('ucinewgame')
                    engine.send('position fen ' + fen)
                    engine.send('isready')
                    engine.until('readyok')
                    cpu = engine.cpu()
                    start = time.monotonic()
                    engine.send('go depth 16')
                    lines = engine.until('bestmove ')
                    elapsed = time.monotonic() - start
                    cpu_elapsed = engine.cpu() - cpu
                    infos = [line for line in lines if line.startswith('info depth ')]
                    last = infos[-1]
                    depth = int(re.search(r'\bdepth (\d+)', last)[1])
                    nodes = int(re.search(r'\bnodes (\d+)', last)[1])
                    assert depth == 16 and lines[-1].split()[1] not in ['0000', '(none)'], lines
                    if width > 1:
                        assert sum(line == f'info string threads configured {width} effective {width}' for line in lines) == 1, lines
                    row = dict(block=block, position=index, threads=width, depth=depth, elapsed_seconds=elapsed,
                               cpu_seconds=cpu_elapsed, occupied_cpus=cpu_elapsed/elapsed, nodes=nodes,
                               aggregate_nps=nodes/elapsed, bestmove=lines[-1].split()[1])
                    records.append(row)
                    with (LANE / 'scaling.jsonl').open('a') as out:
                        out.write(json.dumps(row) + '\n')
        assert len(records) == 108
        totals = {width: sum(row['elapsed_seconds'] for row in records if row['threads'] == width) for width in engines}
        summary = dict(cells=len(records), total_seconds=totals,
            aggregate_time_to_depth_speedup={width: totals[1]/totals[width] for width in engines},
            occupied_cpus={width: sum(row['cpu_seconds'] for row in records if row['threads'] == width)/totals[width] for width in engines},
            per_position_speedup={width: [
                sum(row['elapsed_seconds'] for row in records if row['threads']==1 and row['position']==index)/
                sum(row['elapsed_seconds'] for row in records if row['threads']==width and row['position']==index)
                for index in range(len(FENS))] for width in [2, 4]},
            verdict='DIAGNOSTIC COMPLETE; strength still requires own AA and fixed400 games',
            limitation='SMP changes visited nodes and primary trajectory; time-to-depth and NPS are not Elo')
        write('scaling-summary.json', summary)
        status('scaling-done', summary)
    finally:
        for engine in engines.values():
            engine.close()


if __name__ == '__main__':
    import sys
    prepare() if sys.argv[1:] == ['prepare'] else run()
