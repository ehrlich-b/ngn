#!/usr/bin/env python3
"""Exercise the accepted executable at widths through eight; no scored games."""
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path('/home/ehrli/ngn-data/night-20261007')
OUT = ROOT / 'session91/threads8'
sys.path.insert(0, str(OUT / 'source/scripts/candidate-match'))
from uci_preflight import Protocol

engine = ROOT / 'session80/candidates/singular/candidate-ngn'
net = ROOT / 'session82/net-final115/candidate.nnue'
env = os.environ.copy()
env.pop('GOMAXPROCS', None)  # Exercise main.go's scheduler callback.
legal = {f'{f}{a}{f}{b}' for f in 'abcdefgh' for a, b in [('2', '3'), ('2', '4')]}
legal.update(['b1a3', 'b1c3', 'g1f3', 'g1h3'])
rows = []
with (OUT / 'uci-transcript.jsonl').open('w') as transcript:
    process = subprocess.Popen([str(engine)], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE, env=env)
    protocol = Protocol(process, transcript)
    try:
        protocol.send('uci')
        options = protocol.until(lambda s: s == 'uciok', 15, 'uci')
        assert any('option name Threads type spin' in s and 'max 64' in s for s in options)
        for command in [f'setoption name EvalFile value {net}', 'setoption name UseNNUE value true',
                        'setoption name OwnBook value false', 'setoption name Hash value 256']:
            protocol.send(command)
        for width in [1, 2, 4, 8, 1, 8]:
            protocol.send(f'setoption name Threads value {width}')
            protocol.send('isready')
            configured = protocol.until(lambda s: s == 'readyok', 15, 'ready')
            assert f'info string threads configured {width} effective {width}' in configured
            protocol.send('ucinewgame')
            protocol.send('position startpos')
            protocol.send('go movetime 80')
            searched = protocol.until(lambda s: s.startswith('bestmove '), 15, 'timed search')
            assert searched[-1].split()[1] in legal
            if width > 1:
                assert f'info string threads configured {width} effective {width}' in searched
            protocol.send('isready')
            barrier = protocol.until(lambda s: s == 'readyok', 15, 'post-search barrier')
            assert not any(s.startswith('bestmove ') for s in barrier)
            protocol.send('position startpos')
            protocol.send('go infinite')
            protocol.until(lambda s: s.startswith('info depth '), 15, 'live search')
            protocol.send('stop')
            stopped = protocol.until(lambda s: s.startswith('bestmove '), 15, 'stop')
            assert stopped[-1].split()[1] in legal
            protocol.send('isready')
            barrier = protocol.until(lambda s: s == 'readyok', 15, 'post-stop barrier')
            assert not any(s.startswith('bestmove ') for s in barrier)
            rows.append(dict(threads=width, hash_mb=256, timed=True, stop=True))
        protocol.send('position startpos')
        protocol.send('go infinite')
        protocol.until(lambda s: s.startswith('info depth '), 15, 'pre-reconfiguration search')
        protocol.send('setoption name Threads value 4')
        protocol.send('setoption name Hash value 128')
        protocol.send('setoption name Hash value 256')
        protocol.send('isready')
        protocol.until(lambda s: s == 'readyok', 15, 'live reconfiguration join')
        protocol.send('setoption name Threads value 8')
        protocol.send('go infinite')
        protocol.until(lambda s: s.startswith('info depth '), 15, 'pre-quit search')
        protocol.send('quit')
        assert process.wait(timeout=15) == 0
        for reader in protocol.threads:
            reader.join(timeout=5)
        assert not protocol.stderr_lines
    finally:
        if process.poll() is None:
            process.kill()  # Only this unscored validation child, never a gate.
            process.wait()
(OUT / 'uci-smoke.json').write_text(json.dumps(dict(
    state='PASS', cases=rows, live_reconfiguration=True, live_quit=True,
    scheduler='main.go callback; inherited GOMAXPROCS removed',
    limitation='CPUs13,15 correctness only; no eight-core occupancy or strength claim'), indent=2) + '\n')
print('PASS: accepted executable widths1/2/4/8, Hash256, timed search, stop/restart, live reconfigure/quit')
