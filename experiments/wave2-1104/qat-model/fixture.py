"""Owned H32 KB8/B8 arithmetic fixture; no training data or checkpoint needed."""

import json
from pathlib import Path

import torch

from trainer.king_buckets import DEFAULT_MAP
from trainer.model import NGNN, QA, QB

ROOT = Path(__file__).resolve().parent


def load_cases():
    return json.loads((ROOT / 'cases.json').read_text())


def make_model(fake_quantize=False):
    # Initialization is overwritten; do not perturb the caller's random stream.
    with torch.random.fork_rng(devices=[]):
        model = NGNN(32, 8, DEFAULT_MAP, king_factorizer=True,
                     fake_quantize=fake_quantize)
    with torch.no_grad():
        for parameter in model.parameters():
            parameter.zero_()
        model.b1[:2] = torch.tensor([-3, 300]) / QA
        # Each component rounds to zero, but their merged 0.6 rounds to one.
        model.w1[:, 4] = 0.3 / QA
        model.w1_shared[:, 4] = 0.3 / QA
        for square in range(64):
            if square % 8 < 4:
                kb = DEFAULT_MAP[square]
                model.w1[kb * 768 + 5 * 64 + square, 2] = (32 + 16 * kb + square % 8) / QA
        for kb in range(8):
            model.w1[kb * 768 + 8, 3] = 300 / QA
            model.w1[kb * 768 + 384 + 48, 3] = -300 / QA
        # Exact positive/negative half steps, low/high saturation and dtype ends.
        model.b1[5:9] = torch.tensor([0.5, 1.5, -0.5, -1.5])
        model.b1[13:19] = torch.tensor([-1, 0, 1, 254, 255, 256]) / QA
        model.w1[0, 20:24] = torch.tensor([0.5, 1.5, -0.5, -1.5])
        model.w1[0, 28:30] = torch.tensor([-32768, 32767]) / QA
        model.b1[30:] = torch.tensor([-32768, 32767]) / QA
        model.o[:, [0, 1, 2, 3, 4, 32, 34, 35]] = torch.tensor([64, -1, -64, 7, 64, -64, 32, -3]).float() / QB
        model.o[:, 9:13] = torch.tensor([0.5, 1.5, -0.5, -1.5]) / QB
        # Equal clipped values cancel in pairs. This exercises immediately
        # below/at zero and at/above QA without changing the hand oracle.
        model.o[:, [13, 14, 17, 18]] = torch.tensor([64, -64, 1, -1]).float() / QB
        model.o[:, 28:30] = torch.tensor([-1.98, 1.98])
        model.ob[:] = torch.tensor(load_cases()['output_bias_units'], dtype=torch.float64) / (QA * QB)
        model.clip_quantized()
    return model
