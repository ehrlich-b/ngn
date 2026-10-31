"""Recount the terminal Rodent match from raw pairs and independently replay moves."""
import argparse
from collections import Counter
import hashlib
import json
import math
from pathlib import Path

import chess


def elo(score):
    if score <= .0001:
        return -800.
    if score >= .9999:
        return 800.
    return 400 * math.log10(score / (1 - score))


def replay(game, opening):
    fen = '/' in opening
    board = chess.Board(opening) if fen else chess.Board()
    moves = game['moves'] or []
    if not fen:
        assert moves[:len(opening.split())] == opening.split(), 'opening move mismatch'
    for token in moves:
        move = chess.Move.from_uci(token)
        assert move in board.legal_moves, f'illegal recorded move {token}'
        board.push(move)
    result = game['result']
    outcome, reason = result['Res'], result['Reason']
    assert outcome in (0, 1, 2)
    if reason == 'checkmate':
        assert board.is_checkmate()
        assert outcome == (0 if board.turn == game['ngn_white'] else 2)
    elif reason == 'stalemate':
        assert board.is_stalemate() and outcome == 1
    elif reason == 'draw-rule':
        assert outcome == 1 and (board.halfmove_clock >= 100 or board.is_repetition(3))
        assert not board.is_checkmate() and not board.is_stalemate()
    elif reason == 'max-moves':
        assert outcome == 1 and len(moves) == 400
        assert not board.is_checkmate() and not board.is_stalemate()
        assert board.halfmove_clock < 100 and not board.is_repetition(3)
    elif reason in ('time-forfeit', 'no-move', 'illegal-move'):
        assert board.legal_moves.count() > 0
        assert outcome == (0 if board.turn == game['ngn_white'] else 2)
    else:
        raise AssertionError(f'unexpected termination reason {reason}')


def recount(rows, openings):
    ids, complete, excluded = set(), [], []
    reasons, flags, failures = Counter(), Counter(), Counter()
    for row in rows:
        pair_id = row['pair_id']
        if pair_id == -1:
            assert row.get('error') and not row.get('games')
            failures['startup'] += 1
            excluded.append(row)
            continue
        assert pair_id not in ids and 0 <= pair_id < len(openings)
        ids.add(pair_id)
        assert row['opening'] == openings[pair_id], 'pair opening/index mismatch'
        games = row.get('games') or []
        assert len(games) <= 2
        for index, game in enumerate(games):
            assert game['ngn_white'] is (index == 0), 'colors not reversed'
            replay(game, row['opening'])
            reason = game['result']['Reason']
            reasons[reason] += 1
            if reason == 'time-forfeit':
                flags['ngn' if game['result']['Res'] == 0 else 'opponent'] += 1
            elif reason in ('no-move', 'illegal-move'):
                failures[reason] += 1
        if row.get('error') or len(games) != 2:
            failures['incomplete_pair'] += 1
            excluded.append(row)
        else:
            complete.append(row)
    assert not ids or ids == set(range(max(ids) + 1)), 'missing admitted pair row'
    wdl, penta = [0, 0, 0], [0] * 5
    for row in complete:
        outcomes = [g['result']['Res'] for g in row['games']]
        for outcome in outcomes:
            wdl[2 - outcome] += 1
        penta[sum(outcomes)] += 1
    n = len(complete)
    score = sum(i * count for i, count in enumerate(penta)) / (4 * n) if n else None
    estimates = None
    if n:
        assert abs(score - (wdl[0] + .5 * wdl[1]) / (2 * n)) < 1e-12
        variance = max(sum((i / 4) ** 2 * count for i, count in enumerate(penta)) / n - score**2, .001)
        se = math.sqrt(variance / n)
        estimates = [elo(score), elo(score - 1.96 * se), elo(score + 1.96 * se)]
    return dict(games=2*n, pairs=n, wins=wdl[0], draws=wdl[1], losses=wdl[2],
                score=score, penta=penta, exact_elo_ci95=estimates,
                flag_outs=dict(ngn=flags['ngn'], opponent=flags['opponent']),
                failures=dict(failures), termination_reasons_all_recorded_games=dict(reasons),
                excluded_pairs=len(excluded),
                excluded_recorded_games=sum(len(p.get('games') or []) for p in excluded),
                replay_pass=True, minimum160_met=2*n >= 160)


def report(root):
    assert (root / 'controller.done').is_file(), 'match has not terminated'
    rows = [json.loads(line) for line in (root / 'pairs.jsonl').read_text().splitlines()]
    openings = [' '.join(line.split()) for line in (root / 'openings.txt').read_text().splitlines()
                if line.strip() and not line.lstrip().startswith('#')]
    result = recount(rows, openings)
    summary_path = root / 'summary.json'
    summary = json.loads(summary_path.read_text()) if summary_path.read_text().strip() else None
    if summary is not None:
        for key in ('games', 'pairs', 'wins', 'draws', 'losses', 'penta', 'minimum160_met'):
            assert summary[key] == result[key], f'summary differs: {key}'
        if result['games']:
            for key, value in zip(('elo', 'elo_lower95', 'elo_upper95'), result['exact_elo_ci95']):
                assert math.isclose(summary[key], value, rel_tol=1e-12, abs_tol=1e-9), key
            assert math.isclose(summary['score'], result['score'], abs_tol=1e-12)
    checks = {}
    for suffix in ('before', 'after'):
        p = root / f'input-check-{suffix}.txt'
        lines = p.read_text().splitlines() if p.exists() else []
        checks[suffix] = bool(lines) and all(line.endswith(': OK') for line in lines)
    crash_path = root / 'ngn_crashes.log'
    crash_text = crash_path.read_text() if crash_path.exists() else ''
    recovered_panics = crash_text.count('PANIC RECOVERED - CRASH ID #')
    result.update(controller_exit=int((root / 'controller.exit').read_text()),
                  input_checks=checks, summary_reconciled=summary is not None,
                  raw_pairs_sha256=hashlib.sha256((root / 'pairs.jsonl').read_bytes()).hexdigest(),
                  opponent_label=3532,
                  implied_performance_ci95=([3532+x for x in result['exact_elo_ci95']]
                                            if result['exact_elo_ci95'] else None),
                  ci_method='pair score moments; variance floor .001; normal 1.96 interval transformed to logistic Elo',
                  limitation='one opponent at one time control; not a rating or campaign claim; label uncertainty not included',
                  observed_ngn_recovered_panics=recovered_panics,
                  ngn_crash_log_present=crash_path.exists(),
                  crash_evidence='NGN recovered panics counted from its crash log; no-move/startup failures recorded; opponent exit status/stderr unavailable')
    if not result['minimum160_met']:
        result['conclusion'] = 'minimum not met; report completed pairs and shortfall only'
    elif result['failures'] or any(result['flag_outs'].values()) or recovered_panics or not all(checks.values()) or result['controller_exit']:
        result['conclusion'] = 'operational issue; report completed pairs and faults, no clean strength conclusion'
    elif result['exact_elo_ci95'][1] > 0:
        result['conclusion'] = 'NGN beats Rodent in this match with positive lower95'
    elif result['exact_elo_ci95'][2] < 0:
        result['conclusion'] = 'Rodent beats NGN in this match with negative upper95'
    else:
        result['conclusion'] = 'interval spans zero; this match does not establish which engine is stronger'
    return result


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('root', type=Path)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    result = report(args.root)
    args.out.write_text(json.dumps(result, indent=2, allow_nan=False) + '\n')
    print(json.dumps(result, allow_nan=False))
