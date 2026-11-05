#!/usr/bin/env python3
"""Regenerate owned H32 models, 32 packed records and independent score receipt."""
import argparse
import hashlib
import json
from pathlib import Path
import struct
import sys
import zlib

import numpy as np

sys.path.insert(0, str(Path(__file__).resolve().parents[3]))
from trainer.king_buckets import DEFAULT_MAP
from trainer.ngnp import Position, encode_record, parse_fen, to_fen
from trainer.residual_export import SEED, lift_bytes, receipt, residual_bytes
from trainer.residual_refeval import Network

HERE = Path(__file__).resolve().parent


def source_bytes():
    rng = np.random.Generator(np.random.PCG64(SEED))
    # File-defined map deliberately differs from the default map.
    mapping = bytes((v+3) % 8 for v in DEFAULT_MAP)
    data = struct.pack("<4s7I64s", b"NGN3", 3, 32, 8, 8, 255, 64, 400, mapping)
    for low, high, shape, dtype in (
            (-12, 13, (6144, 32), "<i2"), (-40, 301, (32,), "<i2"),
            (-96, 97, (8, 64), "<i2"), (-25000, 25001, (8,), "<i4")):
        data += rng.integers(low, high, size=shape, dtype=np.int64).astype(dtype).tobytes()
    return data + struct.pack("<I", zlib.crc32(data))


def nonzero_bytes(source):
    rng = np.random.Generator(np.random.PCG64(SEED+1))
    a = rng.integers(-8, 9, size=(16, 64), dtype=np.int64)
    hb = rng.integers(-1024, 1025, size=16, dtype=np.int64)
    r = rng.integers(-90, 91, size=(8, 16), dtype=np.int64)
    rb = rng.integers(-16000, 16001, size=8, dtype=np.int64)
    return residual_bytes(source, a, hb, r, rb)


def positions():
    for bucket in range(8):
        # 2,6,...,30 pieces exercise every material bucket. Both king contexts
        # cross ranks, input buckets and horizontal orientations.
        king = (bucket*8+1) % 64
        other = king ^ 56 ^ 7
        available = [s for s in range(8, 56) if s not in (king, other)]
        squares = np.array([king, other] + available[:bucket*4], dtype=np.int64)
        pieces = np.array([5, 11] + [0 if i % 2 == 0 else 6 for i in range(bucket*4)])
        order = squares.argsort()
        for mirror in (0, 7):
            for stm in (0, 1):
                yield Position(squares[order] ^ mirror, pieces[order], stm)


TRANSITIONS = [
    ("capture", "4k3/8/8/8/8/8/p7/R3K3 w - - 0 1", "a1a2", "4k3/8/8/8/8/8/R7/4K3 b - - 0 1"),
    ("castle", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", "e1g1", "r3k2r/8/8/8/8/8/8/R4RK1 b kq - 1 1"),
    ("castle_black", "r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1", "e8c8", "2kr3r/8/8/8/8/8/8/R3K2R w KQ - 1 2"),
    ("promotion_capture", "1r2k3/P7/8/8/8/8/8/4K3 w - - 0 1", "a7b8q", "1Q2k3/8/8/8/8/8/8/4K3 b - - 0 1"),
    ("promotion_black", "4k3/8/8/8/8/8/p7/4K3 b - - 0 1", "a2a1n", "4k3/8/8/8/8/8/8/n3K3 w - - 0 2"),
    ("en_passant", "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1", "e5d6", "4k3/8/3P4/8/8/8/8/4K3 b - - 0 1"),
    ("en_passant_black", "4k3/8/8/8/3Pp3/8/8/4K3 b - d3 0 1", "e4d3", "4k3/8/8/8/8/3p4/8/4K3 w - - 0 2"),
    ("mirror_crossing", "7k/8/8/8/8/8/8/3K4 w - - 0 1", "d1e1", "7k/8/8/8/8/8/8/4K3 b - - 1 1"),
    ("bucket_crossing", "7k/8/8/8/8/8/8/1K6 w - - 0 1", "b1c1", "7k/8/8/8/8/8/8/2K5 b - - 1 1"),
]


def build_packet():
    source = source_bytes()
    zero = lift_bytes(source)
    nonzero = nonzero_bytes(source)
    base, lift, residual = (Network(raw) for raw in (source, zero, nonzero))
    def case(position):
        linear, correction = residual.components(position)
        assert base.evaluate(position) == lift.evaluate(position)
        return {"fen": to_fen(position), "bucket": base.bucket(position),
                "zero_cp": lift.evaluate(position), "nonzero_cp": residual.evaluate(position),
                "linear_cp": linear, "residual_cp": correction,
                "acc": residual.accumulators(position).reshape(-1).tolist()}
    records, cases = [], []
    for p in positions():
        c = case(p)
        cases.append(c)
        records.append(encode_record(p, c["zero_cp"]).tobytes())
    transitions = []
    for name, before, move, after in TRANSITIONS:
        p = parse_fen(before)
        null = Position(p.squares, p.pieces, 1-p.stm)
        transitions.append({"name": name, "fen": before, "move": move,
                            "before": case(p), "after": case(parse_fen(after)), "null": case(null)})
    result = receipt(source, zero)
    result["nonzero"] = {"seed": SEED+1, "format": "NGNN6", "file_bytes": len(nonzero),
                         "payload_bytes": len(nonzero)-124, "sha256": hashlib.sha256(nonzero).hexdigest()}
    result.update(cases=cases, transitions=transitions, record_count=32,
                  integer_parity="independent NumPy int64; Go production parity checked by TestWave2ResidualParity",
                  nonzero_differs=sum(c["nonzero_cp"] != c["zero_cp"] for c in cases))
    packed = b"".join(records)
    result["records_sha256"] = hashlib.sha256(packed).hexdigest()
    return {"source-h32.ngn3": source, "zero-h32.ngn6": zero,
            "nonzero-h32.ngn6": nonzero, "records.ngnp": packed,
            "parity.json": (json.dumps(result, indent=2)+"\n").encode()}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    for name, data in build_packet().items():
        if args.check:
            assert (HERE/name).read_bytes() == data, f"stale fixture: {name}"
        else:
            (HERE/name).write_bytes(data)
        print(f"{name}: {len(data)} bytes sha256={hashlib.sha256(data).hexdigest()}")
