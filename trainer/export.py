"""Quantize float parameters and write the little-endian NGNN1 + IEEE CRC32."""

import argparse
from pathlib import Path
import struct
import zlib

import numpy as np

QA, QB, SCALE = 255, 64, 400
HEADER = struct.Struct("<4s5I")


def quantize(value, multiplier: int, dtype: str) -> np.ndarray:
    if hasattr(value, "detach"):
        value = value.detach().cpu().numpy()
    value = np.asarray(value, dtype=np.float64)
    rounded = np.rint(value * multiplier)  # Nearest integer, ties to even.
    bounds = np.iinfo(np.dtype(dtype))
    if not np.all(np.isfinite(rounded)) or np.any((rounded < bounds.min) | (rounded > bounds.max)):
        raise ValueError("Parameter cannot be represented after quantization")
    return rounded.astype(dtype)


def export_state(state: dict, path: str | Path) -> None:
    w1 = quantize(state["w1"], QA, "<i2")
    hidden = w1.shape[1] if w1.ndim == 2 else 0
    if w1.shape != (768, hidden) or not 16 <= hidden <= 2048 or hidden % 16:
        raise ValueError("Expected W1 [768,H], H a multiple of 16 in [16,2048]")
    b1 = quantize(state["b1"], QA, "<i2")
    o = quantize(state["o"], QB, "<i2")
    ob = quantize(state["ob"], QA * QB, "<i4")
    if b1.shape != (hidden,) or o.shape != (2 * hidden,) or ob.shape != ():
        raise ValueError("Invalid NGNN1 parameter shape")
    original_o = state["o"]
    if hasattr(original_o, "detach"):
        original_o = original_o.detach().cpu().numpy()
    if np.any(np.abs(np.asarray(original_o, dtype=np.float64)) > 1.980001):
        raise ValueError("Output float weights must be clipped to [-1.98,1.98]")
    data = HEADER.pack(b"NGNN", 1, hidden, QA, QB, SCALE)
    data += w1.tobytes() + b1.tobytes() + o.tobytes() + ob.tobytes()
    data += struct.pack("<I", zlib.crc32(data))
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(path.name + ".tmp")
    temporary.write_bytes(data)
    temporary.replace(path)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("checkpoint")
    parser.add_argument("output")
    args = parser.parse_args()
    import torch
    checkpoint = torch.load(args.checkpoint, map_location="cpu", weights_only=True)
    export_state(checkpoint["model"], args.output)


if __name__ == "__main__":
    main()
