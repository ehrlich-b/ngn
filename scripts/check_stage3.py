#!/usr/bin/env python3
"""Verify retained artifacts and measured results for the training experiment."""
import csv
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

E = Path('../evidence')
def require(condition, message):
    if not condition: raise ValueError(message)
def read(path): return Path(path).read_text()
def check(which):
    if which == 'A1':
        previous = subprocess.check_output(['git','-C','../sources/previous','rev-parse','HEAD'],text=True).strip()
        require(previous == '017c5ded85f5764eabbbfa7d98ea3c432d6c2bf4','previous checkout differs')
        subprocess.run(['git','merge-base','--is-ancestor',previous,'HEAD'],check=True)
    elif which == 'A2':
        require('exit=0' in read(E/'make-check.log'), 'make check did not pass')
        require('exit=0' in read(E/'race.log'), 'race suite did not pass')
    elif which == 'A3':
        require('NVIDIA GeForce RTX 5080' in read(E/'device.log'), 'expected CUDA device absent')
        require(json.loads(read(E/'shards.json'))['origin']=='HCE self-play','dataset provenance mismatch')
    elif which == 'A4':
        require('all Python invocations scoped and measured: yes' in read(E/'memory.md'), 'at least one Python invocation was not scoped and measured')
    elif which == 'A5':
        manifest=json.loads(read(E/'shards.json')); require(sum(s['positions'] for s in manifest['shards'])>=15_000_000,'insufficient data')
        for s in manifest['shards']:
            h=hashlib.sha256()
            with open(s['path'],'rb') as f:
                for block in iter(lambda:f.read(1024*1024),b''): h.update(block)
            require(h.hexdigest()==s['sha256'], 'shard hash mismatch: '+s['path'])
        require(hashlib.sha256((E/'stage2.nnue').read_bytes()).hexdigest()=='c5762cef3565960d7865ac153d5cb80ce7ddff4e6d1ba5935943e1fdbe70edbb','baseline mismatch')
    elif which == 'A6':
        require('exported net holdout blend MSE=0.03093784' in read(E/'baseline-final.log'),'baseline MSE missing')
        require('340490' in read(E/'summary.md') and 'game number divisible by 50' in read(E/'design.md'),'holdout definition missing')
    elif which == 'A7':
        s=json.loads(read(E/'label-counts.json'))
        require(s['positions']>=200000 and s['decisive_nonzero_positions']>0 and s['games']>0,'insufficient label sample')
        require(s['sign_disagreements']<=s['decisive_nonzero_positions'] and s['capped_games']<=s['games'],'invalid label counts')
        require('15.307%' in read(E/'labels.md') and '13.653%' in read(E/'labels.md'),'label report missing')
    elif which == 'A8':
        logs=read(E/'train-retry.log'); minima={}
        for name in ('lr002','lr0005','hidden256','search'):
            with open(E/'ablation'/f'{name}.csv') as f: rows=list(csv.DictReader(f))
            require(len(rows)==10 and f'{name} epoch=10 ' in logs,'incomplete '+name)
            minima[name]=min(float(r['validation']) for r in rows)
        best=min(minima,key=minima.get); summary=read(E/'summary.md')
        require(f'Best by validation: {best}' in summary,'best selection mismatch')
        match=read(E/'match.log')
        require('Games: 1000' in match and 'Penta Elo:' in match,'paired match incomplete')
        require('[-187, -143]' in summary and best in summary,'comparison missing')
    elif which == 'A9':
        obj=json.loads(read('../SHOWCASE.json')); metrics={m['label']:m for m in obj['metrics']}
        for label,direction in [('Elo of best net vs HCE','higher'),('Validation MSE on the shared holdout','lower'),('Peak RSS','lower')]:
            require(label in metrics and metrics[label]['better']==direction and isinstance(metrics[label]['number'],(int,float)),'invalid metric '+label)
        require(metrics['Elo of best net vs HCE']['target']==0 and metrics['Peak RSS']['target']==8,'target mismatch')
        media=[obj['media']['hero']]+obj['media']['fanout']; require(len(media)==5,'plot count mismatch')
        for image in media: require(Path('..',image['path']).read_bytes().startswith(b'\x89PNG\r\n\x1a\n') and image['caption'],'invalid image')
    elif which == 'A10':
        summary=read(E/'summary.md'); require('Budget met: yes' in summary,'budget or finished artifacts not verified')
    elif which == 'A11':
        require('NGN-v1 supports exactly 128 units' in read(E/'design.md'), 'decision record missing')
    else: raise ValueError('unknown check')
    print(which, 'passed')
if __name__=='__main__':
    try: check(sys.argv[1])
    except (ValueError, OSError, subprocess.CalledProcessError, KeyError, IndexError) as exc:
        print(sys.argv[1], 'failed:', exc, file=sys.stderr); sys.exit(1)
