"""Shape contract for GOMAXPROCS evidence, independent of its parser."""
import copy
import unittest

import resources


class GomaxprocsShapeTests(unittest.TestCase):
    def test_evidence_shapes(self):
        cases = [
            (2, 2), ("2", 2), ("002", 2),
            (0, None), (-2, None), ("0", None), ("-2", None),
            (True, None), (False, None), (2.0, None), (2.5, None),
            (float("nan"), None), (float("inf"), None), (float("-inf"), None),
            ("+2", None), (" 2", None), ("2\n", None), ("٢", None),
            ("２", None), ("", None), ([], None), ({}, None), (None, None),
        ]
        expected_common = {
            "requested_worker_width": 4, "effective_worker_width": 3,
            "affinity_cpu_count": 2, "affinity_evidence": "affinity-ref",
            "cpuset_cpu_count": 2, "cpuset_evidence": ["/cg/cpuset.cpus"],
            "cgroup_cpu_quota_cores": 1.5, "cgroup_cpu_quota_state": "limited",
            "cgroup_cpu_quota_evidence_complete": True,
            "cgroup_cpu_quota_unknown_levels": [],
            "cgroup_quota_evidence": ["/cg/cpu.max"],
            "capacity_is_exclusive_physical_cores": False,
            "core_scaling_qualified": False,
            "raw_evidence": {"cgroup": "cgroup-ref", "mountinfo": "mount-ref",
                             "files": {"/cg/cpu.max": "150000 100000",
                                       "/cg/cpuset.cpus": "0-1"}},
        }
        for value, expected in cases:
            with self.subTest(value=repr(value)):
                evidence = {"gomaxprocs": value, "effective_worker_width": 3,
                            "affinity": [0, 1], "affinity_reference": "affinity-ref",
                            "cgroup_reference": "cgroup-ref", "mountinfo_reference": "mount-ref",
                            "files": {"/cg/cpu.max": "150000 100000",
                                      "/cg/cpuset.cpus": "0-1"}}
                original = copy.deepcopy(evidence)
                result = resources.summarize_resources(evidence, 4)
                expected_result = dict(expected_common, subprocess_gomaxprocs=expected,
                                       unknown_capacity_fields=[] if expected is not None else
                                       ["subprocess_gomaxprocs"])
                self.assertEqual(result, expected_result)
                self.assertEqual(evidence, original)


if __name__ == "__main__":
    unittest.main()
