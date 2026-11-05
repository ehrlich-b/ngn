"""Reproduce the small, owned NGNN3 arithmetic fixture, without torch."""

import argparse
from fractions import Fraction
import json
from pathlib import Path

import numpy as np

from trainer.export import export_state
from trainer.king_buckets import DEFAULT_MAP
from trainer.ngnp import Position, RECORD_DTYPE, encode_record, to_fen


def build(out: Path):
    out.mkdir(parents=True, exist_ok=True)
    state = {"king_map": np.asarray(DEFAULT_MAP),
             "w1": np.zeros((8 * 768, 32)), "b1": np.zeros(32),
             "o": np.zeros((8, 64)), "ob": np.zeros(8)}
    state["b1"][:2] = np.asarray([-3, 300]) / 255
    # Only own king and the distinguished pawn can affect active channels.
    for square in range(64):
        if square % 8 < 4:
            kb = DEFAULT_MAP[square]
            state["w1"][kb * 768 + 5 * 64 + square, 2] = (32 + 16 * kb + square % 8) / 255
    for kb in range(8):
        state["w1"][kb * 768 + 8, 3] = 300 / 255
        state["w1"][kb * 768 + 384 + 48, 3] = -300 / 255
    biases = [-18, 1000, 2000, 3000, 4000, 5000, 1100000, -1100000]
    state["o"][:, [0, 1, 2, 3, 32, 34, 35]] = np.asarray([64, -1, -64, 7, -64, 32, -3]) / 64
    state["ob"] = np.asarray(biases) / (255 * 64)
    export_state(state, out / "h32.nnue", format=3)

    cases, records, predictions = [], [], []
    for bucket in range(8):
        for variant in range(4):
            wk = [0, 7, 10, 13][variant]
            bk = [59, 60, 49, 54][variant]
            stm = [0, 1, 1, 0][variant]
            count = min(32, 2 + 4 * bucket + (3 if variant >= 2 else 0))
            squares, pieces = [wk, bk], [5, 11]
            if count > 2:
                squares.append(15 if variant % 2 else 8)
                pieces.append(0)
            for square in range(64):
                if len(squares) == count:
                    break
                if square not in squares:
                    squares.append(square)
                    pieces.append(1)  # Inactive knight features fill material buckets.
            position = Position(np.asarray(squares), np.asarray(pieces), stm)
            score, result = [400, -400, 0, 800][variant], variant % 3
            record = encode_record(position, score, result)
            records.append(record)
            # Hand oracle: low/high channels clip to 0/255, own king to q,
            # distinguished pawn to 255 for White and 0 for Black.
            white_q, black_q = (32, 51) if variant < 2 else (82, 65)
            us, them = (black_q, white_q) if stm else (white_q, black_q)
            pawn_total = (-3 if stm else 7) * 255 ** 2 if count > 2 else 0
            total = -255 ** 2 - 64 * us ** 2 + 32 * them ** 2 + pawn_total
            first = int(Fraction(total, 255))  # Fraction.__int__ truncates to zero.
            cp_before_clamp = int(Fraction((first + biases[bucket]) * 400, 255 * 64))
            expected_cp = max(-24999, min(24999, cp_before_clamp))
            # Exact real-valued forward of these pre-export float parameters.
            float_cp = float(Fraction(total * 400, 255 ** 2 * 64)
                             + Fraction(biases[bucket] * 400, 255 * 64))
            prediction = None if variant == 3 else float_cp
            predictions.append(prediction)
            cases.append({"bucket": bucket, "variant": variant, "fen": to_fen(position),
                          "pieces": count, "stm": stm, "score_white": score,
                          "result_white": result, "king_q_white": white_q, "king_q_black": black_q,
                          "dot_total": total, "first_truncated": first,
                          "output_bias": biases[bucket], "cp_before_clamp": cp_before_clamp,
                          "integer_cp": expected_cp, "float_cp": prediction})
    np.asarray(records, dtype=RECORD_DTYPE).tofile(out / "records.ngnp")
    (out / "float-cp.jsonl").write_text("".join(json.dumps(p) + "\n" for p in predictions))
    (out / "cases.json").write_text(json.dumps({
        "synthetic_only": True, "hidden": 32, "qa": 255, "qb": 64, "score_scale": 400,
        "hand_oracle": "dot=-255^2-64*us_q^2+32*them_q^2+pawn; "
                       "pawn=(7 if White STM else -3)*255^2 when present; "
                       "cp=clamp(trunc((trunc(dot/255)+bias)*400/(255*64)), -24999, 24999)",
        "float_oracle": "cp=dot*400/(255^2*64)+bias*400/(255*64); every fourth prediction unknown",
        "cases": cases,
    }, indent=2) + "\n")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", type=Path, required=True)
    build(parser.parse_args().out)
