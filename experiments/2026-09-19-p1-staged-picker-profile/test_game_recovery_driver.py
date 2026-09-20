from __future__ import annotations

import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock


HERE = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location(
    "p1_game_recovery_driver_under_test", HERE / "game_recovery_driver.py"
)
assert SPEC is not None and SPEC.loader is not None
DRIVER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(DRIVER)


class RecoveryBoundaryTests(unittest.TestCase):
    def make_null_attempt(self, root: Path) -> dict[str, Path]:
        artifacts = {
            name: root / name
            for name in DRIVER.EXPECTED_ARTIFACTS
        }
        artifacts["supervisor"].write_text(
            json.dumps({"state": "RUNNING"}), encoding="utf-8"
        )
        artifacts["witness"].write_text(
            json.dumps({"state": "STARTING"}), encoding="utf-8"
        )
        for name in ("games", "final_epd", "match_stderr"):
            artifacts[name].write_bytes(b"")
        artifacts["match_stdout"].write_text(
            "".join(f"Started game {number}\n" for number in range(1, 5)),
            encoding="utf-8",
        )
        artifacts["fastchess_log"].write_text("startup only\n", encoding="utf-8")
        artifacts["frozen_manifest"].write_text("{}\n", encoding="utf-8")
        artifacts["prelaunch_cpu"].write_text("{}\n", encoding="utf-8")
        artifacts["process_samples"].write_text("{}\n", encoding="utf-8")
        return artifacts

    def test_exact_zero_game_null_is_accepted(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            artifacts = self.make_null_attempt(root)
            with (
                mock.patch.object(DRIVER, "FAILED_ROOT", root),
                mock.patch.object(DRIVER, "live_processes_containing", return_value=[]),
            ):
                state = DRIVER.validate_null_attempt(artifacts)
            self.assertEqual(state["games_finished"], 0)
            self.assertTrue(state["empty_pgn_epd_stderr"])

    def test_any_game_material_rejects_null_classification(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            artifacts = self.make_null_attempt(root)
            artifacts["games"].write_text("[Event \"partial\"]\n", encoding="utf-8")
            with (
                mock.patch.object(DRIVER, "FAILED_ROOT", root),
                mock.patch.object(DRIVER, "live_processes_containing", return_value=[]),
                self.assertRaisesRegex(SystemExit, "games is not empty"),
            ):
                DRIVER.validate_null_attempt(artifacts)

    def test_only_original_held_template_is_accepted(self) -> None:
        template = {
            "schema": "ngn-p1-reference-picker-game-gate-v1",
            "driver_sha256": "driver-hash",
            "release_state": DRIVER.HELD,
            "run_root": str(DRIVER.FAILED_ROOT),
        }
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "template.json"
            path.write_text(json.dumps(template), encoding="utf-8")
            self.assertEqual(
                DRIVER.validate_template(path, "driver-hash")["run_root"],
                str(DRIVER.FAILED_ROOT),
            )
            template["release_state"] = DRIVER.RELEASED
            path.write_text(json.dumps(template), encoding="utf-8")
            with self.assertRaisesRegex(SystemExit, "not held"):
                DRIVER.validate_template(path, "driver-hash")


if __name__ == "__main__":
    unittest.main()
