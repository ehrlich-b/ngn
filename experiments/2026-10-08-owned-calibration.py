"""Fixed owned-data diagnostic; real generation requires a reviewed immediate base."""
import argparse
import copy
import datetime
import hashlib
import importlib.util
import json
import math
import os
from pathlib import Path
import platform
import subprocess
import tempfile
import time

CAMPAIGN = Path('/home/ehrli/ngn-data/night-20261007')
ROOT = CAMPAIGN / 'session10'
BUDGETS = (5000, 25000)
SEEDS = (2026100800, 2026100801)
CPUS = (13, 15)
GAMES = 64
CELL_TIMEOUT = 1800


def digest(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(8 * 1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def load(path):
    return json.loads(Path(path).read_text())


def write(path, value):
    with Path(path).open('x') as stream:
        stream.write(json.dumps(value, indent=2, allow_nan=False) + '\n')


def event(name, detail):
    with (CAMPAIGN / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='owned-data-calibration', event=name, detail=detail)) + '\n')


def settings(nodes, seed, net):
    return dict(net=net, seed=seed, games=GAMES, nodes=nodes, random_plies=-1,
        maxply=400, hash_mb=16, verification_nodes=1000, opening_limit_cp=400,
        win_cp=2000, win_plies=6, draw_cp=8, draw_plies=12, draw_min_ply=80)


def check_meta(meta, expected, binary, commit, net, data):
    assert meta['format'] == 'NGNP1' and meta['closed'] is True
    assert meta['status'] == 'complete' and meta['settings'] == expected
    assert meta['seed'] == expected['seed']
    assert meta['games'] == GAMES == sum(meta['game_results']) == sum(meta['terminations'].values())
    assert set(meta['terminations']) <= {'checkmate', 'stalemate', 'fifty_move', 'repetition',
        'insufficient_material', 'maxply', 'score_win', 'score_draw'}
    assert all(isinstance(x, int) and x >= 0 for x in meta['game_results'])
    assert all(isinstance(x, int) and x >= 0 for x in meta['terminations'].values())
    assert meta['attempts'] == GAMES + meta['rejected_openings']
    assert meta['dropped_games'] == meta['unscored_searches'] == 0
    assert GAMES * 8 <= meta['game_plies'] <= GAMES * 400
    assert 0 < meta['positions'] <= meta['game_plies']
    assert meta['positions'] * 32 == meta['bytes'] == data.stat().st_size
    assert meta['sha256'] == digest(data)
    assert meta['engine_commit'] == commit and meta['binary_sha256'] == binary
    assert meta['net_sha256'] == net
    assert 0 < meta['search_nodes'] <= meta['searches'] * expected['nodes']
    assert 0 <= meta['early_searches'] <= meta['searches']
    assert math.isfinite(meta['elapsed_seconds']) and meta['elapsed_seconds'] > 0
    started = datetime.datetime.fromisoformat(meta['started_at'].replace('Z', '+00:00'))
    closed = datetime.datetime.fromisoformat(meta['closed_at'].replace('Z', '+00:00'))
    assert closed > started


