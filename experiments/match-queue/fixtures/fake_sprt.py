#!/usr/bin/env python3
"""Record an independent lock probe and emit native cmd/sprt terminal syntax."""
import fcntl
import json
import os
from pathlib import Path
import time


def event(kind):
    with Path(os.environ['MATCH_QUEUE_EVENTS']).open('a') as stream:
        stream.write(json.dumps(dict(event=kind, monotonic=time.monotonic(), pid=os.getpid())) + '\n')


def probe(kind):
    with Path(os.environ['MATCH_QUEUE_LOCK']).open('r') as stream:
        try:
            fcntl.flock(stream, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            event(kind)
        else:
            raise SystemExit('CPU lock was not held')


probe('locked-start')
time.sleep(.01)
probe('locked-end')
if os.environ.get('MATCH_QUEUE_FAKE_FAILURE'):
    raise SystemExit('synthetic harness failure')
print('''=== RESULT (0s) ===
Games: 4   W-D-L: 0-4-0   score: 50.0%
Elo(new - base): +0.0   95% CI [-1, +1]
LLR: +0.00   bounds [-2.94, 2.94]
Pentanomial [LL 0  LD 0  {LW,DD} 2  WD 0  WW 0] over 2 pairs
Penta Elo: +0.0   95% CI [-1, +1]   pLLR +0.00
Flag-outs (lost on time): new 0, base 0  of 4 games
Verdict: INCONCLUSIVE (ran out of games — add openings or raise -maxgames)''')
