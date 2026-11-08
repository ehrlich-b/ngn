#!/usr/bin/env python3
"""Tiny UCI peer: legal mate fallback after an optional search error."""
from pathlib import Path
import sys

fault_switch = Path(sys.argv[1])
for line in sys.stdin:
    line = line.strip()
    if line == 'uci':
        print('uciok', flush=True)
    elif line == 'isready':
        print('readyok', flush=True)
    elif line.startswith('go '):
        if line == 'go movetime 200':
            print('bestmove e2e4', flush=True)
            continue
        print('info depth 1 score cp 100', flush=True)
        if fault_switch.read_text().strip() == 'fault':
            print('info string error search session failed: fixture panic', flush=True)
        print('bestmove g6g7', flush=True)
    elif line == 'quit':
        break
