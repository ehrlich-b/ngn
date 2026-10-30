#!/usr/bin/env python3
"""Check the recorded training and match artifacts without rerunning tournaments."""
import csv
import gzip
import json
import re
import sys
from pathlib import Path

root = Path('..')
evidence = root / 'evidence'
item = sys.argv[1]
if item in ('A2', 'A5'):
    path = evidence / ('baseline-1000.log' if item == 'A2' else 'net-vs-hce.log')
    text = path.read_text()
    match = re.search(r'Games: (\d+)\s+W-D-L: (\d+)-(\d+)-(\d+)', text)
    assert match, 'no completed result'
    games, wins, draws, losses = map(int, match.groups())
    assert games >= (400 if item == 'A2' else 1000) and wins + draws + losses == games
    assert 'mode: nodes 1000/move' in text and 'Penta Elo:' in text and '95% CI' in text
    print(path, match.group(), re.search(r'Penta Elo:.*', text).group())
elif item == 'A3':
    info = (evidence / 'training-data.log').read_text()
    match = re.search(r'games=(\d+) positions=(\d+) seconds=([\d.]+) positions_per_hour=([\d.]+) seed=(\d+) nodes=(\d+) aborted=(\d+)', info)
    assert match and int(match[1]) >= 1 and int(match[2]) > 0 and int(match[6]) == 1000
    count = 0
    with gzip.open(evidence / 'training.tsv.gz', 'rt') as f:
        for row in f:
            game, fen, score, result = row.rstrip().split('\t')
            assert len(fen.split()) == 6 and float(result) in (0.0, 0.5, 1.0)
            int(game); int(score); count += 1
    assert count == int(match[2])
    print(info.strip())
elif item == 'A4':
    lines = list(csv.DictReader((evidence / 'loss.csv').open()))
    assert len(lines) >= 2 and all(float(r['train']) >= 0 and float(r['validation']) >= 0 for r in lines)
    assert (evidence / 'loss.png').read_bytes().startswith(b'\x89PNG\r\n\x1a\n')
    assert (evidence / 'ngn-v1.nnue').read_bytes().startswith(b'NGNNUE\0\0')
    log = (evidence / 'training.log').read_text()
    assert 'NVIDIA GeForce RTX 5080' in log and 'epoch=' in log
    print('loss epochs:', len(lines), 'last:', lines[-1])
elif item == 'A6':
    obj = json.loads((root / 'SHOWCASE.json').read_text())
    assert obj['hero'] == 'evidence/loss.png' and (root / obj['hero']).is_file()
    required = {'Elo of ngn-v1 net vs HCE', 'HCE Elo vs Counter 5.5', 'Training positions'}
    assert {m['name'] for m in obj['metrics']} == required
    assert all(m['direction'] == 'higher' and isinstance(m['value'], (int, float)) for m in obj['metrics'])
    print(json.dumps(obj, indent=2))
else:
    raise SystemExit('unknown item')