def summarize(paths, counts=None):
    """Descriptive record-weighted calibration, with no independence/strength claim."""
    import numpy as np
    from trainer.ngnp import RECORD_DTYPE, decode_records
    from trainer.king_buckets import DEFAULT_MAP
    total, score_sum, score_squared, brier_sum = 0, 0., 0., 0.
    outcomes, stm, buckets = np.zeros(3, np.int64), np.zeros(2, np.int64), np.zeros((2, 8), np.int64)
    bins = {str(edge): [0, 0., 0.] for edge in (-2000, -400, -100, 100, 400, 2000, 32768)}
    for path in paths:
        count = path.stat().st_size // 32 if counts is None else counts[str(path)]
        assert count > 0 and path.stat().st_size >= count * 32
        records = np.memmap(path, mode='r', dtype=RECORD_DTYPE, shape=(count,))
        for offset in range(0, count, 16384):
            rows = records[offset:offset + 16384]
            assert np.all(rows['flags'] == 0) and np.all(rows['reserved'] == 0)
            decoded = decode_records(rows)
            board = decoded['board']
            for perspective in (0, 1):
                own_king = board == 5 + 6 * perspective
                assert np.all(own_king.sum(axis=1) == 1)
                relative = own_king.argmax(axis=1) ^ (56 * perspective)
                relative ^= np.where(relative % 8 >= 4, 7, 0)
                ids = np.asarray(DEFAULT_MAP)[relative]
                buckets[perspective] += np.bincount(ids, minlength=8)
            scores = rows['score'].astype(np.float64)
            result = rows['result'].astype(np.float64) / 2
            probability = 1 / (1 + np.exp(-scores / 400.))
            total += len(rows)
            score_sum += scores.sum()
            score_squared += (scores ** 2).sum()
            brier_sum += ((probability - result) ** 2).sum()
            outcomes += np.bincount(rows['result'], minlength=3)
            stm += np.bincount(rows['stm'], minlength=2)
            previous = -32769
            for edge, values in bins.items():
                mask = (scores >= previous) & (scores < int(edge))
                values[0] += int(mask.sum())
                values[1] += float(probability[mask].sum())
                values[2] += float(result[mask].sum())
                previous = int(edge)
    assert total == sum(outcomes) == sum(stm) == sum(v[0] for v in bins.values())
    assert all(sum(row) == total for row in buckets)
    return dict(records=total, result_white_loss_draw_win=outcomes.tolist(), stm_white_black=stm.tolist(),
        mean_score_cp=score_sum / total, rms_score_cp=math.sqrt(score_squared / total),
        score_result_brier_k400=brier_sum / total, king_bucket_counts_white_black=buckets.tolist(),
        score_bin_upper_exclusive={edge: dict(records=v[0], mean_prediction=v[1] / v[0] if v[0] else None,
            mean_result=v[2] / v[0] if v[0] else None) for edge, v in bins.items()},
        interpretation='record-weighted descriptive sample; correlated positions, no CI or Elo')


def baseline(root):
    old = load(root / 'baseline-input.json')
    assert old['shards'] == 14 and old['prefix_records_per_shard'] == 65536
    paths = list(map(Path, sorted(old['prefix_sha256'])))
    counts = {str(p): old['prefix_records_per_shard'] for p in paths}
    before = {str(p): hashlib.sha256(p.open('rb').read(counts[str(p)] * 32)).hexdigest() for p in paths}
    assert before == old['prefix_sha256']
    result = summarize(paths, counts)
    after = {str(p): hashlib.sha256(p.open('rb').read(counts[str(p)] * 32)).hexdigest() for p in paths}
    assert before == after
    write(root / 'baseline-receipt.json', dict(sample=result, prefix_sha256=before,
        prefix_records_per_shard=old['prefix_records_per_shard'], unchanged_prefixes=True,
        scope='same fixed closed gen0b prefixes as session7; historical HCE teacher, not the new teacher'))


def plan(root):
    spec = importlib.util.spec_from_file_location('reviewed_base', root / 'base-review.py')
    reviewer = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(reviewer)
    reviewed = reviewer.plan(root)
    build = load(root / 'calibration-build.json')
    assert build['source_commit'] == reviewed['source_commit']
    assert build['source_reconciled'] is True and build['tests_passed'] is True
    for name, sha in build['binary_sha256'].items():
        assert digest(root / 'frozen' / name) == sha
    assert (root / 'activation-commit.txt').read_text().strip() == load(root / 'nomination.json')['review_commit']
    cells = []
    for nodes in BUDGETS:
        folder = root / f'nodes-{nodes}'
        commands = []
        for cpu, seed in zip(CPUS, SEEDS):
            commands.append(['taskset', '-c', str(cpu), 'nice', '-n', '19', str(root / 'frozen/datagen'),
                '-out', str(folder / f'seed-{seed}.ngnp'), '-net', reviewed['base_net'],
                '-seed', str(seed), '-games', str(GAMES), '-nodes', str(nodes),
                '-random-plies', '-1', '-maxply', '400', '-hash', '16'])
        cells.append(dict(nodes=nodes, commands=commands, folder=str(folder)))
    return dict(base=reviewed['base'], net=reviewed['base_net'], net_sha256=reviewed['base_net_sha256'],
        source_commit=build['source_commit'], binary_sha256=build['binary_sha256'], cells=cells,
        games_per_worker=GAMES, games_total=2 * len(SEEDS) * GAMES,
        cell_timeout_seconds=CELL_TIMEOUT, evidence_sha256=reviewed['evidence_sha256'],
        nomination_sha256=reviewed['nomination_sha256'], interpretation='data calibration only; no strength verdict')


