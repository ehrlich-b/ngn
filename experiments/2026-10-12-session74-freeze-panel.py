"""Freeze deterministic outcome/color/move/material strata; rig only."""
import collections
import hashlib
import json
from pathlib import Path
import chess

root = Path('/home/ehrli/ngn-data/night-20261007')
p = root / 'session74-clock-panel'
source = root / 'session68-rodent/rodent-pairs.jsonl'
weights = {chess.PAWN: 1, chess.KNIGHT: 3, chess.BISHOP: 3, chess.ROOK: 5, chess.QUEEN: 9}
cells = collections.defaultdict(list)
for line in source.open():
    pair = json.loads(line)
    for game in pair['games']:
        clocks = {c['ply']: c for c in game['clocks']}
        ngn_white = game['ngn_white']
        outcome = {0: 'loss', 1: 'draw', 2: 'win'}[game['result']['Res']]
        board = chess.Board()
        for i, uci in enumerate(game['moves']):
            ply = i + 1
            fullmove = (ply + 1) // 2
            # One actual NGN turn at the start of each named move band.
            if fullmove in (16, 32, 48) and board.turn == ngn_white and ply in clocks:
                balance = sum(w * (len(board.pieces(piece, ngn_white)) - len(board.pieces(piece, not ngn_white)))
                              for piece, w in weights.items())
                material = 'deficit' if balance < 0 else 'advantage' if balance > 0 else 'balanced'
                phase = sum(w * len(board.pieces(piece, color)) for piece, w in weights.items()
                            if piece != chess.PAWN for color in (chess.WHITE, chess.BLACK))
                row = dict(id=f"{pair['pair_id']}-{'w' if ngn_white else 'b'}-{ply}", pair_id=pair['pair_id'],
                           opening=pair['opening'], ngn_white=ngn_white, outcome=outcome, move_band=fullmove,
                           material_stratum=material, material_balance=balance, nonpawn_material=phase,
                           fen=board.fen(en_passant='fen'), moves=game['moves'][:i], played_move=uci,
                           white_ms=clocks[ply]['white_before_ms'], black_ms=clocks[ply]['black_before_ms'],
                           prior_own_ply=i-2, source_clock=clocks[ply])
                cells[(ngn_white, outcome, fullmove, material)].append(row)
            board.push_uci(uci)
selected = []
for cell, choices in sorted(cells.items()):
    # Fixed seed hash, independent of evaluations and candidate results.
    choices.sort(key=lambda row: hashlib.sha256(('ngn-s74-panel-v1:' + row['id']).encode()).digest())
    selected.append(choices[0])
assert len(selected) <= 54 and len(selected) >= 30, len(selected)
(p / 'panel.json').write_text(json.dumps(selected, indent=2) + '\n')
reserved = {r['opening'] for r in selected}
all_openings = (root / 'session72-see-rig/openings.txt').read_text().splitlines()
filtered = [op for op in all_openings if op.strip() not in reserved]
assert len(filtered) >= 1000
(p / 'future-openings.txt').write_text('\n'.join(filtered) + '\n')
receipt = dict(selection='one per available color/outcome/fullmove16,32,48/material-sign cell; fixed hash seed; no eval-driven selection',
               rows=len(selected), strata={str(k):len(v) for k,v in sorted(cells.items())},
               source=str(source), source_sha256=hashlib.sha256(source.read_bytes()).hexdigest(),
               panel_sha256=hashlib.sha256((p/'panel.json').read_bytes()).hexdigest(),
               reserved_pair_ids=sorted({r['pair_id'] for r in selected}), future_book_rows=len(filtered),
               reserved_openings=len(reserved), future_book_sha256=hashlib.sha256((p/'future-openings.txt').read_bytes()).hexdigest(),
               use='diagnosis only; full move histories retained; game result is not a per-position target',
               split_limit='one selected frame per cell; source games can recur across cells; cluster summaries by opening/game')
(p / 'panel-selection.json').write_text(json.dumps(receipt, indent=2) + '\n')
print(json.dumps({k:v for k,v in receipt.items() if k!='strata'}, indent=2))
