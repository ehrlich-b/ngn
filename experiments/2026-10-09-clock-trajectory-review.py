#!/usr/bin/env python3
"""Summarize the fixed clock diagnostic without treating it as an Elo gate."""
import collections
import hashlib
import json
from pathlib import Path


def review(path):
    rows = [json.loads(line.split('CLOCKTRAJECTORY ', 1)[1])
            for line in path.read_text().splitlines() if line.startswith('CLOCKTRAJECTORY ')]
    moves = [row for row in rows if row['kind'] == 'move']
    ends = [row for row in rows if row['kind'] == 'game']
    assert len(ends) == 1 and ends[0]['plies'] == len(moves), path
    assert moves and all(row['alive'] for row in moves), path
    assert [row['ply'] for row in moves] == list(range(len(moves))), path
    banks = {0: 120_000_000_000, 1: 120_000_000_000}
    for row in moves:
        side = row['side']
        assert row['bank_before_ns'] == banks[side], (path, row['ply'])
        assert 0 <= row['spent_ns'] < banks[side], (path, row['ply'])
        banks[side] = banks[side] - row['spent_ns'] + 1_000_000_000
        assert row['bank_after_ns'] == banks[side], (path, row['ply'])
        assert 0 <= row['tail_ns'] <= row['stop_elapsed_ns'] <= row['spent_ns'], (path, row['ply'])
    # NGN's Color order is Black=0, White=1.
    assert ends[0]['black_bank_ns'] == banks[0] and ends[0]['white_bank_ns'] == banks[1], path
    checkpoints = {}
    for side in (0, 1):
        side_moves = [row for row in moves if row['side'] == side]
        checkpoints['black' if side == 0 else 'white'] = {
            str(n): side_moves[n-1]['bank_after_ns'] / 1e9
            for n in (10, 20, 30, 40, 60, 80) if len(side_moves) >= n
        }
    elapsed = sum(row['stop_elapsed_ns'] for row in moves)
    tail = sum(row['tail_ns'] for row in moves)
    return dict(variant=ends[0]['variant'], case=ends[0]['case'], plies=len(moves),
                end_reason=ends[0]['reason'], reasons=dict(collections.Counter(row['reason'] for row in moves)),
                elapsed_ns=elapsed, tail_ns=tail, tail_fraction=tail / elapsed,
                bank_checkpoints_seconds=checkpoints,
                minimum_bank_before_seconds=min(row['bank_before_ns'] for row in moves)/1e9,
                score_drops_ge100=sum(b['score'] <= a['score']-100
                                     for side in (0, 1)
                                     for a, b in zip([r for r in moves if r['side'] == side],
                                                     [r for r in moves if r['side'] == side][1:])),
                log_sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def main():
    games = [review(Path(f'trajectory-{kind}-{case}.log'))
             for kind, case in [('baseline', 0), ('candidate', 0), ('candidate', 1), ('baseline', 1)]]
    variants = {}
    for kind in ('baseline', 'candidate'):
        subset = [g for g in games if g['variant'] == kind]
        elapsed = sum(g['elapsed_ns'] for g in subset)
        variants[kind] = dict(plies=sum(g['plies'] for g in subset),
                              tail_fraction=sum(g['tail_ns'] for g in subset)/elapsed)
    result = dict(complete=True, diagnostic_only=True, strength_verdict=None, games=games, variants=variants,
                  limitation='Four divergent self-play trajectories on39eb128, each ending naturally or capped at160 plies. Checkpoints count searched moves from the fixture. No rating, significance claim or allocator selection. Rebase and revalidate before strength games if the accepted source changes.')
    Path('trajectory-summary.json').write_text(json.dumps(result, indent=2)+'\n')
    print(json.dumps(result, indent=2))


if __name__ == '__main__':
    main()
