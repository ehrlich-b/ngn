#!/usr/bin/env python3
"""Reproduce the small owned integer model without tensors or dependencies."""
import argparse
from pathlib import Path
import struct
import zlib


def model_bytes():
    hidden = 16
    body = struct.pack("<4s5I", b"NGNN", 1, hidden, 255, 64, 400)
    body += struct.pack("<" + "h" * (768 * hidden), *[((i * 17 + 3) % 7) - 3 for i in range(768 * hidden)])
    body += struct.pack("<16h", *[10 + i for i in range(hidden)])
    body += struct.pack("<32h", *[1 if i % 3 else -1 for i in range(32)])
    body += struct.pack("<i", 23)
    return body + struct.pack("<I", zlib.crc32(body))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--output", type=Path)
    group.add_argument("--check", action="store_true")
    args = parser.parse_args()
    if args.check:
        assert model_bytes() == Path(__file__).with_name("tiny-owned.nnue").read_bytes()
        print("owned H16 model reproduced exactly (24704 bytes)")
    else:
        args.output.write_bytes(model_bytes())
