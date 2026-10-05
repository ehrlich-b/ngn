"""Run with python -m unittest discover -s trainer/tests -v (on WSL)."""

import hashlib
import json
from pathlib import Path
import struct
import subprocess
import sys
import tempfile
import unittest
import zlib

import numpy as np
import torch

from trainer.export import export_state, quantize
from trainer.make_random_net import generate
from trainer.model import NGNN, blended_loss, decode_gpu
from trainer.ngnp import (Position, RECORD_DTYPE, Shards, decode_record, decode_records,
                         encode_record, feature_indices, parse_fen, to_fen)
from trainer.refeval import Network, trunc_div
from trainer.tests.synthetic import random_fens, random_records
from trainer.train import epoch_batches, learning_rate, parser


def mirrored(position):
    return Position(position.squares ^ 56, (position.pieces + 6) % 12, 1 - position.stm)


def scalar_evaluate(network, position):
    activations = []
    for perspective in (position.stm, 1 - position.stm):
        values = []
        for i in range(network.hidden):
            accumulator = int(network.b1[i])
            for square, piece in zip(position.squares, position.pieces):
                index = (int(piece) // 6 != perspective) * 384 + (int(piece) % 6) * 64 + (int(square) ^ (56 * perspective))
                accumulator += int(network.w1[index, i])
            values.append(max(0, min(network.qa, accumulator)) ** 2)
        activations += values
    total = sum(value * int(weight) for value, weight in zip(activations, network.o))
    value = trunc_div(total, network.qa) + network.ob
    return trunc_div(value * network.scale, network.qa * network.qb)


class TrainerTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="ngn-trainer-test-")
        self.path = Path(self.temporary.name)

    def tearDown(self):
        self.temporary.cleanup()

    def test_hand_built_record_and_pov(self):
        # a1 white rook, e1 white king, e4 white pawn, d5 black queen,
        # e8 black king, h8 black rook. Includes occupancy's high bit.
        squares = [0, 4, 28, 35, 60, 63]
        occupancy = sum(1 << square for square in squares)
        wire = struct.pack("<Q16shBBHBB", occupancy, bytes([0x53, 0xA0, 0x9B]) + bytes(13), -321, 2, 1, 77, 3, 0)
        record = np.frombuffer(wire, dtype=RECORD_DTYPE)[0]
        position = decode_record(record)
        np.testing.assert_array_equal(position.squares, squares)
        np.testing.assert_array_equal(position.pieces, [3, 5, 0, 10, 11, 9])
        self.assertEqual(to_fen(position), "4k2r/8/8/3q4/4P3/8/8/R3K3 b - - 0 1")
        decoded = decode_records(np.asarray([record]))
        self.assertEqual(decoded["score_stm"][0], 321)
        self.assertEqual(decoded["result_stm"][0], 0)
        self.assertEqual(encode_record(position, -321, 2, 77, 3).tobytes(), wire)
        self.assertEqual(RECORD_DTYPE.itemsize, 32)

    def test_feature_mapping(self):
        position = Position(np.asarray([0, 63, 28]), np.asarray([3, 11, 0]), 0)
        np.testing.assert_array_equal(feature_indices(position, 0), [192, 767, 28])
        np.testing.assert_array_equal(feature_indices(position, 1), [632, 327, 420])
        mirror = mirrored(position)
        for p in (0, 1):
            np.testing.assert_array_equal(feature_indices(position, p), feature_indices(mirror, 1 - p))

    def test_shards_filters_shuffle_and_tails(self):
        records = random_records(79, np.random.default_rng(9))
        records["score"] = np.arange(79)
        records["flags"][:5] = [1, 2, 4, 8, 15]
        records["ply"] = 50
        records["ply"][5] = 2
        records[:37].tofile(self.path / "a.ngnp")
        records[37:].tofile(self.path / "b.ngnp")
        shards = Shards([str(self.path / "*.ngnp")], min_ply=10, block_records=13)
        self.assertEqual(shards.raw_count, 79)
        self.assertEqual(shards.count, 73)
        def sequence(seed):
            batches = list(shards.batches(16, seed))
            self.assertEqual([len(batch) for batch in batches], [16, 16, 16, 16, 9])
            return np.concatenate(batches).reshape(-1).view(RECORD_DTYPE)["score"]
        a, b = sequence(3), sequence(4)
        np.testing.assert_array_equal(np.sort(a), np.arange(6, 79))
        np.testing.assert_array_equal(a, sequence(3))
        self.assertFalse(np.array_equal(a, b))
        self.assertEqual(Shards([str(self.path / "*.ngnp")], drop_flags=0).count, 79)
        (self.path / "bad.ngnp").write_bytes(b"NGNP1")
        with self.assertRaisesRegex(ValueError, "multiple of 32"):
            Shards([str(self.path / "bad.ngnp")])

    def test_invalid_record_rejected(self):
        record = random_records(1, np.random.default_rng(1))
        record["stm"] = 2
        record.tofile(self.path / "bad.ngnp")
        with self.assertRaisesRegex(ValueError, "invalid NGNP1"):
            Shards([str(self.path / "bad.ngnp")])
        with self.assertRaises(ValueError):
            decode_records(record)

    def check_device_decode(self, device):
        records = random_records(23, np.random.default_rng(2))
        records = np.concatenate((records, np.asarray([encode_record(parse_fen("7k/8/8/8/8/8/8/K7 b - - 0 1"), 42, 2)])))
        # Inactive nibble padding is allowed; it must never become a feature.
        empty = np.zeros(1, dtype=RECORD_DTYPE)
        empty["pieces"] = 255
        records = np.concatenate((records, empty))
        raw = torch.from_numpy(records.view(np.uint8).reshape(-1, 32)).to(device)
        features, stm, score, result = decode_gpu(raw)
        expected = np.zeros((2 * len(records), 768), dtype=np.float32)
        for i, record in enumerate(records):
            position = decode_record(record)
            for p in (0, 1):
                expected[p * len(records) + i, feature_indices(position, p)] = 1
        np.testing.assert_array_equal(features.cpu().numpy(), expected)
        decoded = decode_records(records)
        np.testing.assert_array_equal(stm.cpu().numpy(), decoded["stm"])
        np.testing.assert_array_equal(score.cpu().numpy(), decoded["score_stm"])
        np.testing.assert_array_equal(result.cpu().numpy(), decoded["result_stm"])

    def test_cpu_decode(self):
        self.check_device_decode("cpu")

    @unittest.skipUnless(torch.cuda.is_available(), "CUDA unavailable")
    def test_gpu_decode(self):
        self.check_device_decode("cuda")

    def test_float_quantized_mirror_roundtrip(self):
        torch.manual_seed(42)
        records = random_records(96, np.random.default_rng(4))
        raw = torch.from_numpy(records.view(np.uint8).reshape(-1, 32))
        features, stm, _, _ = decode_gpu(raw)
        maximum = 0
        for hidden in (16, 32, 64):
            model = NGNN(hidden)
            with torch.no_grad():
                model.o.normal_(0, 0.025)
                model.ob.fill_(-0.0234)
                floated = model(features, stm).numpy()
            export_state(model.state_dict(), self.path / "net.nnue")
            network = Network(self.path / "net.nnue")
            for i, record in enumerate(records):
                position = decode_record(record)
                value = network.evaluate_record(record)
                self.assertEqual(value, network.evaluate_fen(to_fen(position)))
                self.assertEqual(value, network.evaluate(mirrored(position)))
                self.assertEqual(value, scalar_evaluate(network, position))
                maximum = max(maximum, abs(value - floated[i]))
                self.assertLess(abs(value - floated[i]), 5)
        print(f"max float/quantized disagreement: {maximum:.4f} cp", flush=True)

    def test_crc_layout_and_corruption(self):
        model = NGNN(32)
        export_state(model.state_dict(), self.path / "net.nnue")
        data = (self.path / "net.nnue").read_bytes()
        self.assertEqual(len(data), 49376)
        self.assertEqual(struct.unpack_from("<4s5I", data), (b"NGNN", 1, 32, 255, 64, 400))
        self.assertEqual(struct.unpack_from("<I", data, len(data) - 4)[0], zlib.crc32(data[:-4]))
        corrupted = bytearray(data)
        corrupted[80] ^= 1
        (self.path / "bad.nnue").write_bytes(corrupted)
        with self.assertRaisesRegex(ValueError, "CRC"):
            Network(self.path / "bad.nnue")
        (self.path / "bad.nnue").write_bytes(data + b"\0")
        with self.assertRaisesRegex(ValueError, "length"):
            Network(self.path / "bad.nnue")

    def test_signed_division_and_overflow(self):
        self.assertEqual(trunc_div(-254, 255), 0)
        self.assertEqual(trunc_div(-256, 255), -1)
        self.assertEqual(trunc_div(256, 255), 1)
        for hidden in (16, 2048):
            state = {"w1": np.full((768, hidden), 32767 / 255),
                     "b1": np.full(hidden, 32767 / 255),
                     "o": np.full(2 * hidden, -1.98), "ob": np.asarray(-0.001)}
            export_state(state, self.path / "net.nnue")
            network = Network(self.path / "net.nnue")
            position = parse_fen("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 1")
            total = 2 * hidden * 255 ** 2 * -127
            self.assertLess(total, -2 ** 31 if hidden == 2048 else 0)
            expected = trunc_div((trunc_div(total, 255) + network.ob) * 400, 255 * 64)
            self.assertEqual(network.evaluate(position), expected)

    def test_clipping_and_quantization(self):
        model = NGNN(16)
        with torch.no_grad():
            model.w1.fill_(1000)
            model.b1.fill_(-1000)
            model.o.fill_(10)
            model.ob.fill_(1e10)
        model.clip_quantized()
        export_state(model.state_dict(), self.path / "net.nnue")
        network = Network(self.path / "net.nnue")
        self.assertEqual(network.w1[0, 0], 32767)
        self.assertEqual(network.b1[0], -32768)
        self.assertEqual(network.o[0], 127)
        np.testing.assert_array_equal(quantize([0.5, 1.5, -0.5, -1.5], 1, "<i2"), [0, 2, 0, -2])
        with self.assertRaises(ValueError):
            quantize([float("nan")], 1, "<i2")

    def test_loss(self):
        evaluation = torch.tensor([400.0, -400.0])
        score = torch.tensor([200.0, -200.0])
        result = torch.tensor([1.0, 0.0])
        target = 0.75 / (1 + np.exp(-score.numpy() / 400)) + 0.25 * result.numpy()
        expected = np.mean((1 / (1 + np.exp(-evaluation.numpy() / 400)) - target) ** 2)
        self.assertAlmostEqual(blended_loss(evaluation, score, result).item(), float(expected), places=7)

    def test_fixture_reproducible(self):
        payload = generate(self.path / "a.nnue", self.path / "a.json")
        generate(self.path / "b.nnue", self.path / "b.json")
        self.assertEqual((self.path / "a.nnue").read_bytes(), (self.path / "b.nnue").read_bytes())
        self.assertEqual((self.path / "a.json").read_bytes(), (self.path / "b.json").read_bytes())
        self.assertEqual(len(payload["cases"]), 64)
        committed = Path("testdata/ngnn1/random_h32.nnue")
        if committed.exists():
            self.assertEqual(committed.read_bytes(), (self.path / "a.nnue").read_bytes())
            self.assertEqual(json.loads(Path("testdata/ngnn1/random_h32_evals.json").read_text()), payload)
        self.assertEqual(payload["network_sha256"], hashlib.sha256((self.path / "a.nnue").read_bytes()).hexdigest())

    def test_resume_matches_uninterrupted(self):
        records = random_records(64, np.random.default_rng(91))
        data = self.path / "data.ngnp"
        records.tofile(data)
        common = [sys.executable, "-m", "trainer.train", "--data", str(data), "--hidden", "16",
                  "--batch", "32", "--device", "cpu", "--precision", "fp32", "--threads", "2", "--decay", "none"]
        for out, epochs, resume in [("full", "2", None), ("resumed", "1", None),
                                     ("resumed", "2", self.path / "resumed/epoch-0001.pt")]:
            command = common + ["--out", str(self.path / out), "--epochs", epochs]
            if resume:
                command += ["--resume", str(resume)]
            subprocess.run(command, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=60)
        a = torch.load(self.path / "full/epoch-0002.pt", weights_only=True)
        b = torch.load(self.path / "resumed/epoch-0002.pt", weights_only=True)
        for key in a["model"]:
            self.assertTrue(torch.equal(a["model"][key], b["model"][key]), key)
        self.assertEqual(a["step"], 4)
        self.assertEqual(a["history"][-1]["loss"], b["history"][-1]["loss"])

    def test_superbatch_cycles_without_dropping_positions(self):
        records = random_records(5, np.random.default_rng(90))
        records["score"] = np.arange(5)
        records.tofile(self.path / "data.ngnp")
        shards = Shards([str(self.path / "data.ngnp")], block_records=2)
        batches = list(epoch_batches(shards, 3, 13, 2))
        self.assertEqual([len(batch) for batch in batches], [3, 2, 3, 2, 3])
        scores = np.concatenate(batches).reshape(-1).view(RECORD_DTYPE)["score"]
        np.testing.assert_array_equal(np.sort(scores[:5]), np.arange(5))
        np.testing.assert_array_equal(np.sort(scores[5:10]), np.arange(5))
        self.assertEqual(len(scores), 13)

    def test_schedule_endpoints(self):
        args = parser().parse_args(["--data", "unused", "--out", "unused", "--lr", "0.01"])
        self.assertAlmostEqual(learning_rate(args, 0, 101), 0.01)
        self.assertAlmostEqual(learning_rate(args, 50, 101), 0.0055)
        self.assertAlmostEqual(learning_rate(args, 100, 101), 0.001)
        args.decay, args.decay_steps = "step", 10
        self.assertEqual(learning_rate(args, 9, 101), 0.01)
        self.assertEqual(learning_rate(args, 10, 101), 0.005)
        self.assertEqual(learning_rate(args, 20, 101), 0.0025)


if __name__ == "__main__":
    torch.set_num_threads(4)
    unittest.main()
