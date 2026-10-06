"""Train three owned net1 refinements and run sequential fixed WSL gates."""

import argparse
import collections
import datetime
import glob
import hashlib
import importlib.util
import json
import math
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import sys
import time

import numpy as np
import torch

from trainer.ngnp import RECORD_DTYPE, Shards, decode_record, to_fen

DATA_ROOT = Path('/home/ehrli/ngn-data')
BASE = DATA_ROOT / 'nets/net1/net1.nnue'
BASE_SHA = '6a203ffe128541ef422f9344f2906bffbebea6d93931bb1f50e27f44bf2beff6'
BOOK = DATA_ROOT / 'net1-gates-20261006/openings.txt'
BOOK_SHA = '974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'
PLAN = [('c1', .75, 80, .01), ('c2', 1., 32, .1), ('c3', .9, 32, .1)]


def sha(path):
    digest = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def write(path, value):
    temporary = path.with_suffix('.json.tmp')
    temporary.write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')
    temporary.replace(path)


def event(kind, **fields):
    print(json.dumps(dict(event=kind, **fields), allow_nan=False), flush=True)


def disk(stage, root):
    raw = subprocess.check_output(['powershell.exe', '-NoProfile', '-Command',
                                   '(Get-PSDrive C).Free/1GB'], text=True)
    free = float(raw.strip())
    write(root / 'disk.json', dict(stage=stage, free_gib=free))
    if free < .25:
        raise RuntimeError(f'NEEDS_DISK: {stage}, C: free {free:.6f} GiB')
    return free


def parse_gate(log):
    text = log.read_text()
    result = re.search(r'Games: (\d+)\s+W-D-L: (\d+)-(\d+)-(\d+)', text)
    pairs = re.search(r'Pentanomial \[LL (\d+)\s+LD (\d+)\s+\{LW,DD\} (\d+)\s+WD (\d+)\s+WW (\d+)\] over (\d+) pairs', text)
    flags = re.search(r'Flag-outs \(lost on time\): new (\d+), base (\d+)', text)
    assert result and pairs and flags, log
    n, w, d, l = map(int, result.groups())
    counts = list(map(int, pairs.groups()[:5]))
    assert n == w + d + l == 400 and sum(counts) == int(pairs[6]) == 200
    assert sum(k * c for k, c in enumerate(counts)) == 2 * w + d
    previous, reasons = [0, 0, 0], collections.Counter()
    for game, cw, cd, cl, reason in re.findall(r'^G(\d+)\s+(\d+)W\s+(\d+)D\s+(\d+)L\s+.*?\s+(\S+)\s*$', text, re.M):
        now = list(map(int, (cw, cd, cl)))
        assert sum(now) == int(game) == sum(previous) + 1
        assert sorted(now[i] - previous[i] for i in range(3)) == [0, 0, 1]
        reasons[reason] += 1
        previous = now
    assert previous == [w, d, l] and sum(reasons.values()) == n
    markers = ('WATCHDOG:', 'no-move', 'illegal-move', 'info string error',
               'failed to start', 'no uciok within', 'no readyok within',
               'handshakes but emitted no warmup move', 'panic:', 'fatal error:')
    anomalies = [marker for marker in markers if marker in text]
    anomalies += [reason for reason in reasons if reason not in
                  ('checkmate', 'stalemate', 'draw-rule', 'max-moves')]
    mean = (w + d / 2) / n
    variance = max(.001, sum(c * (k / 4) ** 2 for k, c in enumerate(counts)) / 200 - mean ** 2)
    se = math.sqrt(variance / 200)

    def elo(p):
        # Match the owned runner's display bounds at degenerate tails.
        return -800. if p <= 1e-4 else 800. if p >= 1 - 1e-4 else 400 * math.log10(p / (1 - p))

    estimate, ci = elo(mean), [elo(mean - 1.96 * se), elo(mean + 1.96 * se)]
    printed = re.search(r'Penta Elo: ([+-]?[\d.]+)\s+95% CI \[([+-]?\d+), ([+-]?\d+)\]', text)
    assert printed and abs(float(printed[1]) - estimate) <= .051
    assert all(abs(float(printed[i + 2]) - ci[i]) <= .501 for i in (0, 1))
    return dict(wdl=[w, d, l], penta=counts, elo=estimate, elo_95_ci=ci,
                flags=list(map(int, flags.groups())), anomalies=anomalies,
                reasons=dict(reasons), log_sha256=sha(log))


