"""Reconcile the two completed, fixed speed samples without rerunning engines."""
import hashlib
import json
import math
from pathlib import Path
import statistics

ROOT = Path(__file__).resolve().parent
RESULTS = ROOT / '2026-10-09-session34-results'


def load(path):
    return json.loads(path.read_text())


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def close(a, b):
    assert math.isclose(a, b, rel_tol=1e-12), (a, b)


def interval(ratios, t):
    logs = [math.log(x) for x in ratios]
    center = statistics.mean(logs)
    margin = t * statistics.stdev(logs) / math.sqrt(len(logs))
    return dict(geometric_ratio=math.exp(center),
                ci95=[math.exp(center-margin), math.exp(center+margin)],
                order_strata_ratios=[math.exp(statistics.mean(logs[i::2])) for i in range(2)])


def nps(path):
    rows = [json.loads(line) for line in path.read_text().splitlines()]
    assert len(rows) == 6
    assert all(r['nodes'] > 0 and 2.94 <= r['seconds'] <= 5 for r in rows)
    assert not path.with_suffix('.stderr').stat().st_size
    return sum(r['nodes'] for r in rows) / sum(r['seconds'] for r in rows)


for name, topic, count, t, modes in [
    ('session32-vnni512', 'campaign-ngnn4-vnni512', 20, 2.093024054408263,
     ['ngnn4', 'ngnn3', 'ngnn2', 'hce']),
    ('session33-null-frame', 'campaign-null-frame', 40, 2.0226909200367604,
     ['ngnn3', 'ngnn2', 'hce']),
]:
    run = RESULTS / name
    receipt = load(run / 'timing/receipt.json')
    build = load(run / 'build-receipt.json')
    terminal = 'speed-controller' if count == 20 else 'controller'
    assert (run / f'{terminal}.exit').read_text().strip() == '0'
    assert (run / f'{terminal}.done').stat().st_size
    for f, expected in build['source_sha256'].items():
        assert sha(ROOT.parent / 'output' / topic / f) == expected, f
    assert sha(run / 'candidate.patch') == build['patch_sha256']
    pin = 'speed-input' if count == 20 else 'input'
    for when in ['before', 'after']:
        lines = (run / f'{pin}-check-{when}.txt').read_text().splitlines()
        assert lines and all(x.endswith(': OK') for x in lines)
    for mode in modes:
        paths = [run / f'timing/identity-{mode}-{kind}.jsonl' for kind in ['baseline', 'candidate']]
        rows = [[{k: v for k, v in json.loads(line).items() if k not in ['nps', 'seconds']}
                 for line in p.read_text().splitlines()] for p in paths]
        assert len(rows[0]) == 12 and rows[0] == rows[1]
        assert all(not p.with_suffix('.stderr').stat().st_size for p in paths)
    journal = [json.loads(line) for line in (run / 'timing/rounds.jsonl').read_text().splitlines()]
    assert len(journal) == count
    assert len(list((run / 'timing').glob('round-*-baseline.jsonl'))) == count
    ratios = []
    for i, row in enumerate(journal):
        order = ['baseline', 'candidate'] if i % 2 == 0 else ['candidate', 'baseline']
        assert row['index'] == i and row['order'] == order
        rates = {k: nps(run / f'timing/round-{i:02d}-{k}.jsonl') for k in order}
        for k in order:
            close(rates[k], row['nps'][k])
        ratio = rates['candidate'] / rates['baseline']
        close(ratio, row['ratio'])
        ratios.append(ratio)
    stats = interval(ratios, t)
    for k, v in stats.items():
        if isinstance(v, list):
            for a, b in zip(v, receipt[k]):
                close(a, b)
        else:
            close(v, receipt[k])
    assert stats['ci95'][0] > 1 and all(x > 1 for x in stats['order_strata_ratios'])
    assert receipt['repeatable_gain']
    for kind in ['baseline', 'candidate']:
        nps(run / f'timing/warmup-{kind}.jsonl')
    assert receipt['cpu'] == 14 and receipt['environment'] == {'GOMAXPROCS': '1'}
    assert receipt['nice'] == 19
    expected_bins = load(run / 'speed-manifest.json')['binary_sha256'] if count == 20 else build['binary_sha256']
    assert receipt['binary_sha256'] == expected_bins
    for log in ['full-short.log', 'full-race.log']:
        contents = (run / log).read_text()
        assert 'github.com/ehrlich-b/ngn/engine' in contents and 'FAIL' not in contents
    assert not (run / 'vet.log').stat().st_size
    review = dict(run=name, blocks=count, exact_identities=12*len(modes),
                  source_sha256=build['source_sha256'], binary_sha256=expected_bins,
                  terminal=(run / f'{terminal}.done').read_text().strip(),
                  full_short_race_vet=True, input_pins=True, raw_nps_reconciled=True,
                  **stats, verdict='pure-speed-keep', no_elo_claim=True)
    if count == 20:
        cost = load(run / 'architecture-cost.json')
        assert len(cost['rounds']) == 12
        ratios = []
        for i, row in enumerate(cost['rounds']):
            order = ['ngnn3', 'ngnn4'] if i % 2 == 0 else ['ngnn4', 'ngnn3']
            assert row['index'] == i and row['order'] == order
            rates = {k: nps(run / f'timing/cost-{i:02d}-{k}.jsonl') for k in order}
            for k in order:
                close(rates[k], row['nps'][k])
            ratios.append(rates['ngnn4']/rates['ngnn3'])
            close(ratios[-1], row['ratio'])
        budget = interval(ratios, 2.200985160082949)
        close(budget['geometric_ratio'], cost['geometric_ratio'])
        assert budget['geometric_ratio'] >= .85 and cost['budget_admitted']
        review['architecture_budget'] = dict(blocks=12, **budget, threshold=.85,
                                             admitted=True, limitation=cost['limitation'])
        review['adoption'] = 'architecture topic only; no training before data/factorizer sequence'
    else:
        assert 'copied a null accumulator' in (run / 'baseline-witness.log').read_text()
        for tag in ['avx2', 'purego']:
            contents = (run / f'differential-{tag}.log').read_text()
            assert '--- PASS: TestNGNNNullFrameOwnership' in contents and 'FAIL' not in contents
        review['ownership_old_fail_new_pass'] = True
        review['base_source'] = 'f8686be'
        review['adoption'] = 'exact tested playing-base patch'
    target = RESULTS / ('vnni512-review.json' if count == 20 else 'null-frame-review.json')
    target.write_text(json.dumps(review, indent=2)+'\n')
    print(json.dumps(review))
