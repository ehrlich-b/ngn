"""Positive and falsifying controls for the confirmation clock audit."""
import copy
import importlib.util
from pathlib import Path

spec = importlib.util.spec_from_file_location('confirmation', Path(__file__).with_name('confirmation.py'))
confirmation = importlib.util.module_from_spec(spec)
spec.loader.exec_module(confirmation)


def row(opening, moves, white, ply):
    clock = dict(ply=ply, white=white, go='go wtime 120000 btime 120000 winc 1000 binc 1000',
                 move=moves[-1], elapsed_ms=107, white_before_ms=120000, black_before_ms=120000,
                 white_after_ms=120893 if white else 120000,
                 black_after_ms=120000 if white else 120893,
                 increment_applied_ms=1000, flagged=False)
    return [dict(opening=opening, games=[dict(moves=moves, clocks=[clock])])]


samples = [row('7k/5K2/6Q1/8/8/8/8/8 w - - 0 1', ['g6g7'], True, 1),
           row('f2f3 e7e5 g2g4', ['f2f3', 'e7e5', 'g2g4', 'd8h4'], False, 4)]
for rows in samples:
    assert confirmation.clock_audit(rows)['searched_plies'] == 1
for field, value in [('go', 'go wtime 120001 btime 120000 winc 1000 binc 1000'),
                     ('white_before_ms', 119999), ('black_after_ms', 120001),
                     ('increment_applied_ms', 999), ('flagged', True),
                     ('elapsed_ms', 120301), ('ply', 2), ('white', False),
                     ('move', 'g6g8')]:
    rows = copy.deepcopy(samples[0])
    rows[0]['games'][0]['clocks'][0][field] = value
    try:
        confirmation.clock_audit(rows)
    except AssertionError:
        continue
    raise AssertionError('failed to reject ' + field)
rows = copy.deepcopy(samples[0])
rows[0]['games'][0]['clocks'] = []
try:
    confirmation.clock_audit(rows)
except AssertionError:
    pass
else:
    raise AssertionError('failed to reject missing clocks')
print('PASS: 2 legal color/opening controls and 10 falsifying clock controls')
