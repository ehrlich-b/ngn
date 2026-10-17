"""Rig-only identity checks for two independent speed topics; no timing verdict."""
import json
import os
from pathlib import Path
import queue
import re
import subprocess
import threading
import time

ROOT = Path('/home/ehrli/ngn-data/night-20261007')
S = ROOT / 'session100-speed'
NET = ROOT / 'session82/net-final115/candidate.nnue'
FRAMES = json.loads((ROOT / 'session77-smp-panel/panel.json').read_text())


class UCI:
    def __init__(self, kind, hash_mib):
        self.q = queue.Queue()
        self.errors = []
        self.child = subprocess.Popen(
            ['taskset', '-c', '15', 'nice', '-n', '19', str(S / (kind + '-ngn'))],
            cwd=S, env=dict(os.environ, GOMAXPROCS='1'), stdin=subprocess.PIPE,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, bufsize=1)

        def read():
            for line in self.child.stdout:
                self.q.put(line.strip())
            self.q.put(None)

        self.reader = threading.Thread(target=read)
        self.err_reader = threading.Thread(target=lambda: self.errors.extend(self.child.stderr.read().splitlines()))
        self.reader.start()
        self.err_reader.start()
        self.send('uci')
        self.until('uciok')
        for option in [f'EvalFile value {NET}', 'UseNNUE value true',
                       f'Hash value {hash_mib}', 'Threads value 1', 'OwnBook value false']:
            self.send('setoption name ' + option)
        self.send('isready')
        self.until('readyok')

    def send(self, text):
        self.child.stdin.write(text + '\n')
        self.child.stdin.flush()

    def until(self, prefix):
        lines = []
        while True:
            line = self.q.get(timeout=180)
            assert line is not None, self.errors
            lines.append(line)
            if line.startswith(prefix):
                return lines

    def search(self, row):
        self.send('ucinewgame')
        self.send('position startpos moves ' + ' '.join(row['moves']))
        self.send('isready')
        self.until('readyok')
        self.send('go depth 12')
        lines = self.until('bestmove ')
        info = [line for line in lines if line.startswith('info depth ')][-1]
        return dict(depth=int(re.search(r'\bdepth (\d+)', info)[1]),
                    nodes=int(re.search(r'\bnodes (\d+)', info)[1]),
                    score=re.search(r'\bscore (cp|mate) (-?\d+)', info).group(0),
                    pv=info.split(' pv ', 1)[1] if ' pv ' in info else '',
                    bestmove=lines[-1].split()[1])

    def close(self):
        self.send('quit')
        assert self.child.wait(timeout=15) == 0
        self.reader.join()
        self.err_reader.join()
        assert all('Crash Handler Initialized' in line or 'Crash log:' in line for line in self.errors), self.errors


def main():
    records = []
    for hash_mib in (64, 256):
        engines = [UCI(kind, hash_mib) for kind in ('baseline', 's1', 's2')]
        try:
            for row in FRAMES:
                results = [e.search(row) for e in engines]
                assert results[0] == results[1] == results[2], (hash_mib, row['id'], results)
                records.append(dict(hash_mib=hash_mib, id=row['id'], identity=results[0]))
        finally:
            for e in engines:
                e.close()

    # The established six-position panel also covers fixed-depth and iterative
    # search under the current NGNN3, embedded NGNN2, and HCE evaluators.
    for name, net, depth in [('ngnn3', NET, 14),
                             ('ngnn2', S / 'baseline/engine/nets/ngn-h512-c6c12796.nnue', 14),
                             ('hce', '', 11)]:
        results = []
        for kind in ('baseline', 's1', 's2'):
            command = ['taskset', '-c', '15', 'nice', '-n', '19', str(S / (kind + '-probe')),
                       '-net', str(net), '-mode', 'identity', '-depth', str(depth)]
            raw = subprocess.check_output(command, env=dict(os.environ, GOMAXPROCS='1'), text=True, timeout=900)
            (S / f'identity-{name}-{kind}.jsonl').write_text(raw)
            rows = [json.loads(line) for line in raw.splitlines()]
            assert len(rows) == 12 and all(row['nodes'] > 0 for row in rows)
            results.append([{k: v for k, v in row.items() if k not in ('seconds', 'nps')} for row in rows])
        assert results[0] == results[1] == results[2], name

    # One operational witness, not a speed test: record the TT's huge-page VMA
    # while the actual UCI executable searches with Hash256.
    e = UCI('s1', 256)
    try:
        e.send('position startpos moves ' + ' '.join(FRAMES[0]['moves']))
        e.send('go movetime 5000')
        time.sleep(4)
        smaps = Path(f'/proc/{e.child.pid}/smaps').read_text()
        regions = []
        for line in smaps.splitlines():
            if re.match(r'^[0-9a-f]+-[0-9a-f]+ ', line):
                regions.append({'mapping': line})
            elif line.startswith(('Size:', 'AnonHugePages:', 'VmFlags:')):
                key, value = line.split(':', 1)
                regions[-1][key] = value.strip()
        advised = [r for r in regions if 'hg' in r.get('VmFlags', '').split()]
        huge_kib = sum(int(r.get('AnonHugePages', '0 kB').split()[0]) for r in advised)
        (S / 's1-hugepage-witness.json').write_text(json.dumps(dict(pid=e.child.pid,advised_regions=advised,anon_huge_kib=huge_kib),indent=2)+'\n')
        e.until('bestmove ')
    finally:
        e.close()

    (S / 'identity.json').write_text(json.dumps(dict(verdict='PASS',candidates=['s1','s2'],
        uci_cases_per_candidate=len(records),probe_cases_per_candidate=36,
        width=1,uci_depth=12,uci_records=records,hugepage_witness_kib=huge_kib,
        adoption=False),indent=2)+'\n')


if __name__ == "__main__":
    main()
