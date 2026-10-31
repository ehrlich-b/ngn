"""Revalidate the already-proved PGO profile with the accepted KB8 H512 net."""
import hashlib
import json
import math
import os
from pathlib import Path
import statistics
import subprocess
import sys
import time

root = Path('/home/ehrli/ngn-data/night-20261007')
out = root / 'session1/pgo-kb8'
out.mkdir(exist_ok=False)
profile = root / 'pgo/cpu.pprof'
net = root / 'kb8h512/best.nnue'
bins = {k: root / f'pgo/probe-{k}' for k in ('nopgo', 'pgo')}
expected = {'nopgo': '238be60f0bd4d73bede7b817de55d661f39915ac662be3abee7b16691a759bb7',
            'pgo': 'fdadca20c838d9997abc879fd3988eeb6353c17df194ed54693f181528655684'}
digest = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
assert digest(net) == 'be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
assert digest(profile) == '616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0'
assert all(digest(bins[k]) == expected[k] for k in bins)
env = dict(os.environ, GOMAXPROCS='1')
commands = []

def run(kind, mode, name, extra):
    argv = ['taskset', '-c', '4', str(bins[kind]), '-net', str(net), '-mode', mode, *extra]
    commands.append(argv)
    with (out / f'{name}.jsonl').open('w') as log, (out / f'{name}.stderr').open('w') as err:
        subprocess.run(argv, env=env, stdout=log, stderr=err, check=True, timeout=300)
    assert not (out / f'{name}.stderr').stat().st_size
    return [json.loads(line) for line in (out / f'{name}.jsonl').read_text().splitlines()]

started = time.time()
identity = {k: run(k, 'identity', f'identity-{k}', ['-depth', '14']) for k in bins}
strip = lambda row: {k: v for k, v in row.items() if k not in ('seconds', 'nps')}
assert len(identity['nopgo']) == len(identity['pgo']) == 12
assert [strip(r) for r in identity['nopgo']] == [strip(r) for r in identity['pgo']]
for kind in bins:
    run(kind, 'timed', f'warmup-{kind}', ['-duration', '2s'])
rounds = []
for i in range(10):
    order = ('nopgo', 'pgo') if i % 2 == 0 else ('pgo', 'nopgo')
    nps = {}
    for kind in order:
        rows = run(kind, 'timed', f'round-{i:02d}-{kind}', ['-duration', '2s'])
        assert len(rows) == 6
        nps[kind] = sum(r['nodes'] for r in rows) / sum(r['seconds'] for r in rows)
    rounds.append(dict(index=i, order=list(order), nps=nps, ratio=nps['pgo']/nps['nopgo']))
logs = [math.log(r['ratio']) for r in rounds]
mean = statistics.mean(logs)
margin = 2.2621571628540993 * statistics.stdev(logs) / math.sqrt(len(logs))
ci = [math.exp(mean-margin), math.exp(mean+margin)]
strata = [math.exp(statistics.mean(logs[i::2])) for i in range(2)]
passed = ci[0] > 1 and all(r > 1 for r in strata)
assert all(digest(bins[k]) == expected[k] for k in bins)
assert digest(net) == 'be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
receipt = dict(identity_cases=12, depth=14, binary_sha256=expected, net_sha256=digest(net),
               profile_sha256=digest(profile), cpu=4, environment={'GOMAXPROCS': '1'},
               seconds=time.time()-started, rounds=rounds, geometric_ratio=math.exp(mean), ci95=ci,
               order_strata_ratios=strata, repeatable_gain=passed, commands=commands)
(out / 'receipt.json').write_text(json.dumps(receipt, indent=2)+'\n')
print(json.dumps({k:v for k,v in receipt.items() if k not in ('rounds', 'commands')}))
sys.exit(0 if passed else 1)
