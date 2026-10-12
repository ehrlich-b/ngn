"""Reconcile clock traces for diagnosis only; emit no strength verdict."""
import json
from pathlib import Path

p = Path.cwd()
rows = [json.loads(line) for line in (p / 'pairs.jsonl').read_text().splitlines()]
flags, errors, margins = [], [], []
games = samples = 0
for row in rows:
    if row.get('error'):
        errors.append({'pair_id': row['pair_id'], 'error': row['error']})
    for game in row['games']:
        games += 1
        banks = [10000, 10000]
        for clock in game['clocks']:
            samples += 1
            side = 0 if clock['white'] else 1
            assert [clock['white_before_ms'], clock['black_before_ms']] == banks
            assert clock['go'] == f'go wtime {banks[0]} btime {banks[1]} winc 100 binc 100'
            margin = banks[side] - clock['elapsed_ms']
            margins.append(margin)
            assert clock['flagged'] == (margin < -300)
            increment = 0 if clock['flagged'] else 100
            assert clock['increment_applied_ms'] == increment
            banks[side] = margin + increment
            assert [clock['white_after_ms'], clock['black_after_ms']] == banks
            if clock['flagged']:
                flags.append(dict(pair_id=row['pair_id'], ngn_white=game['ngn_white'],
                    position='position startpos moves ' + ' '.join(game['moves'][:clock['ply'] - 1]),
                    clock=clock))
summary = json.loads((p / 'summary.json').read_text())
result = dict(target_games=24, recorded_games=games, recorded_pairs=len(rows),
              clock_samples=samples, command_and_bank_conservation=True,
              minimum_bank_margin_ms=min(margins) if margins else None,
              flags=flags, runner_errors=errors, runner_complete=summary['complete'],
              diagnostic_exit=int((p / 'diagnostic.exit').read_text()),
              worker_receipts=summary['worker_receipts'],
              verdict='DIAGNOSTIC ONLY',
              limitation='No Elo, no replayed SEE control, no automatic reopening of forge; absence of a reproduced fault does not explain the original flag.')
(p / 'clock-review.json').write_text(json.dumps(result, indent=2) + '\n')
print(json.dumps({k: v for k, v in result.items() if k not in ('flags', 'worker_receipts')}, indent=2))