def run(command, log, deadline, root, stage, env):
    remaining = deadline - time.monotonic()
    if remaining <= 0:
        raise RuntimeError('Experiment wall budget exhausted before ' + stage)
    begin = time.monotonic()
    with log.open('w') as output:
        process = subprocess.Popen(command, stdout=output, stderr=subprocess.STDOUT,
                                   start_new_session=True, env=env)
        write(root / 'current-job.json', dict(stage=stage, pid=process.pid, command=command))
        try:
            code = process.wait(timeout=remaining)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGTERM)
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
            raise RuntimeError('Experiment wall budget exhausted during ' + stage)
    if code:
        raise RuntimeError(f'{stage} exited {code}; see {log}')
    return time.monotonic() - begin


def main():
    cli = argparse.ArgumentParser(description=__doc__)
    cli.add_argument('--root', type=Path, required=True)
    cli.add_argument('--budget-seconds', type=float, default=7200)
    args = cli.parse_args()
    if args.budget_seconds <= 0:
        cli.error('Budget must be positive')
    root = args.root.resolve()
    root.mkdir(exist_ok=False)
    deadline = time.monotonic() + args.budget_seconds
    env = dict(os.environ, GOAMD64='v3', GOMAXPROCS='2', GOTOOLCHAIN='local',
               GOPROXY='off', OMP_NUM_THREADS='2', MKL_NUM_THREADS='2',
               OPENBLAS_NUM_THREADS='2', PYTHONDONTWRITEBYTECODE='1')
    os.environ.update(env)
    torch.set_num_threads(2)
    assert sha(BASE) == BASE_SHA and sha(BOOK) == BOOK_SHA
    openings = [line.strip() for line in BOOK.read_text().splitlines()
                if line.strip() and not line.strip().startswith('#')]
    assert len(set(openings[:200])) == 200
    metadata = dict(commit=subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip(),
                    started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                    baseline=str(BASE), baseline_sha256=BASE_SHA, openings=str(BOOK),
                    opening_sha256=BOOK_SHA, tc='10+0.1', concurrency=8)
    old = json.loads((DATA_ROOT / 'trials-20261006/snapshot.json').read_text())
    holdout = old['holdout']
    paths = sorted(glob.glob(str(DATA_ROOT / 'gen0/data/*/*.ngnp')) +
                   glob.glob(str(DATA_ROOT / 'gen0b/data/*/*.ngnp')))
    assert set(holdout) <= set(paths) and len(holdout) == 2
    assert all(Path(path).stat().st_size % 32 == 0 for path in paths)
    snapshot = [dict(path=p, bytes=Path(p).stat().st_size, sha256=sha(p),
                     split='validation' if p in holdout else 'train') for p in paths]
    write(root / 'snapshot.json', dict(holdout=holdout, shards=snapshot))
    data = [p for p in paths if p not in holdout]
    counts = {p: Path(p).stat().st_size // 32 for p in data}
    write(root / 'record-counts.json', counts)
    validation = Shards(holdout, require_kings=True)
    targets = np.linspace(0, validation.count - 1, 20, dtype=np.int64)
    selected, seen = [], 0
    for raw in validation.batches(16384, 0, shuffle=False):
        records = raw.reshape(-1).view(RECORD_DTYPE)
        for target in targets[(targets >= seen) & (targets < seen + len(raw))]:
            selected.append(records[int(target - seen)].copy())
        seen += len(raw)
    fens = [to_fen(decode_record(record)) for record in selected]
    assert len(fens) == 20
    assert fens == json.loads((DATA_ROOT / 'trials-20261006/parity-positions.json').read_text())['fens']
    write(root / 'parity-positions.json', dict(holdout=holdout, fens=fens, filtered_indices=targets.tolist()))
    spec = importlib.util.spec_from_file_location('owned_trials', Path(__file__).with_name('2026-10-06-net1-recipe-trials.py'))
    owned = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(owned)
    disk('build', root)
    for name, package in [('ngn', '.'), ('sprt', './cmd/sprt')]:
        run(['/usr/local/go/bin/go', 'build', '-p', '2', '-o', str(root / name), package],
            root / (name + '-build.log'), deadline, root, 'build-' + name, env)
        metadata[name + '_sha256'] = sha(root / name)
    run(['/usr/local/go/bin/go', 'test', '-p', '2', './cmd/sprt', './internal/uci'],
        root / 'go-checks.log', deadline, root, 'runner-uci-checks', env)
    write(root / 'metadata.json', metadata)
    trials, gates = [], []
    for name, lam, epochs, minimum in PLAN:
        free = disk('train-' + name, root)
        event('trial_started', name=name, free_gib=free)
        out = root / name
        out.mkdir()
        command = [sys.executable, '-m', 'trainer.train', '--data', *data,
                   '--validation-data', *holdout, '--record-counts', str(root / 'record-counts.json'),
                   '--out', str(out), '--format', '2', '--hidden', '256', '--buckets', '8',
                   '--batch', '16384', '--lr', '.001', '--epochs', str(epochs), '--wdl', str(lam),
                   '--decay', 'cosine', '--min-lr-ratio', str(minimum), '--threads', '2',
                   '--precision', 'bf16', '--seed', '20261006', '--keep-best-only']
        wall = run(command, out / 'train.log', deadline, root, 'train-' + name, env)
        history = json.loads((out / 'history.json').read_text())
        assert len(history) == epochs
        best = min(history, key=lambda row: row['validation_loss'])
        receipt = dict(name=name, lambda_=lam, epochs=epochs, wall_seconds=wall,
                       best=best, **owned.parity(out, selected))
        for filename in ('latest.pt', 'latest.nnue'):
            (out / filename).unlink()
        trials.append(receipt)
        write(root / 'training-results.json', dict(metadata=metadata, trials=trials))
        event('trial_completed', name=name, best_epoch=receipt['best_epoch'], wall_seconds=wall)
        if sum(p.stat().st_size for p in root.rglob('*') if p.is_file()) > 50_000_000:
            raise RuntimeError('Run outputs exceed 50 MB disk allocation')
    for trial in trials:
        name = trial['name']
        free = disk('gate-' + name, root)
        event('gate_started', name=name, free_gib=free)
        net = root / name / 'best.nnue'
        assert sha(net) == trial['network_sha256']
        command = [str(root / 'sprt'), '-new', str(root / 'ngn'), '-base', str(root / 'ngn'),
                   '-tc', '10+0.1', '-concurrency', '8', '-maxgames', '400', '-mingames', '1000000',
                   '-maxmoves', '400', '-gametimeout', '180', '-openings', str(BOOK), '-lowpower=false']
        for side, network in [('new', net), ('base', BASE)]:
            for option in ['EvalFile=' + str(network), 'UseNNUE=true', 'Hash=64', 'Threads=1', 'OwnBook=false']:
                command += ['-' + side + 'option', option]
        log = root / (name + '-gate.log')
        wall = run(command, log, deadline, root, 'gate-' + name, dict(env, GOMAXPROCS='1'))
        result = dict(name=name, wall_seconds=wall, command=command, **parse_gate(log))
        gates.append(result)
        write(root / 'gate-results.json', dict(metadata=metadata, gates=gates))
        event('gate_completed', **{k: result[k] for k in ('name', 'wdl', 'elo', 'elo_95_ci', 'flags', 'anomalies')})
        if result['anomalies'] or any(result['flags']):
            raise RuntimeError('Gate anomaly: ' + name)
    for row in snapshot:
        assert Path(row['path']).stat().st_size == row['bytes'] and sha(row['path']) == row['sha256']
    assert sha(BASE) == BASE_SHA and sha(BOOK) == BOOK_SHA
    assert all(sha(root / binary) == metadata[binary + '_sha256'] for binary in ('ngn', 'sprt'))
    qualifying = [r for r in gates if r['elo_95_ci'][0] > 0 or r['elo'] >= 10]
    winner = max(qualifying, key=lambda r: r['elo']) if qualifying else None
    selection = dict(winner=winner['name'] if winner else None,
                     rule='paired CI lower > 0 or Elo >= +10, zero anomalies; highest qualifying Elo',
                     sha256=BASE_SHA, path=str(BASE))
    if winner:
        destination = DATA_ROOT / 'nets/net1b'
        destination.mkdir(exist_ok=False)
        source = root / winner['name'] / 'best.nnue'
        target = destination / 'net1b.nnue'
        shutil.copyfile(source, target)
        selection.update(path=str(target), sha256=sha(target))
        assert selection['sha256'] == next(t['network_sha256'] for t in trials if t['name'] == winner['name'])
        (destination / 'SHA256SUMS').write_text(selection['sha256'] + '  net1b.nnue\n')
        write(destination / 'selection.json', selection)
    selection['free_gib_at_end'] = disk('completed', root)
    selection['data_hashes_unchanged'] = True
    write(root / 'selection.json', selection)
    write(root / 'completed.json', dict(selection=selection, trials=3, gates=3))
    event('completed', **selection)


if __name__ == '__main__':
    main()
