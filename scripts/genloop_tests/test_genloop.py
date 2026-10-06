"""Orchestration invariants; no engine, trainer, NumPy or PyTorch needed."""

import contextlib
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

from scripts import genloop as g


class GenerationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def shard(self, name, count, status="interrupted"):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(bytes(32 * count))
        g.atomic_json(str(path) + ".json", {
            "format": "NGNP1", "closed": True, "status": status,
            "positions": count, "bytes": 32 * count, "sha256": g.sha256(path)})
        return path

    def test_atomic_manifest_and_resumable_options(self):
        m = g.Manifest(self.root)
        self.assertTrue(m.begin("datagen", {"positions": 20}, False))
        m.finish("interrupted")
        m = g.Manifest(self.root)
        with self.assertRaisesRegex(ValueError, "--resume"):
            m.begin("datagen", {"positions": 20}, False)
        with self.assertRaisesRegex(ValueError, "changed"):
            m.begin("datagen", {"positions": 21}, True)
        self.assertTrue(m.begin("datagen", {"positions": 20}, True))
        m.finish()
        self.assertFalse(g.Manifest(self.root).begin("datagen", {"positions": 20}, False))
        self.assertEqual(g.read_json(m.path)["stages"]["datagen"]["status"], "complete")
        self.assertFalse(m.path.with_suffix(".json.tmp").exists())

    def test_completed_stage_and_step_artifacts_cannot_change(self):
        shard = self.shard("a.ngnp", 2)
        m = g.Manifest(self.root)
        m.begin("train", {}, False)
        m.state["steps"]["fit"] = {"status": "complete", "artifacts": [g.identity(shard)]}
        m.finish()
        shard.write_bytes(bytes(96))
        with self.assertRaisesRegex(ValueError, "Changed artifact"):
            g.Manifest(self.root).begin("train", {}, True)

    def test_resume_refuses_a_live_recorded_child(self):
        m = g.Manifest(self.root)
        m.begin("gate", {}, False)
        m.state["attempts"].append({"pid": 1})
        m.finish("failed")
        with mock.patch.object(g.Path, "exists", return_value=True):
            with self.assertRaisesRegex(ValueError, "still exists"):
                m.begin("gate", {}, True)

    def test_generation_lock_excludes_other_stages(self):
        with g.generation_lock(self.root):
            with self.assertRaisesRegex(ValueError, "Another stage"):
                with g.generation_lock(self.root):
                    pass

    def test_target_sums_all_runs_and_floors_partial_record(self):
        self.shard("run1/a.ngnp", 3)
        path = self.shard("run2/b.ngnp", 7)
        with path.open("ab") as stream:
            stream.write(b"partial")
        self.assertEqual(g.records(self.root), 10)
        self.assertEqual(g.stop_reason(10, 10, 1, 0, {"wsl": 9, "c": 9}, 6), "target")
        self.assertEqual(g.stop_reason(11, 10, 1, 0, {"wsl": 9, "c": 9}, 6), "target")
        self.assertIsNone(g.stop_reason(9, 10, 1, 0, {"wsl": 9, "c": 9}, 6))
        with self.assertRaisesRegex(ValueError, "corrupt"):
            g.closed_shards(self.root)

    def test_disk_priority_and_time_target_distinction(self):
        self.assertEqual(g.stop_reason(20, 10, 60, 60, {"wsl": 9, "c": 5}, 6), "disk-low")
        self.assertEqual(g.stop_reason(0, 0, 60, 60, {"wsl": 9, "c": 9}, 6), "time-limit")
        self.assertEqual(g.stop_reason(1, 10, 60, 60, {"wsl": 9, "c": 9}, 6), "time-limit")
        self.assertIsNone(g.stop_reason(1, 10, 59, 60, {"wsl": 6, "c": 6}, 6))

    def test_closed_signal_shards_accepted_unclosed_or_error_refused(self):
        shard = self.shard("a.ngnp", 3)
        self.assertEqual(g.closed_shards(self.root), [g.identity(shard)])
        sidecar = Path(str(shard) + ".json")
        meta = g.read_json(sidecar)
        for key, value in [("closed", False), ("status", "error"), ("sha256", "bad")]:
            g.atomic_json(sidecar, {**meta, key: value})
            with self.assertRaisesRegex(ValueError, "corrupt"):
                g.closed_shards(self.root)

    def test_split_is_disjoint_deterministic_and_keeps_shards_whole(self):
        items = [g.identity(self.shard(f"{i}.ngnp", i)) for i in range(5)]
        first = g.split_shards(items, 0.25, 17)
        self.assertEqual(first, g.split_shards(items, 0.25, 17))
        self.assertEqual(len(first["validation"]), 1)
        self.assertFalse({x["path"] for x in first["validation"]} & {x["path"] for x in first["training"]})
        self.assertEqual(sum(x["bytes"] for rows in first.values() for x in rows), 320)
        with self.assertRaisesRegex(ValueError, "two nonempty"):
            g.split_shards(items[:2], 0.1, 1)

    def test_parity_record_decode_and_sampling(self):
        # Owned two-king NGNP1 record: white e1, black e8, black STM.
        raw = bytearray(32)
        raw[:8] = ((1 << 4) | (1 << 60)).to_bytes(8, "little")
        raw[8], raw[27] = 5 | (11 << 4), 1
        path = self.root / "kings.ngnp"
        path.write_bytes(raw * 20)
        fens = g.parity_fens([g.identity(path)])
        self.assertEqual(len(fens), 20)
        self.assertEqual(fens[0], "4k3/8/8/8/8/8/8/4K3 b - - 0 1")

    def test_paired_gate_tally_operational_errors_and_incomplete_pairs(self):
        log = ("Games: 8   W-D-L: 2-4-2\n"
               "Pentanomial [LL 1  LD 0  {LW,DD} 2  WD 0  WW 1] over 4 pairs\n"
               "Flag-outs (lost on time): new 0, base 0  of 8 games\n")
        result = g.gate_result(log, 8)
        self.assertEqual(result["paired_elo"], 0)
        self.assertLess(result["paired_ci95"][0], 0)
        self.assertGreater(result["paired_ci95"][1], 0)
        for bad in [log + "WATCHDOG", log + "illegal-move", log + "no-move", log + log,
                    log.replace("over 4", "over 3"), log.replace("base 0", "base 1"),
                    log.replace("2-4-2", "3-4-1")]:
            with self.assertRaises(ValueError):
                g.gate_result(bad, 8)

    def test_dry_run_has_no_writes_or_subprocesses_for_every_stage(self):
        for stage, options in [("datagen", ["--positions", "10"]), ("train", []),
                               ("gate", ["--openings", "/missing/openings", "--candidate-net", "/missing/net"])]:
            root = self.root / stage
            with mock.patch.object(g.subprocess, "Popen", side_effect=AssertionError("process")), \
                    mock.patch.object(g.subprocess, "run", side_effect=AssertionError("process")), \
                    contextlib.redirect_stdout(io.StringIO()) as output:
                self.assertEqual(g.main([stage, "--generation", str(root), "--dry-run", *options]), 0)
            self.assertFalse(root.exists())
            self.assertEqual(json.loads(output.getvalue())["stage"], stage)

    def test_venv_python_path_keeps_its_symlink(self):
        target = self.root / "system-python"
        target.touch()
        interpreter = self.root / "venv/bin/python"
        interpreter.parent.mkdir(parents=True)
        interpreter.symlink_to(target)
        with contextlib.redirect_stdout(io.StringIO()) as output:
            g.main(["train", "--generation", str(self.root / "generation"),
                    "--python", str(interpreter), "--dry-run"])
        self.assertEqual(json.loads(output.getvalue())["options"]["python"], str(interpreter))


if __name__ == "__main__":
    unittest.main()
