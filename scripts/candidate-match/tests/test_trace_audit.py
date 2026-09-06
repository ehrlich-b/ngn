from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))

from common import CandidateMatchError
from trace_audit import audit_trace, crash_lines

OPTIONS = [
    {"name": "Threads", "value": "1"},
    {"name": "OwnBook", "value": "false"},
    {"name": "EvalFile", "value": "<empty>"},
    {"name": "EvalBackend", "value": "hce"},
    {"name": "Hash", "value": "64"},
    {"name": "Move Overhead", "value": "50"},
]


def line(level: str, thread: str, body: str) -> str:
    return f"[{level:<6}] [00:00:00.000001] <{thread:>21}> {body}"


def roles() -> list[dict]:
    return [
        {"id": "a", "display_name": "A", "resolved_options": OPTIONS, "expected_processes": 1, "expected_refreshes": 1},
        {"id": "b", "display_name": "B", "resolved_options": OPTIONS, "expected_processes": 1, "expected_refreshes": 1},
    ]


def good_lines() -> list[str]:
    result = []
    for thread, name in (("1", "A"), ("2", "B")):
        result.append(line("TRACE", thread, f"fastchess --- Refreshing engine {name}"))
        for option in OPTIONS:
            result.append(line("TRACE", thread, f"fastchess --- Sending setoption to engine {name} {option['name']} {option['value']}"))
        result.append(line("TRACE", thread, f"fastchess --- Engine {name} refreshed."))
        result.append(line("Engine", thread, f"<stderr> {name} ---> Process exited normally with status 0"))
    return result


class TraceAuditTests(unittest.TestCase):
    def test_exact_benign_profile(self) -> None:
        report = audit_trace(good_lines(), roles())
        self.assertTrue(report["pass"])
        self.assertEqual(report["probable_embedded_book_signature_plies"], 0)
        self.assertEqual(len(report["completed_option_sequences"]), 2)

    def test_error_plus_legal_fallback_and_natural_tail_rejected(self) -> None:
        lines = good_lines() + [
            line("Engine", "1", "A ---> info string error search recovered"),
            line("Engine", "1", "A ---> info depth 2 score cp 0 nodes 42 time 1 nps 42000 pv e2e4"),
            line("Engine", "1", "A ---> bestmove e2e4"),
            line("INFO", "1", "fastchess --- Game 1 finished: White mates"),
        ]
        with self.assertRaisesRegex(CandidateMatchError, "canonical engine error"):
            audit_trace(lines, roles())

    def test_book_signature_and_illegal_pv_rejected(self) -> None:
        with self.assertRaisesRegex(CandidateMatchError, "opening-book signature"):
            audit_trace(good_lines() + [line("Engine", "1", "A ---> info depth 1 score cp 50 nodes 1 time 0 nps 0 pv e2e4")], roles())
        with self.assertRaisesRegex(CandidateMatchError, "illegal pv"):
            audit_trace(good_lines() + [line("WARN", "1", "fastchess --- Illegal PV from engine A")], roles())

    def test_generic_fatal_log_levels_are_rejected_without_phrase_match(self) -> None:
        for level in ("ERROR", "FATAL", "CRITICAL"):
            with self.subTest(level=level):
                with self.assertRaisesRegex(CandidateMatchError, "fatal log level"):
                    audit_trace(good_lines() + [line(level, "1", "opaque injected payload")], roles())

    def test_option_order_and_refresh_count_rejected(self) -> None:
        lines = good_lines()
        lines[1], lines[2] = lines[2], lines[1]
        with self.assertRaisesRegex(CandidateMatchError, "option sequence"):
            audit_trace(lines, roles())
        bad_roles = roles()
        bad_roles[0]["expected_refreshes"] = 2
        with self.assertRaisesRegex(CandidateMatchError, "option refreshes=1 expected=2"):
            audit_trace(good_lines(), bad_roles)

    def test_crash_log_allows_only_exact_initialization_pairs(self) -> None:
        with tempfile.TemporaryDirectory(prefix="candidate-crash-") as raw:
            path = Path(raw) / "ngn_crashes.log"
            logged = Path(raw) / "original" / "ngn_crashes.log"
            path.write_text(
                "[NGN CRASH] 2026/09/06 00:00:00.000001 crash_handler.go:64: === NGN Crash Handler Initialized ===\n"
                f"[NGN CRASH] 2026/09/06 00:00:00.000002 crash_handler.go:65: Crash log: {logged}\n",
                encoding="utf-8",
            )
            self.assertEqual(crash_lines(path, logged, 1)["pairs"], 1)
            path.write_text(path.read_text() + "panic: injected\n", encoding="utf-8")
            with self.assertRaises(CandidateMatchError):
                crash_lines(path, logged, 1)


if __name__ == "__main__":
    unittest.main()
