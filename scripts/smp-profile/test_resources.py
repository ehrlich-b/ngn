"""Offline resource reporting and mocked runner contract tests."""
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import resources
import run as profile_run


class ResourceSummaryTests(unittest.TestCase):
    def test_quota_limited_and_unrestricted_capacity(self):
        limited = {"gomaxprocs": "2", "affinity": [0, 1, 2, 3], "files": {
            "/cg/job/cpu.max": "150000 100000\n", "/cg/job/cpuset.cpus.effective": "0-3\n"}}
        result = resources.summarize_resources(limited, 4)
        self.assertEqual(result["cgroup_cpu_quota_cores"], 1.5)
        self.assertEqual(result["affinity_cpu_count"], 4)
        self.assertEqual(result["subprocess_gomaxprocs"], 2)
        self.assertFalse(result["core_scaling_qualified"])
        unlimited = {"gomaxprocs": 2, "affinity": [0, 1], "files": {
            "/cg/job/cpu.max": "max 100000", "/cg/job/cpuset.cpus.effective": "0-1"}}
        result = resources.summarize_resources(unlimited, 1)
        self.assertIsNone(result["cgroup_cpu_quota_cores"])
        self.assertEqual(result["cpuset_cpu_count"], 2)

    def test_cpuset_ranges_and_width_runtime_mismatch(self):
        evidence = {"gomaxprocs": 2, "effective_worker_width": 3, "affinity": [4, 5, 6], "files": {
            "/cg/team/cpuset.cpus.effective": "4-5,7"}}
        summary = resources.summarize_resources(evidence, requested_width=8)
        self.assertEqual(summary["cpuset_cpu_count"], 3)
        self.assertEqual(summary["effective_worker_width"], 3)
        row = profile_run.wire_cell_resources({"gomaxprocs": 2, "effective_threads": 3},
                                               {"resource_evidence": evidence}, 8, "/evidence/cell")
        self.assertEqual(row["measurement_classification"], "oversubscribed worker measurement")

    def test_v2_mount_root_and_inherited_limits(self):
        cgroup = "0::/tenant.slice/workload.scope\n"
        mountinfo = "36 25 0:32 /tenant.slice /sys/fs/cgroup/unified rw - cgroup2 cgroup rw\n"
        mounts = list(resources._mounts(mountinfo, cgroup))
        self.assertEqual(mounts, [("/sys/fs/cgroup/unified", "/tenant.slice",
                                  "/tenant.slice/workload.scope", 2, set())])
        evidence = {"cgroup": cgroup, "mountinfo": mountinfo, "affinity": [0, 1, 2, 3],
                    "files": {"/cg/tenant/cpu.max": "max 100000",
                              "/cg/tenant.slice/cpu.max": "300000 100000",
                              "/cg/tenant.slice/workload.scope/cpu.max": "200000 100000",
                              "/cg/tenant.slice/workload.scope/cpuset.cpus.effective": "0-3"}}
        summary = resources.summarize_resources(evidence, 8)
        self.assertEqual(summary["cgroup_cpu_quota_cores"], 2)
        self.assertIn("/cg/tenant.slice/cpu.max", summary["cgroup_quota_evidence"])

    def test_v1_nontrivial_mount_and_inherited_quota(self):
        cgroup = "5:cpu,cpuacct:/docker/abc/job\n7:cpuset:/docker/abc/job\n"
        mountinfo = ("31 20 0:29 /docker/abc /sys/fs/cgroup/cpu rw - cgroup cgroup rw,cpu,cpuacct\n"
                     "32 20 0:30 /docker/abc /sys/fs/cgroup/cpuset rw - cgroup cgroup rw,cpuset\n")
        mounts = list(resources._mounts(mountinfo, cgroup))
        self.assertEqual(len(mounts), 2)
        self.assertEqual(mounts[0][2], "/docker/abc/job")
        files = {"/cg/cpu/cpu.cfs_quota_us": "-1", "/cg/cpu/cpu.cfs_period_us": "100000",
                 "/cg/cpu/job/cpu.cfs_quota_us": "250000", "/cg/cpu/job/cpu.cfs_period_us": "100000",
                 "/cg/cpuset/job/cpuset.cpus": "2-5"}
        summary = resources.summarize_resources({"affinity": [2, 3, 4, 5], "files": files}, 8)
        self.assertEqual(summary["cgroup_cpu_quota_cores"], 2.5)
        self.assertEqual(summary["cpuset_cpu_count"], 4)

    def test_unavailable_malformed_remain_unknown_and_never_qualify(self):
        summary = resources.summarize_resources({"gomaxprocs": "bad", "affinity": None,
                                                  "files": {"/cg/cpu.max": "nonsense",
                                                            "/cg/cpuset.cpus": {"unavailable": "denied"}}}, 8)
        self.assertIsNone(summary["subprocess_gomaxprocs"])
        self.assertIsNone(summary["affinity_cpu_count"])
        self.assertIsNone(summary["cpuset_cpu_count"])
        self.assertIsNone(summary["cgroup_cpu_quota_cores"])
        self.assertIn("cgroup_cpu_quota_cores", summary["unknown_capacity_fields"])
        self.assertFalse(summary["core_scaling_qualified"])

    def test_external_load_arithmetic(self):
        self.assertEqual(resources.external_busy_cpu_estimate(8.25, 3.0), 5.25)
        self.assertEqual(resources.external_busy_cpu_estimate(2, 4), 0.0)
        self.assertIsNone(resources.external_busy_cpu_estimate(None, 1))

    def test_report_resource_wiring_for_normal_and_blocked_reports(self):
        evidence = {"gomaxprocs": 2, "affinity": [0, 1], "files": {"/cg/cpu.max": "max 100000"}}
        for status in ("MEASURED_CLOSE_NO_POLICY", "BLOCKED"):
            report = profile_run.wire_report_resources({"status": status}, evidence)
            self.assertEqual(report["resources"]["subprocess_gomaxprocs"], 2)
            self.assertFalse(report["resources"]["core_scaling_qualified"])