def guards():
    assert platform.system() == 'Linux' and 'microsoft' in platform.release().lower()
    # Only immutable bounded output is written; no assistant polling is needed.
    import shutil
    assert shutil.disk_usage('/mnt/c').free >= 25 * 1024 ** 3
    for process in Path('/proc').glob('[0-9]*/cmdline'):
        try:
            argv = process.read_bytes().decode().split('\0')
        except (OSError, UnicodeError):
            continue
        assert 'trainer.train' not in argv, 'trainer owns its window'
        assert not any(Path(a).name in ('datagen', 'sprt', 'claimtool', 'engine.test', 'benchprobe')
            for a in argv if a), 'another evidence job owns its window'


def run(root):
    proposal = plan(root)
    guards()
    write(root / 'calibration.started', dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat()))
    write(root / 'launch-plan.json', proposal)
    results = []
    event('start', f'fixed 256 games; CPUs 13,15; teacher {proposal["net_sha256"][:8]}')
    for cell in proposal['cells']:
        guards()
        folder = Path(cell['folder'])
        folder.mkdir()
        children, outputs = [], []
        started = time.monotonic()
        try:
            for argv, seed in zip(cell['commands'], SEEDS):
                output = (folder / f'seed-{seed}.log').open('x')
                outputs.append(output)
                children.append(subprocess.Popen(argv, stdin=subprocess.DEVNULL, stdout=output,
                    stderr=subprocess.STDOUT, env=dict(os.environ, GOMAXPROCS='1')))
            write(folder / 'launch.json', dict(commands=cell['commands'], pids=[p.pid for p in children]))
            deadline = started + CELL_TIMEOUT
            for child in children:
                assert child.wait(timeout=max(.01, deadline - time.monotonic())) == 0
        finally:
            for child in children:
                if child.poll() is None:
                    child.terminate()
            for child in children:
                try:
                    child.wait(timeout=30)
                except subprocess.TimeoutExpired:
                    child.kill()
                    child.wait()
            for output in outputs:
                output.close()
        wall = time.monotonic() - started
        metas, paths = [], []
        for seed in SEEDS:
            data = folder / f'seed-{seed}.ngnp'
            meta = load(str(data) + '.json')
            check_meta(meta, settings(cell['nodes'], seed, proposal['net']),
                proposal['binary_sha256']['datagen'], proposal['source_commit'], proposal['net_sha256'], data)
            metas.append(meta)
            paths.append(data)
        argv = ['taskset', '-c', '13,15', 'nice', '-n', '19', str(root / 'frozen/datastat'),
            '-wall-seconds', str(wall), str(folder)]
        stats = json.loads(subprocess.check_output(argv, timeout=120, text=True))
        assert stats['validated_sidecars'] == stats['files'] == 2
        assert stats['completed_games'] == 128 and stats['flagged_records'] == stats['unscored_searches'] == 0
        assert stats['records'] == sum(m['positions'] for m in metas)
        sample = summarize(paths)
        assert sample['records'] == stats['records']
        write(folder / 'receipt.json', dict(nodes=cell['nodes'], wall_seconds=wall, datastat_argv=argv,
            statistics=stats, sample=sample, sidecars=metas,
            artifacts_sha256={str(p): digest(p) for p in paths + [Path(str(p) + '.json') for p in paths]}))
        results.append(load(folder / 'receipt.json'))
    assert digest(proposal['net']) == proposal['net_sha256']
    for name, sha in proposal['binary_sha256'].items():
        assert digest(root / 'frozen' / name) == sha
    write(root / 'calibration-receipt.json', dict(operationally_valid=True, games=256,
        cells=results, plan=proposal, next_action='review data distribution; declare a separate gen1 recipe before large generation'))
    event('done', 'both fixed cells complete and audited; data calibration supplies no Elo')


