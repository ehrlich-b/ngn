"""Fixed pure-speed gate for the campaign's original refresh-cache row pairing."""
import hashlib
import json
import math
import os
from pathlib import Path
import statistics
import subprocess
import sys
import time

root = Path('/home/ehrli/ngn-data/night-20261007/session4')
out = root / 'timing'
out.mkdir(exist_ok=False)
build = json.loads((root / 'build-receipt.json').read_text())
net = Path('/home/ehrli/ngn-data/night-20261007/kb8h512/best.nnue')
old_net = root / 'baseline/engine/nets/ngn-h512-c6c12796.nnue'
bins = {k: root / f'{k}-probe' for k in ('baseline', 'candidate')}
digest = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
assert digest(net) == 'be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
assert all(digest(bins[k]) == build['binary_sha256'][k] for k in bins)
env = dict(os.environ, GOMAXPROCS='1')
commands = []


def run(kind, mode, name, selected_net, extra, expected):
    argv = ['taskset', '-c', '4', 'nice', '-n', '19', str(bins[kind]),
            '-net', str(selected_net) if selected_net else '', '-mode', mode, *extra]
    commands.append(argv)
    with (out / f'{name}.jsonl').open('w') as log, (out / f'{name}.stderr').open('w') as err:
        subprocess.run(argv, env=env, stdout=log, stderr=err, check=True, timeout=600)
    assert not (out / f'{name}.stderr').stat().st_size
    rows = [json.loads(line) for line in (out / f'{name}.jsonl').read_text().splitlines()]
    assert len(rows) == expected and all(r['nodes'] > 0 for r in rows)
    if mode == 'timed':
        assert all(2.94 <= r['seconds'] <= 5 for r in rows)
    return rows


started = time.time()
identities = {}
strip = lambda row: {k: v for k, v in row.items() if k not in ('seconds', 'nps')}
for name, selected_net, depth in [('ngnn3', net, 14), ('ngnn2', old_net, 14), ('hce', None, 11)]:
    rows = {k: run(k, 'identity', f'identity-{name}-{k}', selected_net,
                   ['-depth', str(depth)], 12) for k in bins}
    assert [strip(r) for r in rows['baseline']] == [strip(r) for r in rows['candidate']], name
    identities[name] = {'depth': depth, 'equal': True, 'cases': 12}
(out / 'identity-receipt.json').write_text(json.dumps(identities, indent=2) + '\n')
for kind in bins:
    run(kind, 'timed', f'warmup-{kind}', net, ['-duration', '3s'], 6)
rounds = []
with (out / 'rounds.jsonl').open('w') as journal:
    for i in range(40):
        order = ('baseline', 'candidate') if i % 2 == 0 else ('candidate', 'baseline')
        nps = {}
        for kind in order:
            rows = run(kind, 'timed', f'round-{i:02d}-{kind}', net, ['-duration', '3s'], 6)
            nps[kind] = sum(r['nodes'] for r in rows) / sum(r['seconds'] for r in rows)
        row = dict(index=i, order=list(order), nps=nps, ratio=nps['candidate']/nps['baseline'])
        rounds.append(row)
        journal.write(json.dumps(row) + '\n')
        journal.flush()
logs = [math.log(r['ratio']) for r in rounds]
mean = statistics.mean(logs)
margin = 2.0226909200367604 * statistics.stdev(logs) / math.sqrt(len(logs))
ci = [math.exp(mean-margin), math.exp(mean+margin)]
strata = [math.exp(statistics.mean(logs[i::2])) for i in range(2)]
passed = ci[0] > 1 and all(r > 1 for r in strata)
assert all(digest(bins[k]) == build['binary_sha256'][k] for k in bins)
assert digest(net) == 'be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
assert digest(old_net) == build['old_net_sha256']
receipt = dict(identity=identities, binary_sha256=build['binary_sha256'], net_sha256=digest(net),
               old_net_sha256=digest(old_net), cpu=4, environment={'GOMAXPROCS': '1'}, nice=19,
               seconds=time.time()-started, rounds=rounds, geometric_ratio=math.exp(mean), ci95=ci,
               order_strata_ratios=strata, repeatable_gain=passed, commands=commands)
(out / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n')
print(json.dumps({k: v for k, v in receipt.items() if k not in ('rounds', 'commands')}))
sys.exit(0 if passed else 3)
