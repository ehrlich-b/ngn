"""Offline affinity identity validation and capacity integration fixtures."""
import copy
import unittest

import resources
import run as profile_run


class AffinityShapeTests(unittest.TestCase):
    def test_affinity_shapes_and_other_capacity_evidence(self):
        class IntSubclass(int):
            pass

        fixtures = [
            ("bool list", [True, 2], None),
            ("bool tuple", (False, 2), None),
            ("bool set", {True, 2}, None),
            ("negative list", [-1, 2], None),
            ("negative tuple", (-1, 2), None),
            ("negative set", {-1, 2}, None),
            ("mixed list", [1, "2"], None),
            ("mixed tuple", (1, "2"), None),
            ("mixed set", {1, "2"}, None),
            ("duplicate list", [2, 2], None),
            ("duplicate tuple", (2, 2), None),
            ("empty list", [], None), ("empty tuple", (), None),
            ("empty set", set(), None), ("none", None, None),
            ("integer subclass", [IntSubclass(2)], None),
            ("wrong container", frozenset({2, 9}), None),
            ("sparse list", [0, 7, 42], 3),
            ("sparse tuple", (0, 7, 42), 3),
            ("sparse set", {0, 7, 42}, 3),
        ]
        for label, affinity, expected in fixtures:
            with self.subTest(label=label):
                files = {"/cg/cpu.max": "150000 100000", "/cg/cpuset.cpus.effective": "0-3"}
                evidence = {"affinity": affinity, "affinity_reference": "fixture affinity",
                            "gomaxprocs": 2, "files": files}
                before = copy.deepcopy(evidence)
                result = resources.summarize_resources(evidence, 4)
                self.assertEqual(result["affinity_cpu_count"], expected)
                self.assertEqual("affinity_cpu_count" in result["unknown_capacity_fields"], expected is None)
                self.assertEqual(evidence, before)
                self.assertEqual(result["affinity_evidence"], "fixture affinity")
                self.assertIs(result["raw_evidence"]["files"], files)
                self.assertEqual(result["cgroup_cpu_quota_cores"], 1.5)
                self.assertEqual(result["cgroup_cpu_quota_state"], "limited")
                self.assertTrue(result["cgroup_cpu_quota_evidence_complete"])
                self.assertEqual(result["cpuset_cpu_count"], 4)
                self.assertEqual(result["cpuset_evidence"], ["/cg/cpuset.cpus.effective"])
                row = profile_run.wire_cell_resources({"gomaxprocs": 2, "effective_threads": 3},
                                                       {"resource_evidence": evidence}, 4, "/evidence/cell")
                self.assertEqual(row["measurement_classification"], "oversubscribed worker measurement")
                self.assertEqual(evidence, before)


if __name__ == "__main__":
    unittest.main()
