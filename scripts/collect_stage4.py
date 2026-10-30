#!/usr/bin/env python3
"""Summarize finished paired matches, validation and measured memory."""
import json
from pathlib import Path
import re
from check_stage4 import E, match
from plot_stage4 import render

rows=[('Stage 2 vs HCE','stage2-ngn'),('Stage 3 vs HCE','stage3-ngn'),('Stage 3 vs stage 2','stage3-stage2'),('Stage 4 vs HCE','stage4-ngn'),('Stage 4 vs stage 3','stage4-stage3')]
results={name:match(key) for name,key in rows}
lines=['# Paired match results','','500 paired openings, 1,000 nodes per move and a 200-ply cap.','', '| Match | Elo | Paired 95% interval |','| --- | ---: | ---: |']
for name,key in rows:
    value,lo,hi=results[name]
    lines.append(f'| {name} | {value:+.1f} | [{lo:+.0f}, {hi:+.0f}] |')
(E/'comparison.md').write_text('\n'.join(lines)+'\n')
validation=float(re.search(r'exported net holdout blend MSE=([\d.]+)',(E/'validation.log').read_text())[1])
rss=int(re.search(r'Maximum resident set size \(kbytes\):\s*(\d+)',(E/'train.time').read_text())[1])*1024/1e9
model,chosen,mse=max(
    [('Stage 2',results['Stage 2 vs HCE'],.03093784),
     ('Stage 3',results['Stage 3 vs HCE'],.03108383),
     ('Stage 4',results['Stage 4 vs HCE'],validation)],
    key=lambda candidate: candidate[1][0])
render(E/'comparison.png')
showcase={'metrics':[
    {'label':'Elo of best net vs HCE','number':chosen[0],'unit':'Elo','better':'higher','target':0,'note':f'{model}; 1,000 games; paired 95% interval [{chosen[1]:+.0f}, {chosen[2]:+.0f}].'},
    {'label':'Validation MSE on the shared holdout','number':mse,'unit':'MSE','better':'lower','target':0,'note':f'{model} exported net; stage 4 exported MSE {validation:.8f}.'},
    {'label':'Peak RSS','number':round(rss,3),'unit':'GB','better':'lower','target':8,'note':'Stage 4 training process, /usr/bin/time -v under an 8 GB systemd scope.'}],
    'media':{'hero':{'path':'evidence/comparison.png','caption':'Controlled replay and new network: paired Elo and 95% intervals.'},'fanout':[]}}
Path('../SHOWCASE.json').write_text(json.dumps(showcase,indent=2)+'\n')
