"""Fixed resource sample for an eval candidate; no proxy acceptance/rejection."""
import hashlib
import json
import math
from pathlib import Path
import statistics
import subprocess

root = Path('/home/ehrli/ngn-data/night-20261007/session24-continuation')
net = Path('/home/ehrli/ngn-data/night-20261007/kb8h512/best.nnue')


def sample(side, name):
    argv = ['taskset', '-c', '14', 'nice', '-n', '19', str(root / (side+'-probe')),
            '-mode', 'timed', '-duration', '1s', '-net', str(net)]
    with (root / (name+'.jsonl')).open('w') as out, (root / (name+'.err')).open('w') as err:
        subprocess.run(argv, stdin=subprocess.DEVNULL, stdout=out, stderr=err,
                       check=True, timeout=60)
    assert not (root / (name+'.err')).stat().st_size
    rows = [json.loads(line) for line in (root / (name+'.jsonl')).read_text().splitlines()]
    assert len(rows) == 6 and [row['position'] for row in rows] == list(range(6))
    assert all(row['kind']=='iterative' and row['nodes']>0 and row['seconds']>0 for row in rows)
    return sum(row['nodes'] for row in rows)/sum(row['seconds'] for row in rows)


manifest = dict(base_source='39eb128', net_sha256=hashlib.sha256(net.read_bytes()).hexdigest(),
                cpus=[14], gomaxprocs=1, nice=19, sample_blocks=12,
                schedule=['AB' if i % 2 == 0 else 'BA' for i in range(12)],
                warmups=['baseline','candidate'], seconds_per_position=1,
                fixtures='unchanged six cmd/speedprobe positions',
                argv_by_side={side: ['taskset','-c','14','nice','-n','19',str(root/(side+'-probe')),
                                     '-mode','timed','-duration','1s','-net',str(net)]
                              for side in ['baseline','candidate']},
                diagnostic_only=True, stop_rule='exactly12 blocks; no extension or verdict threshold')
(root/'resource-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
sample('baseline', 'warm-baseline')
sample('candidate', 'warm-candidate')
blocks = []
for block in range(12):
    order = ['baseline', 'candidate'] if block % 2 == 0 else ['candidate', 'baseline']
    nps = {side: sample(side, f'block{block:02d}-{side}') for side in order}
    blocks.append(dict(block=block, order=order, nps=nps, ratio=nps['candidate']/nps['baseline']))
logs = [math.log(block['ratio']) for block in blocks]
mean, se = statistics.mean(logs), statistics.stdev(logs)/math.sqrt(12)
result = dict(blocks=blocks, sample_blocks=12, geometric_ratio=math.exp(mean),
              ci95=[math.exp(mean-2.200985*se),math.exp(mean+2.200985*se)], t_df=11,
              order_strata=[math.exp(statistics.mean(logs[start::2])) for start in (0,1)],
              added_worker_table_bytes=2*16384*2, diagnostic_only=True,
              limitation='changed search paths and concurrent workload; no speed or Elo verdict')
(root/'resource.json').write_text(json.dumps(result,indent=2,allow_nan=False)+'\n')
print(json.dumps(result,allow_nan=False))
