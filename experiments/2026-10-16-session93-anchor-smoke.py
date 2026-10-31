"""Unscored v4 anchor preparation; run only on WSL CPUs 13,15."""
import hashlib
import json
import os
from pathlib import Path
import queue
import re
import subprocess
import threading
import time

ROOT = Path('/home/ehrli/ngn-data/claim-v4-20261016')
OUT = Path('/home/ehrli/ngn-data/night-20261007/session93')
ANCHORS = [
    ('Koivisto 6.0', ROOT / 'anchors/koivisto6/Koivisto_6.0-x64-linux-avx2'),
]
LEGAL_START = {f'{f}{r}{f}{s}' for f in 'abcdefgh' for r, s in [('2', '3'), ('2', '4')]}
LEGAL_START.update(['b1a3', 'b1c3', 'g1f3', 'g1h3'])
results = []

for name, binary in ANCHORS:
    events, stderr, lines = [], [], queue.Queue()
    proc = subprocess.Popen([str(binary)], cwd='/tmp', env={**os.environ, 'GOMAXPROCS': '8'},
                            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            text=True, bufsize=1)
    started = time.monotonic()

    def read(stream, label):
        for line in stream:
            line = line.rstrip('\n')
            events.append([time.monotonic() - started, label, line])
            if label == 'stdout':
                lines.put(line)
            else:
                stderr.append(line)
        if label == 'stdout':
            lines.put(None)

    readers = [threading.Thread(target=read, args=(proc.stdout, 'stdout'), daemon=True),
               threading.Thread(target=read, args=(proc.stderr, 'stderr'), daemon=True)]
    for reader in readers:
        reader.start()

    def send(line):
        events.append([time.monotonic() - started, 'stdin', line])
        proc.stdin.write(line + '\n')
        proc.stdin.flush()

    def until(token, timeout=20):
        captured = []
        deadline = time.monotonic() + timeout
        while True:
            line = lines.get(timeout=max(.001, deadline - time.monotonic()))
            assert line is not None, f'{name}: EOF before {token}'
            captured.append(line)
            if line == token or line.startswith(token + ' '):
                return captured

    try:
        send('uci')
        handshake = until('uciok')
        options = dict(re.match(r'option name (.+) type (.+)', x).groups()
                       for x in handshake if x.startswith('option name '))
        selected = {}
        for key, desc in options.items():
            match = re.search(r' default (.*?)(?: min | max | var |$)', desc)
            if match and match.group(1):
                selected[key] = match.group(1)
        for key, value in [('Threads', 8), ('Hash', 256)]:
            desc = options[key]
            low, high = re.search(r' min (\d+) max (\d+)', desc).groups()
            assert int(low) <= value <= int(high), (name, key, desc)
            selected[key] = str(value)
        for key in options:
            if key.lower() in ('ownbook', 'usebook', 'book', 'ponder'):
                selected[key] = 'false'
            if key.lower() == 'move overhead':
                selected[key] = '100'
        for key in ['EvalFile', 'UseNNUE'] + sorted(k for k in selected if k not in ('EvalFile', 'UseNNUE')):
            if key in selected:
                send(f'setoption name {key} value {selected[key]}')
        send('isready')
        until('readyok')
        moves = []
        for _ in range(2):
            send('ucinewgame')
            send('isready')
            until('readyok')
            send('position startpos')
            send('go movetime 200')
            move = until('bestmove')[-1].split()[1]
            assert move in LEGAL_START, (name, move)
            moves.append(move)
        send('quit')
        assert proc.wait(timeout=10) == 0
        for reader in readers:
            reader.join(timeout=2)
        errors = [x for _, stream, x in events if stream != 'stdin' and
                  re.search(r'\b(error|panic|fatal|unknown option|illegal move)\b', x, re.I)]
        assert not errors, (name, errors)
        record = dict(name=name, binary=str(binary), sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),
                      status='PASS', selected=selected, options=options, moves=moves,
                      exit_code=proc.returncode, stderr=stderr, events=events,
                      note='Unscored start-position smoke on CPUs13,15; not an eight-core occupancy or strength measurement')
        results.append(record)
        (OUT / f'{binary.parent.name}-smoke.json').write_text(json.dumps(record, indent=2) + '\n')
        print(name, 'PASS', record['sha256'], flush=True)
    finally:
        if proc.poll() is None:
            proc.kill()
            proc.wait()

(OUT / 'anchor-smokes.json').write_text(json.dumps(results, indent=2) + '\n')
