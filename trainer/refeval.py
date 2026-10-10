"""Independent integer reference: int64 accumulators, Go-style signed division."""

import argparse
from pathlib import Path
import struct
import zlib

import numpy as np

from trainer.ngnp import Position, decode_record, feature_indices, parse_fen
from trainer.king_buckets import feature_indices as king_features


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
        king_buckets, self.king_map, self.hidden2 = 1, None, 0
        if (magic, version) == (b"NGNN", 1):
            offset, buckets = 24, 1
            qa, qb, scale = struct.unpack_from("<3I", data, 12)
        elif (magic, version) == (b"NGN2", 2):
            offset = 28
            buckets, qa, qb, scale = struct.unpack_from("<4I", data, 12)
        elif (magic, version) in ((b"NGN3", 3), (b"NGN4", 4)):
            offset = 100 if version == 4 else 96
            if len(data) < offset:
                raise ValueError("Truncated NGNN3 header")
            if version == 4:
                king_buckets, buckets, self.hidden2, qa, qb, scale = struct.unpack_from("<6I", data, 12)
                if not 16 <= self.hidden2 <= 64 or self.hidden2 % 16:
                    raise ValueError("Invalid NGNN4 second hidden size")
            else:
                king_buckets, buckets, qa, qb, scale = struct.unpack_from("<5I", data, 12)
            self.king_map = tuple(data[offset-64:offset])
            if not 1 <= king_buckets <= 8 or any(b >= king_buckets for b in self.king_map):
                raise ValueError("Invalid NGNN3 king layout")
        else:
            raise ValueError("Unsupported NGNN magic/version")
        if buckets not in (1, 2, 4, 8):
            raise ValueError("Invalid NGNN2 bucket count")
        if not 16 <= hidden <= 2048 or hidden % 16:
            raise ValueError("Invalid NGNN1 hidden size")
        if (qa, qb, scale) != (255, 64, 400):
            raise ValueError("Invalid NGNN1 quantization constants")
        features = king_buckets * 768
        count = (features + 1) * hidden + (0 if self.hidden2 else buckets * 2 * hidden)
        expected = offset + count * 2 + buckets * 4 + 4
        if self.hidden2:
            expected += buckets * self.hidden2 * (2 * hidden + 4)
        if len(data) != expected:
            raise ValueError("Wrong NGNN1 payload length")
        stored_crc = struct.unpack_from("<I", data, len(data) - 4)[0]
        if zlib.crc32(data[:-4]) != stored_crc:
            raise ValueError("NGNN1 CRC mismatch")
        self.hidden, self.qa, self.qb, self.scale = hidden, qa, qb, scale
        self.version, self.buckets = version, buckets
        self.king_buckets = king_buckets
        # Widen before any arithmetic; valid int16 endpoints cannot overflow.
        weights = np.frombuffer(data, dtype="<i2", count=count, offset=offset).astype(np.int64)
        self.w1 = weights[:features * hidden].reshape(features, hidden)
        self.b1 = weights[features * hidden:(features + 1) * hidden]
        self.o = weights[(features + 1) * hidden:]
        if self.hidden2:
            offset += count * 2
            count2 = buckets * self.hidden2 * 2 * hidden
            self.w2 = np.frombuffer(data, dtype="i1", count=count2, offset=offset).astype(np.int64).reshape(buckets, self.hidden2, 2 * hidden)
            if np.any((self.w2 < -64) | (self.w2 > 64)):
                raise ValueError("NGNN4 second-layer weights outside [-64,64]")
            offset += count2
            self.b2 = np.frombuffer(data, dtype="<i2", count=buckets*self.hidden2, offset=offset).astype(np.int64).reshape(buckets, self.hidden2)
            offset += buckets * self.hidden2 * 2
            self.o = np.frombuffer(data, dtype="<i2", count=buckets*self.hidden2, offset=offset).astype(np.int64).reshape(buckets, self.hidden2)
            offset += buckets * self.hidden2 * 2
            count = 0
        biases = np.frombuffer(data, dtype="<i4", count=buckets, offset=offset + count * 2).astype(np.int64)
        self.ob = int(biases[0]) if buckets == 1 else biases
        if buckets > 1 and not self.hidden2:
            self.o = self.o.reshape(buckets, 2 * hidden)

    def bucket(self, position: Position) -> int:
        return max(0, min(self.buckets - 1, (len(position.squares) - 2) * self.buckets // 32))

    def evaluate(self, position: Position) -> int:
        indices = [feature_indices(position, p) if self.king_map is None else
                   king_features(position, p, self.king_map) for p in (0, 1)]
        acc = [self.b1 + self.w1[index].sum(axis=0, dtype=np.int64) for index in indices]
        us = np.clip(acc[position.stm], 0, self.qa)
        them = np.clip(acc[1 - position.stm], 0, self.qa)
        bucket = self.bucket(position)
        bias = self.ob if self.buckets == 1 else int(self.ob[bucket])
        if self.hidden2:
            sums = self.w2[bucket] @ np.concatenate((us, them))
            middle = np.asarray([trunc_div(int(v), self.qb) for v in sums]) + self.b2[bucket]
            total = int((np.clip(middle, 0, self.qa) ** 2) @ self.o[bucket])
        else:
            weights = self.o if self.buckets == 1 else self.o[bucket]
            total = int(us ** 2 @ weights[:self.hidden] + them ** 2 @ weights[self.hidden:])
        value = trunc_div(total, self.qa) + bias
        cp = trunc_div(value * self.scale, self.qa * self.qb)
        # Preserve the legacy reference's unbounded diagnostic scores.
        return max(-24999, min(24999, cp)) if self.version >= 3 else cp

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
