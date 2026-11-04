#!/usr/bin/env python3
"""Reconstruct descriptive findings from retained NGN-CLAIM-4 games."""
import argparse
import collections
import json
import math
from pathlib import Path
import statistics

LABELS = {'Koivisto 6.0': 3568, 'akimbo 1.0.0': 3614, 'Black Marlin 9.0': 3623, 'Berserk 12': 3741}

def board_from_fen(fen):
    board = {}
    for rank, row in enumerate(fen.split()[0].split('/')):
        file = 0
        for c in row:
            if c.isdigit():
                file += int(c)
            else:
                board[chr(97 + file) + str(8 - rank)] = c
                file += 1
    return board

def move(board, uci):
    src, dst = uci[:2], uci[2:4]
    piece = board.pop(src)
    # En passant is the only pawn capture onto an empty square.
    if piece.lower() == 'p' and src[0] != dst[0] and dst not in board:
        board.pop(dst[0] + src[1], None)
    if piece.lower() == 'k' and abs(ord(src[0]) - ord(dst[0])) == 2:
        rook_src, rook_dst = (('h' if dst[0] == 'g' else 'a') + src[1], ('f' if dst[0] == 'g' else 'd') + src[1])
        board[rook_dst] = board.pop(rook_src)
    board[dst] = (uci[4].upper() if piece.isupper() else uci[4]) if len(uci) > 4 else piece

def material(board):
    return '/'.join(''.join(p + str(sum(v == p for v in board.values())) for p in pieces) for pieces in ('QRBNP', 'qrbnp'))

def phase(board, ply):
    if ply <= 20:
        return 'opening'
    return 'endgame' if sum(v.lower() in 'qrbn' for v in board.values()) <= 6 else 'middlegame'

def analyze_game(g):
    board = board_from_fen(g['Opening']['FEN'])
    for m in g['Opening']['History']:
        move(board, m)
    offset = len(g['Opening']['History'])
    events = g['Candidate']['Events']
    observations = []
    prev = None
    drop = None
    for i, ply in enumerate(g['Plies'], 1):
        number = offset + i
        ph = phase(board, number)
        if ply['Role'] == 'candidate':
            lines = [e['Line'].split() for e in events[ply['GoEvent'] + 1:ply['BestEvent']] if e['Stream'] == 'stdout' and e['Line'].startswith('info depth ')]
            if lines:
                t = lines[-1]
                def val(k):
                    try: return int(t[t.index(k) + 1])
                    except (ValueError, IndexError): return None
                score = val('cp') if 'score' in t and 'cp' in t else None
                observations.append((ph, val('depth'), val('nodes'), val('nps'), ply['UsedNS'] / 1e6))
                if score is not None:
                    if prev is not None and prev - score >= 100 and drop is None:
                        drop = {'ply': number, 'phase': ph, 'material': material(board), 'previous_cp': prev, 'score_cp': score}
                    prev = score
        move(board, ply['Move'])
    clock = g['Plies'][-1]['After'][0 if g['CandidateColor'] == 'white' else 1] / 1e9
    return observations, drop, clock

def med(values):
    values = [x for x in values if x is not None]
    return statistics.median(values) if values else None

