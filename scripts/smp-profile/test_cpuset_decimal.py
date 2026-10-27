"""Deterministic kernel cpuset token and summary regression tests."""
import unittest

import resources


class CpusetDecimalTests(unittest.TestCase):
    def test_malformed_tokens_are_unavailable(self):
        for value in ("+1", "١", "１", "0-+1", "+0-1", "0-١", "١-2",
                      "0- 1", "0 -1", "0-1 ", "0, 1", "-1", "1--2"):
            # Outer whitespace remains supported; exercise whitespace inside ranges.
            if value == "0-1 ":
                continue
            with self.subTest(value=value):
                self.assertIsNone(resources._count_cpus(value))

    def test_valid_and_existing_unavailable_cases(self):
        for value, expected in (("0", 1), ("0-3,2", 4), ("0-3,2-5", 6),
                                ("0-3,2\n", 4), ("", None),
                                ({"unavailable": "denied"}, None),
                                ("3-1", None), ("0-1000001", None)):
            with self.subTest(value=value):
                self.assertEqual(resources._count_cpus(value), expected)

    def test_summary_keeps_malformed_cpuset_unknown(self):
        for value in ("+1", "١", "0-+1", "0- 1"):
            with self.subTest(value=value):
                path = "/cg/job/cpuset.cpus.effective"
                result = resources.summarize_resources({"files": {path: value}}, 4)
                self.assertIsNone(result["cpuset_cpu_count"])
                self.assertIn("cpuset_cpu_count", result["unknown_capacity_fields"])
                self.assertEqual(result["raw_evidence"]["files"][path], value)


if __name__ == "__main__":
    unittest.main()
