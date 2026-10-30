#!/usr/bin/env python3
"""Verify retained stage-two run records and their referenced training shards."""
import csv
import gzip
import hashlib
import json
import re
import sys
from pathlib import Path

root = Path('..')
evidence = root / 'evidence'
item = sys.argv[1]


def result():
    text = (evidence / 'net-vs-hce.log').read_text()
    games = re.search(r'Games: (\d+)\s+W-D-L: (\d+)-(\d+)-(\d+)', text)
    elo = re.search(r'Penta Elo: ([\d.-]+)\s+95% CI \[([\d.-]+),\s*([\d.-]+)\]', text)
    assert games and elo and 'mode: nodes 1000/move' in text
    counts = list(map(int, games.groups()))
    assert counts[0] == 1000 and sum(counts[1:]) == counts[0]
    return counts, list(map(float, elo.groups()))


def losses(path):
    rows = list(csv.DictReader(path.open()))
    assert rows and all(float(r['train']) >= 0 and float(r['validation']) >= 0 for r in rows)
    return min(float(r['validation']) for r in rows)


if item == 'A3':
    manifest = json.loads((evidence / 'shards.json').read_text())
    assert manifest['source_commit'].startswith('a7252d5')
    assert manifest['elapsed_seconds'] <= 5100
    total = 0
    for shard in manifest['shards']:
        path = Path(shard['path'])
        assert path.parent == Path('/work/cache/ngn-nnue-training/data')
        assert re.fullmatch(r'selfplay-\d{8}T\d{6}Z-\d+\.tsv\.gz', path.name)
        h = hashlib.sha256()
        with path.open('rb') as src:
            for chunk in iter(lambda: src.read(1024 * 1024), b''):
                h.update(chunk)
        assert h.hexdigest() == shard['sha256']
        with gzip.open(path, 'rb') as src:
            count = sum(1 for _ in src)
        assert count == shard['positions']
        total += count
    assert total >= 15_000_000
    training = (evidence / 'training.log').read_text()
    split = re.search(r'positions train/valid: (\d+) (\d+)', training)
    assert split and sum(map(int, split.groups())) == total
    assert int(split[1]) >= 15_000_000
    assert losses(evidence / 'loss.csv') >= 0
    assert sum(map(int, re.findall(r'positions_seen=(\d+)', training))) >= 15_000_000
    assert 0.018 <= int(split[2])/total <= 0.022
    assert 'training_seconds=' in training and float(re.search(r'training_seconds=([\d.]+)', training)[1]) <= 1200
    print('positions:', total, 'train/valid:', split.groups(), 'generation seconds:', manifest['elapsed_seconds'])
elif item == 'A4':
    counts, elo = result()
    print('games/W/D/L:', counts, 'paired Elo/95% interval:', elo)
    print('stage 2 best validation:', losses(evidence / 'loss.csv'), 'stage 1 best validation:', losses(evidence / 'stage1-loss.csv'))
elif item == 'A5':
    counts, elo = result()
    if elo[0] < 0:
        diagnosis = (evidence / 'stage3.md').read_text()
        assert all(term in diagnosis.lower() for term in ('stage 3', 'label', 'learning rate', 'size', 'data'))
        print('stage 3 diagnosis:', diagnosis)
    else:
        print('net did not lose; stage 3 loss diagnosis not applicable:', elo[0])
elif item == 'A6':
    obj = json.loads((root / 'SHOWCASE.json').read_text())
    assert obj['hero'] == 'evidence/loss.png' and (root / obj['hero']).is_file()
    metrics = {m['name']: m for m in obj['metrics']}
    assert set(metrics) == {'Elo of ngn-v1 net vs HCE', 'Training positions', 'HCE Elo vs Counter 5.5'}
    assert all(m['direction'] == 'higher' for m in metrics.values())
    _, elo = result()
    assert metrics['Elo of ngn-v1 net vs HCE']['value'] == elo[0]
    training = (evidence / 'training.log').read_text()
    train_size = int(re.search(r'positions train/valid: (\d+) (\d+)', training)[1])
    seen = sum(map(int, re.findall(r'positions_seen=(\d+)', training)))
    assert metrics['Training positions']['value'] == min(train_size, seen)
    assert metrics['HCE Elo vs Counter 5.5']['value'] == -63
    print(json.dumps(obj, indent=2))
else:
    raise SystemExit('unknown item')
