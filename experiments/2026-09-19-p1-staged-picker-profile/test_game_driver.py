from __future__ import annotations

import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest import mock


HERE = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location("p1_game_driver_under_test", HERE / "game_driver.py")
assert SPEC is not None and SPEC.loader is not None
DRIVER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(DRIVER)


class DecisionTests(unittest.TestCase):
    def test_pilot_and_confirmation_boundaries(self) -> None:
        self.assertEqual(
            DRIVER.decide({"elo": 20.0, "upper95_elo": -0.01}, True),
            "SHELVE_CLEARLY_BAD_PILOT",
        )
        self.assertEqual(
            DRIVER.decide({"elo": 0.0, "upper95_elo": 10.0}, False),
            "SHELVE_NONPOSITIVE_WITHOUT_MECHANISM_ADVANTAGE",
        )
        pilot = {"elo": 0.0, "upper95_elo": 10.0}
        self.assertEqual(DRIVER.decide(pilot, True), "NEED_CONFIRMATION")
        self.assertEqual(
            DRIVER.decide(pilot, True, {"lower95_elo": 0.0}),
            "SHELVE",
        )
        self.assertEqual(
            DRIVER.decide(pilot, True, {"lower95_elo": 0.000001}),
            "ACCEPT",
        )


class ProcSnapshotTests(unittest.TestCase):
    def test_disappearing_proc_entry_is_ignored(self) -> None:
        with mock.patch.object(Path, "read_text", side_effect=ProcessLookupError):
            self.assertIsNone(DRIVER.read_proc_entry(Path("/proc/123")))

    def test_proc_entry_is_parsed(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            entry = Path(directory)
            fields = ["123", "(worker)", "S", "7"] + ["0"] * 9 + ["11", "13"]
            (entry / "stat").write_text(" ".join(fields) + "\n", encoding="utf-8")
            (entry / "status").write_text("Name:\tengine\n", encoding="utf-8")
            self.assertEqual(DRIVER.read_proc_entry(entry), (123, 24, 7, "engine"))


if __name__ == "__main__":
    unittest.main()