def selftest():
    import numpy as np
    from trainer.ngnp import RECORD_DTYPE, encode_record, parse_fen
    rejected = 0
    with tempfile.TemporaryDirectory() as directory:
        path = Path(directory) / 'fixture.ngnp'
        position = parse_fen('4k3/8/8/8/8/8/8/4K3 w - - 0 1')
        rows = np.asarray([encode_record(position, score, result) for score, result in
            ((0, 0), (0, 1), (0, 2), (-400, 0), (400, 2))], dtype=RECORD_DTYPE)
        rows.tofile(path)
        result = summarize([path])
        expected_brier = (.25 + .25 + 2 * (1 / (1 + math.exp(1))) ** 2) / 5
        assert result['records'] == 5 and result['result_white_loss_draw_win'] == [2, 1, 2]
        assert abs(result['score_result_brier_k400'] - expected_brier) < 1e-14
        assert result['king_bucket_counts_white_black'] == [[0, 5, 0, 0, 0, 0, 0, 0]] * 2
        expected = settings(5000, SEEDS[0], '/net')
        meta = dict(format='NGNP1', closed=True, status='complete', settings=expected, seed=SEEDS[0],
            games=64, game_results=[20, 24, 20], terminations={'checkmate': 40, 'score_draw': 24},
            attempts=67, rejected_openings=3, dropped_games=0, unscored_searches=0,
            game_plies=1000, positions=5, bytes=160, sha256=digest(path), engine_commit='a' * 40,
            binary_sha256='b' * 64, net_sha256='c' * 64, search_nodes=100000, searches=100,
            early_searches=0, elapsed_seconds=1., started_at='2026-10-08T00:00:00Z',
            closed_at='2026-10-08T00:00:01Z')
        check = lambda m: check_meta(m, expected, 'b' * 64, 'a' * 40, 'c' * 64, path)
        check(meta)
        changes = dict(status='interrupted', closed=False, net_sha256='d' * 64, binary_sha256='d' * 64,
            engine_commit='d' * 40, games=63, game_results=[20, 23, 20], terminations={'unknown': 64},
            attempts=64, dropped_games=1, unscored_searches=1, bytes=161, positions=6,
            sha256='d' * 64, search_nodes=500001, early_searches=101, elapsed_seconds=float('nan'),
            closed_at='2026-10-07T00:00:01Z', game_plies=25601, seed=1,
            settings=dict(expected, nodes=25000))
        for key, value in changes.items():
            changed = copy.deepcopy(meta)
            changed[key] = value
            try:
                check(changed)
            except AssertionError:
                rejected += 1
            else:
                raise AssertionError(f'failed to reject changed {key}')
        rows['flags'][0] = 1
        rows.tofile(path)
        try:
            summarize([path])
        except AssertionError:
            rejected += 1
        else:
            raise AssertionError('noisy row accepted')
    return dict(passed=True, rejection_controls=rejected, exact_brier_control=True,
        score_bins=True, both_king_perspectives=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('action', choices=('selftest', 'baseline', 'plan', 'run'))
    args = parser.parse_args()
    if args.action == 'selftest':
        print(json.dumps(selftest()))
    elif args.action == 'baseline':
        baseline(ROOT)
    elif args.action == 'plan':
        print(json.dumps(plan(ROOT), indent=2))
    else:
        run(ROOT)


if __name__ == '__main__':
    main()