def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--games-root', type=Path, default=Path.home() / 'ngn-data/claim-v4-20261016/attempt-002')
    p.add_argument('--reference', type=Path, default=Path('experiments/2026-10-19-session118-results/screen/analysis.json'))
    p.add_argument('--output', type=Path, required=True)
    args = p.parse_args()
    ref = json.loads(args.reference.read_text())
    assert ref['Schema'] == 'NGN-CLAIM-4'
    report = {'method': 'Final UCI info before bestmove; time is wall UsedNS, clock is candidate clock after final ply. Opening through ply 20; endgame thereafter with at most six non-pawn non-king pieces; otherwise middlegame. First drop compares consecutive candidate centipawn scores; mate scores are omitted. Material is pre-move white/black QRBNP counts. Missing drops are null.', 'modes': {}}
    for mode in ('screen', 'confirmation'):
        groups = collections.defaultdict(list)
        ids = set()
        for path in sorted((args.games_root / mode / 'games').glob('*.json')):
            if path.name in ('summary.json', 'failure.json'):
                continue
            g = json.loads(path.read_text())
            if mode == 'confirmation' and g.get('Reason') == 'operational-invalid' and not g.get('Result'):
                continue  # interrupted game, not a completed result
            if g['Schema'] != 'NGN-CLAIM-4' or g['ID'] in ids or path.stem != g['ID'] or g['Anchor'] not in LABELS or g['Error']:
                raise ValueError(f'invalid game {path}')
            ids.add(g['ID'])
            result = 'D' if g['Result'] == '1/2-1/2' else ('W' if (g['Result'], g['CandidateColor']) in [('1-0', 'white'), ('0-1', 'black')] else 'L')
            obs, drop, clock = analyze_game(g)
            groups[g['Anchor']].append((g, result, obs, drop, clock))
        total = collections.Counter(r for games in groups.values() for _, r, _, _, _ in games)
        expected = {'screen': (60, 150, 110, 4), 'confirmation': (94, 216, 58, 3)}[mode]
        if (total['W'], total['D'], total['L'], len(groups)) != expected:
            raise ValueError(f'{mode} totals mismatch: {total}, {len(groups)} anchors')
        output = {'WDL': [total[x] for x in 'WDL'], 'anchors': {}}
        terms = []
        for anchor, games in sorted(groups.items()):
            counts = collections.Counter(r for _, r, _, _, _ in games)
            row = {'WDL': [counts[x] for x in 'WDL'], 'termination': dict(collections.Counter(g['Reason'] for g, *_ in games)), 'end_clock_seconds_median': med([c for *_, c in games]), 'phases': {}, 'losses': {g['ID']: d for g, r, _, d, _ in games if r == 'L'}}
            for ph in ('opening', 'middlegame', 'endgame'):
                obs = [o for _, _, os, _, _ in games for o in os if o[0] == ph]
                row['phases'][ph] = {'moves': len(obs), **{key: med([o[i] for o in obs]) for i, key in enumerate(('depth', 'nodes', 'nps', 'time_ms'), 1)}}
            if mode == 'screen':
                n = len(games); score = (counts['W'] + .5 * counts['D'] + .5) / (n + 1)
                perf = LABELS[anchor] + 400 * math.log10(score / (1 - score))
                weight = n * score * (1 - score) / (400 / math.log(10))**2
                expected_perf = next(a['Performance'] for a in ref['Anchors'] if a['Name'] == anchor)
                if abs(perf - expected_perf) > 1e-8: raise ValueError(f'{anchor} performance mismatch')
                row['performance'] = perf
                terms.append((perf, weight))
            output['anchors'][anchor] = row
        if mode == 'screen':
            pool = sum(x*w for x,w in terms) / sum(w for _,w in terms)
            if abs(pool - ref['Primary']['Estimate']) > 1e-8 or round(pool, 4) != 3582.3957: raise ValueError('pool mismatch')
            output['pooled_performance'] = pool
        report['modes'][mode] = output
    clusters = collections.defaultdict(list)
    for mode, data in report['modes'].items():
        for game, drop in data['anchors'].get('Berserk 12', {}).get('losses', {}).items():
            key = (drop['phase'], drop['material']) if drop else ('no comparable drop', '')
            clusters[key].append(mode + '/' + game)
    report['berserk_largest_loss_clusters'] = [{'phase': k[0], 'material': k[1], 'games': sorted(v)} for k,v in sorted(clusters.items(), key=lambda x: (-len(x[1]), x[0]))[:3]]
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2, sort_keys=True) + '\n')
    print('validated screen and partial confirmation; wrote', args.output)

if __name__ == '__main__':
    main()
