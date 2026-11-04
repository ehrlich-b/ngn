#!/usr/bin/env python3
"""Exact bounded correctness comparison, never a speed/strength estimate."""
import json
import pathlib
import sys
base, candidate = ([json.loads(line) for line in pathlib.Path(p).read_text().splitlines()] for p in sys.argv[1:])
assert len(base) == len(candidate) == 24
assert base == candidate, '\n'.join(str((a,b)) for a,b in zip(base,candidate) if a != b)
assert {r['position'] for r in base} == set(range(12))
assert all(r['depth'] == 4 and r['threads'] == 1 for r in base)
print('PASS: 12 positions x fixed/iterative HCE at depth 4; exact nodes, scores, best moves and PV; one thread')
