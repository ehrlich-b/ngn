"""Owned hidden-layer forward, gradients, integer export and resume tests."""

import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import numpy as np
import torch

from trainer.export import export_state
from trainer.king_buckets import DEFAULT_MAP
from trainer.make_random_net import generate
from trainer.model import NGNN, decode_gpu
from trainer.ngnp import RECORD_DTYPE, encode_record
from trainer.refeval import Network, trunc_div
from trainer.tests.test_ngnn3 import positions


class NGNN4Tests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        torch.set_num_threads(2)

    def check_forward(self, device, amp=False):
        for buckets in (1, 8):
            torch.manual_seed(8001)
            model = NGNN(32, buckets, DEFAULT_MAP, hidden2=16).to(device)
            records = np.asarray([encode_record(pos, i) for i, pos in enumerate(positions())], dtype=RECORD_DTYPE)
            raw = torch.from_numpy(records.view(np.uint8).reshape(-1, 32)).to(device)
            features, stm, _, _, bucket = decode_gpu(raw, buckets, model.king_map, model.king_buckets)
            with torch.autocast(device_type=device, dtype=torch.bfloat16, enabled=amp):
                actual = model(features, stm, bucket)
                white, black = (features @ model.effective_w1() + model.b1).clamp(0, 1).chunk(2)
                ordered = torch.cat((torch.where((stm == 0)[:, None], white, black),
                                     torch.where((stm == 0)[:, None], black, white)), 1)
                rows = []
                for i in range(len(stm)):
                    b = int(bucket[i])
                    middle = (model.w2[b] @ ordered[i]).float() + model.b2[b]
                    bias = model.ob if buckets == 1 else model.ob[b]
                    rows.append(((middle.clamp(0, 1).square() * model.o[b]).sum().float() + bias) * 400)
                expected = torch.stack(rows)
            torch.testing.assert_close(actual, expected, rtol=0.01 if amp else 1e-5, atol=0.05 if amp else 1e-5)
            actual.square().mean().backward()
            for name in ("w1", "b1", "w2", "b2", "o", "ob"):
                grad = getattr(model, name).grad
                self.assertTrue(torch.isfinite(grad).all(), name)
                self.assertGreater(grad.abs().sum().item(), 0, name)
            optimizer = torch.optim.AdamW(model.parameters(), lr=.001, fused=device == "cuda")
            before = model.w2.detach().clone()
            optimizer.step()
            model.clip_quantized()
            self.assertFalse(torch.equal(before, model.w2))
            with tempfile.TemporaryDirectory() as directory:
                path = Path(directory) / "net.nnue"
                export_state(model.state_dict(), path, 4)
                net = Network(path)
                self.assertEqual((net.version, net.hidden2), (4, 16))
                for pos in positions():
                    self.assertIsInstance(net.evaluate(pos), int)

    def test_cpu_forward_gradients_and_optimizer(self):
        self.check_forward("cpu")

    @unittest.skipUnless(torch.cuda.is_available(), "CUDA unavailable")
    def test_cuda_fp32_bf16_gradients_and_fused_step(self):
        self.check_forward("cuda")
        self.check_forward("cuda", True)

    def test_clipping_and_signed_integer_division(self):
        model = NGNN(16, 1, DEFAULT_MAP, hidden2=16)
        with torch.no_grad():
            model.w1.zero_()
            model.b1.fill_(1 / 255)
            model.w2.zero_()
            model.w2[0, 0, 0] = -1 / 64
            model.b2.zero_()
            model.b2[0, 0] = 200 / 255
            model.o.zero_()
            model.o[0, 0] = 1
            model.ob.zero_()
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "net.nnue"
            export_state(model.state_dict(), path)
            # -1/64 truncates to zero before bias, so hidden0=200.
            net = Network(path)
            cp = trunc_div(trunc_div(200 * 200 * 64, 255) * 400, 255 * 64)
            for pos in positions():
                self.assertEqual(net.evaluate(pos), cp)
            with torch.no_grad():
                model.w2.fill_(100)
                model.b2.fill_(-1000)
                model.clip_quantized()
            export_state(model.state_dict(), path)
            net = Network(path)
            self.assertEqual(net.w2.max(), 64)
            self.assertEqual(net.b2.min(), -32768)

    def test_invalid_architecture_and_exports(self):
        for hidden2 in (-1, 1, 15, 17, 80):
            with self.assertRaises(ValueError):
                NGNN(16, 8, DEFAULT_MAP, hidden2=hidden2)
        with self.assertRaises(ValueError):
            NGNN(16, hidden2=16)
        model = NGNN(16, 8, DEFAULT_MAP, hidden2=16)
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "net.nnue"
            for version in (1, 2, 3):
                with self.assertRaises(ValueError):
                    export_state(model.state_dict(), path, version)
            for name, replacement in (
                ("w2", torch.zeros(8, 17, 32)), ("w2", torch.full((8, 16, 32), 1.1)),
                ("w2", torch.full((8, 16, 32), float("nan"))),
                ("b2", torch.zeros(8, 32)), ("o", torch.zeros(8, 32))):
                state = model.state_dict().copy()
                state[name] = replacement
                with self.assertRaises(ValueError):
                    export_state(state, path)

    def test_fixture_reproducibility(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            args = dict(hidden=16, seed=20261008, buckets=8, format=4, hidden2=16)
            generate(root / "first.nnue", root / "first.json", **args)
            generate(root / "second.nnue", root / "second.json", **args)
            self.assertEqual((root / "first.nnue").read_bytes(), (root / "second.nnue").read_bytes())
            self.assertEqual((root / "first.json").read_bytes(), (root / "second.json").read_bytes())

    def test_cpu_training_resume_and_reexport(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            shard = root / "data.ngnp"
            np.asarray([encode_record(pos, i * 5) for i, pos in enumerate(positions())], dtype=RECORD_DTYPE).tofile(shard)
            common = [sys.executable, "-m", "trainer.train", "--data", str(shard), "--hidden", "16",
                      "--hidden2", "16", "--buckets", "8", "--format", "4", "--batch", "32", "--decay", "none",
                      "--device", "cpu", "--precision", "fp32", "--threads", "2"]
            env = dict(os.environ, OMP_NUM_THREADS="2", OPENBLAS_NUM_THREADS="2", MKL_NUM_THREADS="2")
            def run(extra, success=True):
                proc = subprocess.run(common + extra, env=env, capture_output=True, text=True, timeout=90)
                self.assertEqual(proc.returncode == 0, success, proc.stdout + proc.stderr)
                return proc
            run(["--out", str(root / "full"), "--epochs", "2"])
            run(["--out", str(root / "resume"), "--epochs", "1"])
            checkpoint = root / "resume/epoch-0001.pt"
            resume = ["--out", str(root / "resume"), "--epochs", "2", "--resume", str(checkpoint)]
            changed = run(resume + ["--hidden2", "32"], False)
            self.assertIn("hidden2", changed.stderr)
            run(resume)
            self.assertEqual((root / "full/epoch-0002.nnue").read_bytes(), (root / "resume/epoch-0002.nnue").read_bytes())
            proc = subprocess.run([sys.executable, "-m", "trainer.export", str(root / "full/epoch-0002.pt"),
                                   str(root / "reexport.nnue")], env=env, capture_output=True, text=True, timeout=90)
            self.assertEqual(proc.returncode, 0, proc.stderr)
            self.assertEqual((root / "reexport.nnue").read_bytes(), (root / "full/epoch-0002.nnue").read_bytes())


if __name__ == "__main__":
    unittest.main()
