#!/usr/bin/env python3
"""Construct a tiny handcrafted NGNN6 wire fixture without any external weights."""
import argparse
from pathlib import Path
import struct
import zlib

parser = argparse.ArgumentParser()
parser.add_argument('--check', action='store_true')
args = parser.parse_args()
data = bytearray(b'NGN6' + struct.pack('<7I', 6, 32, 8, 8, 255, 64, 400))
data.extend((k // 8 + k % 8) % 8 for k in range(64))
data.extend(struct.pack('<6I', 16, 64, 255, 64, 400, 0))
def i16(values):
    for value in values:
        data.extend(struct.pack('<h', value))
def i32(values):
    for value in values:
        data.extend(struct.pack('<i', value))
i16((k * 3 + f * 5 + i * 7) % 19 - 9 for k in range(8) for f in range(768) for i in range(32))
i16(i * 19 - 150 for i in range(32))
i16((row * 5 + i * 3) % 31 - 15 for row in range(8) for i in range(64))
i32(row * 151 - 400 for row in range(8))
i16((j * 7 + i * 11) % 17 - 8 for j in range(16) for i in range(64))
i32(j * 811 - 1000 for j in range(16))
i16((row * 13 + j * 7) % 127 - 63 for row in range(8) for j in range(16))
i32(row * 521 - 2000 for row in range(8))
data.extend(struct.pack('<I', zlib.crc32(data)))
path = Path(__file__).with_name('synthetic-h32.ngn6')
assert len(data) == 396860
if args.check:
    assert path.read_bytes() == data, 'fixture differs from deterministic constructor'
else:
    path.write_bytes(data)
print(f'NGNN6 H32 handcrafted fixture: {len(data)} bytes; exact={args.check}')