class RunnerReceiptTests(unittest.TestCase):
    def test_missing_and_wrong_model_rejected_before_search(self):
        with self.assertRaisesRegex(ValueError, "unavailable"):
            profile_run.validate_model(None)
        with tempfile.TemporaryDirectory() as tmp:
            wrong = Path(tmp) / "wrong.bin"
            wrong.write_bytes(b"not a model")
            with self.assertRaisesRegex(ValueError, "identity mismatch"):
                profile_run.validate_model(wrong)

    def test_main_blocks_before_search_when_model_missing(self):
        with tempfile.TemporaryDirectory() as tmp:
            wrong = Path(tmp) / "wrong.bin"
            wrong.write_bytes(b"not a model")
            for label, model in (("missing", None), ("wrong", wrong)):
                with self.subTest(model=label):
                    output = Path(tmp) / (label + "-evidence")
                    argv = ["run.py", "--output", str(output)]
                    if model is not None:
                        argv += ["--model", str(model)]
                    with mock.patch.object(sys, "argv", argv), \
                         mock.patch.object(profile_run, "git", side_effect=lambda *args: "" if args[0] == "diff" else profile_run.BASE), \
                         mock.patch.object(profile_run, "capture_resource_evidence", return_value={"files": {}}), \
                         mock.patch.object(profile_run, "run", side_effect=AssertionError("search/build must not run")):
                        self.assertEqual(profile_run.main(), 1)
                    report_files = list(output.glob("run-*/report.json"))
                    self.assertEqual(len(report_files), 1)
                    report = json.loads(report_files[0].read_text())
                    self.assertEqual(report["status"], "BLOCKED")

    def test_no_tt_samples_annotation_and_corrupt_profile_receipt(self):
        with tempfile.TemporaryDirectory() as tmp:
            folder = Path(tmp)
            with mock.patch.object(profile_run, "run", side_effect=RuntimeError("pprof failed")):
                (folder / "time.json").write_text(json.dumps({"exit_code": 2}))
                (folder / "stderr.txt").write_text("no matches found for regexp: Cache.*(Get|Set)\n")
                self.assertEqual(profile_run.tt_annotation(Path("engine.test"), Path("cpu.pprof"), folder),
                                 "NO_TT_SAMPLES")
                (folder / "time.json").write_text(json.dumps({"exit_code": 1}))
                with self.assertRaisesRegex(RuntimeError, "pprof failed"):
                    profile_run.tt_annotation(Path("engine.test"), Path("corrupt.pprof"), folder)


if __name__ == "__main__":
    unittest.main()
