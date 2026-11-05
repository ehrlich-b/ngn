#!/usr/bin/env python3
"""Generate only synthetic H32 parameters and an independent integer oracle."""
import argparse
import hashlib
import json
from pathlib import Path
import struct
import zlib


def trunc(n, d):
    return n // d if n >= 0 else -((-n) // d)


def clip(x, lo=0, hi=255):
    return min(hi, max(lo, x))


def fixture():
    h = 32
    mapping = []
    for square in range(64):
        rank, file = square // 8, min(square % 8, 7 - square % 8)
        mapping.append(rank * 2 + file // 2 if rank < 2 else
                       4 if rank == 2 else 5 if rank == 3 else 6 if rank < 6 else 7)
    weights = [((i * 13 + i // h * 7) % 33) - 16 for i in range(6144 * h)]
    bias = [(i * 41) % 384 - 64 for i in range(h)]
    linear = [(i * 29) % 513 - 256 for i in range(8 * 2 * h)]
    lb = [b * 137 - 463 for b in range(8)]
    hidden = [(i * 17 + i // (2 * h)) % 65 - 32 for i in range(16 * 2 * h)]
    hb = [j * 2311 - 15001 for j in range(16)]
    output = [(i * 37) % 401 - 200 for i in range(8 * 16)]
    rb = [b * 257 - 1001 for b in range(8)]
    data = b'NGN6' + struct.pack('<7I', 6, h, 8, 8, 255, 64, 400)
    data += bytes(mapping) + struct.pack('<6I', 16, 64, 255, 64, 400, 0)
    for values, code in ((weights, 'h'), (bias, 'h'), (linear, 'h'), (lb, 'i'),
                         (hidden, 'h'), (hb, 'i'), (output, 'h'), (rb, 'i')):
        data += struct.pack('<' + code * len(values), *values)
    data += struct.pack('<I', zlib.crc32(data))
    assert len(data) == 396860
    cases = []
    for b in range(8):
        board = [''] * 64  # rank 1 first, a-file first
        board[(b % 4) * 8 + b] = 'K'
        board[63 - b * 3] = 'k'
        count = 0
        for i in range(64):
            sq = (i * 19 + b * 7) % 64
            if not board[sq] and count < b * 4:
                board[sq] = 'PNBRQpnbrq'[count % 10]
                count += 1
        rows = []
        for rank in range(7, -1, -1):
            row, empty = '', 0
            for piece in board[rank * 8:(rank + 1) * 8]:
                if piece:
                    row += (str(empty) if empty else '') + piece
                    empty = 0
                else:
                    empty += 1
            rows.append(row + (str(empty) if empty else ''))
        accum = []
        for own_white in (True, False):
            king = board.index('K' if own_white else 'k')
            oriented = king if own_white else king ^ 56
            mirror = 7 if oriented % 8 >= 4 else 0
            kb = mapping[oriented ^ mirror]
            a = bias.copy()
            for sq, piece in enumerate(board):
                if piece:
                    relative = 0 if piece.isupper() == own_white else 1
                    transformed = (sq if own_white else sq ^ 56) ^ mirror
                    feature = kb * 768 + relative * 384 + 'pnbrqk'.index(piece.lower()) * 64 + transformed
                    for i in range(h):
                        a[i] += weights[feature * h + i]
            accum += a
        for turn in ('w', 'b'):
            ordered = accum if turn == 'w' else accum[h:] + accum[:h]
            x = [clip(a) for a in ordered]
            s = sum(a * a * w for a, w in zip(x, linear[b * 2 * h:(b + 1) * 2 * h]))
            linear_cp = trunc((trunc(s, 255) + lb[b]) * 400, 255 * 64)
            z = [clip(trunc(hb[j] + sum(x[i] * hidden[j * 2 * h + i]
                                       for i in range(2 * h)), 64)) for j in range(16)]
            projection = rb[b] + sum(z[j] * output[b * 16 + j] for j in range(16))
            residual_cp = trunc(projection * 400, 255 * 64)
            cases.append(dict(fen='/'.join(rows) + f' {turn} - - 0 1', bucket=b,
                              accum=accum, linear_cp=linear_cp, residual_cp=residual_cp,
                              cp=clip(linear_cp + residual_cp, -24999, 24999)))
    oracle = dict(format='NGNN6', hidden=h, network_sha256=hashlib.sha256(data).hexdigest(), cases=cases)
    return {'synthetic-h32.ngn6': data, 'oracle-h32.json': (json.dumps(oracle, indent=2) + '\n').encode()}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument('--write', action='store_true')
    mode.add_argument('--check', action='store_true')
    args = parser.parse_args()
    root = Path(__file__).resolve().parent
    for name, data in fixture().items():
        if args.write:
            (root / name).write_bytes(data)
        else:
            assert (root / name).read_bytes() == data, f'{name} differs'
        print(f'{name}: {len(data)} bytes sha256={hashlib.sha256(data).hexdigest()}')


if __name__ == '__main__':
    main()
