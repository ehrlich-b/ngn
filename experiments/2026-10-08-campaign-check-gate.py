"""Check completed fixed-count campaign logs; emit a reviewable gate summary."""
import json
from pathlib import Path
import re
import sys

mode, path = sys.argv[1:]
assert mode in ('aa', 'gate')
text = Path(path).read_text()
games, wins, draws, losses = map(int, re.findall(r'Games: (\d+)\s+W-D-L: (\d+)-(\d+)-(\d+)', text)[-1])
elo, lower, upper = map(float, re.findall(r'Penta Elo: ([-+\d.]+)\s+95% CI \[([-+\d.]+),\s*([-+\d.]+)\]', text)[-1])
new_flags, base_flags, flag_games = map(int, re.findall(r'Flag-outs \(lost on time\): new (\d+), base (\d+)\s+of (\d+) games', text)[-1])
expected = 40 if mode == 'aa' else 400
assert games == wins + draws + losses == flag_games == expected
assert new_flags == base_flags == 0
assert not re.search(r'(?im)panic|fatal|illegal|no.move|watchdog|timeout|crash|forfeit|engine.error', text)
if mode == 'aa':
    assert lower <= 0 <= upper
print(json.dumps(dict(mode=mode, games=games, wins=wins, draws=draws, losses=losses,
                      elo=elo, ci95=[lower, upper], flags=[new_flags, base_flags],
                      verdict='aa-pass' if mode == 'aa' else ('eligible' if lower > 0 else 'rejected'))))
