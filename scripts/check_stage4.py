#!/usr/bin/env python3
"""Check retained evidence from the controlled NNUE training experiment."""
import gzip
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

E = Path('../evidence')

def require(ok, message):
    if not ok: raise ValueError(message)

def match(name):
    text = (E / ('match-' + name + '.log')).read_text()
    require('over 500 pairs' in text and 'nodes 1000/move | openings: 500 (x2 colors)' in text, name + ': match configuration or result incomplete')
    m = re.search(r'Penta Elo:\s*([+-]?[\d.]+)\s+95% CI \[([+-]?\d+),\s*([+-]?\d+)\]', text)
    require(m is not None, name + ': missing paired interval')
    return tuple(map(float, m.groups()))

def shards(path):
    data = json.loads(path.read_text())['shards']
    for s in data:
        p = Path(s['path']); h = hashlib.sha256()
        with p.open('rb') as f:
            for block in iter(lambda: f.read(1024*1024), b''): h.update(block)
        require(h.hexdigest() == s['sha256'], 'shard hash mismatch: '+str(p))
    return sum(s['positions'] for s in data)

def check(which):
    if which == 'A1':
        prev = 'dd3f51faa1eeb8a9f26edd57d05f6cdf6ce7c5aa'
        subprocess.run(['git','merge-base','--is-ancestor',prev,'HEAD'], check=True)
        require(prev in (E/'previous.log').read_text(), 'previous clone record missing')
    elif which == 'A2':
        require('root: no' in (E/'host.log').read_text(), 'host record missing')
        make = (E/'make-check.log').read_text()
        require('golangci-lint not found' in make and 'Coverage report:' in make, 'make check outcome missing')
        require(all(f'{suite} exit=0' in (E/'suites.log').read_text() for suite in ('unit','race','coverage')), 'ordinary suites incomplete')
    elif which == 'A3':
        require('NVIDIA GeForce RTX 5080' in (E/'device.log').read_text(), 'authorized device missing')
        require(shards(E/'shards.json') >= 15_000_000, 'original shards missing')
        require(shards(E/'new-shards.json') > 0, 'new shards missing')
    elif which == 'A4':
        text = (E/'stage3-summary.md').read_text()
        for value in ('0.03093784', '0.03108383', '-58.9', '-164.5'):
            require(value in text, 'historical measurement missing: '+value)
    elif which == 'A5':
        for name in ('stage2-ngn','stage3-ngn','stage3-stage2'): print(name, match(name))
    elif which == 'A6':
        require(shards(E/'shards.json') >= 15_000_000 and shards(E/'new-shards.json') > 0, 'data incomplete')
        new_count = 0
        for entry in json.loads((E/'new-shards.json').read_text())['shards']:
            count = 0
            with gzip.open(entry['path'], 'rt') as source:
                for line in source:
                    game, fen, score, outcome = line.rstrip().split('\t')
                    require(int(game) % 50 != 0 and 0 <= float(outcome) <= 1, 'new game leaked into holdout or has invalid outcome')
                    int(score); require(bool(fen), 'empty FEN')
                    count += 1
            require(count == entry['positions'], 'new shard position count differs')
            new_count += count
        original_count = sum(map(int, re.findall(r'train=(\d+) holdout=', (E/'prepare-retry.log').read_text())))
        require(original_count >= 15_000_000, 'original train split too small')
        text=(E/'train.log').read_text()
        require('epoch=10 ' in text and 'best=' in text and 'original_shards=11 total_shards=13 units=128 learning_rate=0.0005 schedule=cosine' in text, 'training incomplete or wrong settings')
        seen = re.findall(r'epoch=\d+ .*?seen=(\d+)', text)
        require(len(seen) == 10 and all(int(n) == original_count + new_count for n in seen), 'training count differs from original plus new positions')
        require((E/'stage4.nnue').stat().st_size == 197506, 'export missing')
        require('exported net holdout blend MSE=' in (E/'validation.log').read_text(), 'holdout evaluation missing')
        for name in ('stage4-ngn','stage4-stage3'): print(name, match(name))
    elif which == 'A7':
        require('Running as unit:' in (E/'train.log').read_text(), 'training scope missing')
        text=(E/'train.time').read_text()
        require('Maximum resident set size (kbytes):' in text and 'Exit status: 0' in text, 'training RSS missing')
    elif which == 'A8':
        require(all(f'{suite} exit=0' in (E/'suites.log').read_text() for suite in ('unit','race','coverage')), 'unit/race/coverage failed')
        require('unavailable' in (E/'host.log').read_text(), 'lint status missing')
    elif which == 'A9':
        data=json.loads(Path('../SHOWCASE.json').read_text())
        require([x['label'] for x in data['metrics']] == ['Elo of best net vs HCE','Validation MSE on the shared holdout','Peak RSS'], 'metric labels differ')
        require((Path('..')/data['media']['hero']['path']).read_bytes().startswith(b'\x89PNG\r\n\x1a\n'), 'hero missing')
        validation=re.search(r'exported net holdout blend MSE=([\d.]+)',(E/'validation.log').read_text())
        require(validation is not None, 'stage 4 validation missing')
        candidates=[('Stage 2',match('stage2-ngn'),.03093784),
                    ('Stage 3',match('stage3-ngn'),.03108383),
                    ('Stage 4',match('stage4-ngn'),float(validation[1]))]
        model,expected,mse=max(candidates,key=lambda candidate: candidate[1][0])
        require(data['metrics'][0]['number'] == expected[0], 'Elo metric does not match results')
        require(data['metrics'][1]['number'] == mse, 'MSE metric does not match best network')
        require(all(metric['note'].startswith(model) for metric in data['metrics'][:2]), 'metric notes do not identify best network')
        require(data['metrics'][2]['target'] == 8 and data['metrics'][0]['target'] == 0, 'targets differ')
    elif which == 'A10':
        for seed in (103041,103042):
            text=(E/f'generate-{seed}.log').read_text()
            m=re.search(r'games=(\d+) positions=(\d+) aborted=(\d+) seconds=([\d.]+) nodes=1000 label_nodes=4000',text)
            require(m is not None and int(m[2])>0 and float(m[4]) < 2705, 'generation budget or completion failed')
        for s in json.loads((E/'new-shards.json').read_text())['shards']:
            with gzip.open(s['path'],'rb') as f:
                for _ in f: pass
    elif which == 'A11':
        require('headless' in (E/'decisions.md').read_text(), 'decision record missing')
    else: raise ValueError('unknown check '+which)
    print(which, 'passed')

if __name__ == '__main__': check(sys.argv[1])
