"""Quantize float parameters and write little-endian NGNN1/2/3 + IEEE CRC32."""

import argparse
from pathlib import Path
import struct
import zlib

import numpy as np

from trainer.king_buckets import validate_map

QA, QB, SCALE = 255, 64, 400
HEADER = struct.Struct("<4s5I")
HEADER2 = struct.Struct("<4s6I")
HEADER3 = struct.Struct("<4s7I64s")
HEADER4 = struct.Struct("<4s8I64s")


def quantize(value, multiplier: int, dtype: str) -> np.ndarray:
    if hasattr(value, "detach"):
        value = value.detach().cpu().numpy()
    value = np.asarray(value, dtype=np.float64)
    rounded = np.rint(value * multiplier)  # Nearest integer, ties to even.
    bounds = np.iinfo(np.dtype(dtype))
    if not np.all(np.isfinite(rounded)) or np.any((rounded < bounds.min) | (rounded > bounds.max)):
        raise ValueError("Parameter cannot be represented after quantization")
    return rounded.astype(dtype)


def export_state(state: dict, path: str | Path, format: int | None = None) -> None:
    king_map = validate_map(state["king_map"]) if "king_map" in state else None
    king_buckets = max(king_map) + 1 if king_map is not None else 1
    transformer = state["w1"]
    if "w1_shared" in state:
        shared = state["w1_shared"]
        if hasattr(transformer, "detach"):
            transformer = transformer.detach().cpu().numpy()
        if hasattr(shared, "detach"):
            shared = shared.detach().cpu().numpy()
        transformer, shared = np.asarray(transformer), np.asarray(shared)
        if (king_map is None or king_buckets < 2 or transformer.ndim != 2
                or transformer.shape[0] != king_buckets * 768
                or shared.shape != (768, transformer.shape[1])
                or transformer.dtype != np.float32 or shared.dtype != np.float32):
            raise ValueError("King factorizer requires float32 W1 [KB*768,H] and shared [768,H]")
        # Match the model's float32 sum, then quantize once. Quantizing the two
        # components separately changes rounding and can overflow int16.
        transformer = (transformer.reshape(king_buckets, *shared.shape) + shared).reshape(transformer.shape)
    w1 = quantize(transformer, QA, "<i2")
    hidden = w1.shape[1] if w1.ndim == 2 else 0
    if w1.shape != (768 * king_buckets, hidden) or not 16 <= hidden <= 2048 or hidden % 16:
        raise ValueError("Expected W1 [KB*768,H], H a multiple of 16 in [16,2048]")
    b1 = quantize(state["b1"], QA, "<i2")
    h2 = 0
    layer2 = None
    if "w2" in state or "b2" in state:
        if "w2" not in state or "b2" not in state or king_map is None:
            raise ValueError("NGNN4 requires W2, B2 and a king map")
        layer2 = quantize(state["w2"], QB, "i1")
        b2 = quantize(state["b2"], QA, "<i2")
        h2 = layer2.shape[1] if layer2.ndim == 3 else 0
        nb = layer2.shape[0] if layer2.ndim == 3 else 0
        if (not 16 <= h2 <= 64 or h2 % 16 or nb not in (1, 2, 4, 8)
                or layer2.shape != (nb, h2, 2 * hidden) or b2.shape != (nb, h2)
                or np.any((layer2 < -64) | (layer2 > 64))):
            raise ValueError("Invalid NGNN4 shape or second-layer weight outside [-64,64]")
    o = quantize(state["o"], QB, "<i2")
    ob = quantize(state["ob"], QA * QB, "<i4")
    buckets = 1 if o.ndim == 1 else o.shape[0] if o.ndim == 2 else 0
    expected_o = (buckets, h2) if h2 else (2 * hidden,) if o.ndim == 1 else (buckets, 2 * hidden)
    expected_ob = () if o.ndim == 1 or (h2 and buckets == 1) else (buckets,)
    if buckets not in (1, 2, 4, 8) or b1.shape != (hidden,) or o.shape != expected_o or ob.shape != expected_ob:
        raise ValueError("Invalid NGN parameter shape or bucket count")
    if h2 and buckets != nb:
        raise ValueError("NGNN4 layer and output bucket counts differ")
    format = (4 if h2 else 3 if king_map is not None else 2 if buckets > 1 else 1) if format is None else format
    if (format not in (1, 2, 3, 4) or (format == 1 and buckets != 1)
            or (format >= 3) != (king_map is not None) or (format == 4) != bool(h2)):
        raise ValueError("NGNN1 requires one output bucket; king maps require NGNN3/4 and W2 requires NGNN4")
    original_o = state["o"]
    if hasattr(original_o, "detach"):
        original_o = original_o.detach().cpu().numpy()
    if np.any(np.abs(np.asarray(original_o, dtype=np.float64)) > 1.980001):
        raise ValueError("Output float weights must be clipped to [-1.98,1.98]")
    if format == 4:
        data = HEADER4.pack(b"NGN4", 4, hidden, king_buckets, buckets, h2, QA, QB, SCALE, bytes(king_map))
    elif format == 3:
        data = HEADER3.pack(b"NGN3", 3, hidden, king_buckets, buckets, QA, QB, SCALE, bytes(king_map))
    else:
        data = (HEADER.pack(b"NGNN", 1, hidden, QA, QB, SCALE) if format == 1 else
                HEADER2.pack(b"NGN2", 2, hidden, buckets, QA, QB, SCALE))
    data += w1.tobytes() + b1.tobytes()
    if h2:
        data += layer2.tobytes() + b2.tobytes()
    data += o.tobytes() + ob.tobytes()
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
    parser.add_argument("--format", type=int, choices=(1, 2, 3, 4), help="Default: checkpoint format or infer from parameters")
    args = parser.parse_args()
    import torch
    checkpoint = torch.load(args.checkpoint, map_location="cpu", weights_only=True)
    export_state(checkpoint["model"], args.output, args.format or checkpoint.get("config", {}).get("format"))


if __name__ == "__main__":
    main()
