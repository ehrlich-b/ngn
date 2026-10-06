"""Independent integer reference: int64 accumulators, Go-style signed division."""

import argparse
from pathlib import Path
import struct
import zlib

import numpy as np

from trainer.ngnp import Position, decode_record, feature_indices, parse_fen


def trunc_div(numerator: int, denominator: int) -> int:
    if denominator <= 0:
        raise ValueError("Divisor must be positive")
    return numerator // denominator if numerator >= 0 else -((-numerator) // denominator)


class Network:
    def __init__(self, path: str | Path):
        data = Path(path).read_bytes()
        if len(data) < 32:
            raise ValueError("Truncated NGN file")
        magic, version, hidden = struct.unpack_from("<4s2I", data)
        if (magic, version) == (b"NGNN", 1):
            offset, buckets = 24, 1
            qa, qb, scale = struct.unpack_from("<3I", data, 12)
        elif (magic, version) == (b"NGN2", 2):
            offset = 28
            buckets, qa, qb, scale = struct.unpack_from("<4I", data, 12)
        else:
            raise ValueError("Unsupported NGNN magic/version")
        if buckets not in (1, 2, 4, 8):
            raise ValueError("Invalid NGNN2 bucket count")
        if not 16 <= hidden <= 2048 or hidden % 16:
            raise ValueError("Invalid NGNN1 hidden size")
        if (qa, qb, scale) != (255, 64, 400):
            raise ValueError("Invalid NGNN1 quantization constants")
        count = 769 * hidden + buckets * 2 * hidden
        expected = offset + count * 2 + buckets * 4 + 4
        if len(data) != expected:
            raise ValueError("Wrong NGNN1 payload length")
        stored_crc = struct.unpack_from("<I", data, len(data) - 4)[0]
        if zlib.crc32(data[:-4]) != stored_crc:
            raise ValueError("NGNN1 CRC mismatch")
        self.hidden, self.qa, self.qb, self.scale = hidden, qa, qb, scale
        self.version, self.buckets = version, buckets
        # Widen before any arithmetic; valid int16 endpoints cannot overflow.
        weights = np.frombuffer(data, dtype="<i2", count=count, offset=offset).astype(np.int64)
        self.w1 = weights[:768 * hidden].reshape(768, hidden)
        self.b1 = weights[768 * hidden:769 * hidden]
        self.o = weights[769 * hidden:]
        biases = np.frombuffer(data, dtype="<i4", count=buckets, offset=offset + count * 2).astype(np.int64)
        self.ob = int(biases[0]) if buckets == 1 else biases
        if buckets > 1:
            self.o = self.o.reshape(buckets, 2 * hidden)

    def bucket(self, position: Position) -> int:
        return max(0, min(self.buckets - 1, (len(position.squares) - 2) * self.buckets // 32))

    def evaluate(self, position: Position) -> int:
        acc = [self.b1 + self.w1[feature_indices(position, p)].sum(axis=0, dtype=np.int64) for p in (0, 1)]
        us = np.clip(acc[position.stm], 0, self.qa) ** 2
        them = np.clip(acc[1 - position.stm], 0, self.qa) ** 2
        bucket = self.bucket(position)
        weights = self.o if self.buckets == 1 else self.o[bucket]
        bias = self.ob if self.buckets == 1 else int(self.ob[bucket])
        total = int(us @ weights[:self.hidden] + them @ weights[self.hidden:])
        value = trunc_div(total, self.qa) + bias
        return trunc_div(value * self.scale, self.qa * self.qb)

    def evaluate_fen(self, fen: str) -> int:
        return self.evaluate(parse_fen(fen))

    def evaluate_record(self, record: np.void) -> int:
        return self.evaluate(decode_record(record))


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("network")
    parser.add_argument("fens", nargs="*", help="Quoted FENs")
    parser.add_argument("--fen-file", type=Path, help="One FEN per line")
    args = parser.parse_args()
    fens = args.fens + (args.fen_file.read_text().splitlines() if args.fen_file else [])
    if not fens:
        import sys
        fens = sys.stdin.read().splitlines()
    network = Network(args.network)
    for fen in fens:
        if fen.strip():
            print(f"{network.evaluate_fen(fen)}\t{fen}")


if __name__ == "__main__":
    main()
