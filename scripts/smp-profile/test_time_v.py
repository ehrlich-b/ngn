"""Bounded real-child tests for native timing and exception cleanup."""
import contextlib
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parent))
import time_v


class ChildCleanupTests(unittest.TestCase):
    def setUp(self):
        self.children = []
        self.stack = contextlib.ExitStack()
        self.addCleanup(self.stack.close)
        self.addCleanup(self.cleanup_children)
        self.root = Path(self.stack.enter_context(tempfile.TemporaryDirectory()))
        self.folder = self.root / "execution"
        self.env = time_v.environment()
        self.env["HOME"] = str(self.root)
        self.env["TEST_MARKER"] = "child-environment"
        real_popen = subprocess.Popen

        def spawn(*args, **kwargs):
            child = real_popen(*args, **kwargs)
            self.children.append(child)
            print(json.dumps({"test": self.id(), "spawned_pid": child.pid}), flush=True)
            return child

        self.stack.enter_context(mock.patch.object(time_v.subprocess, "Popen", spawn))
        old_handler = signal.signal(signal.SIGALRM, self.timeout)
        self.stack.callback(signal.signal, signal.SIGALRM, old_handler)
        signal.setitimer(signal.ITIMER_REAL, 8)
        self.stack.callback(signal.setitimer, signal.ITIMER_REAL, 0)

    @staticmethod
    def timeout(*_):
        raise TimeoutError("real-child regression exceeded eight seconds")

    def cleanup_children(self):
        # Independent of Popen.poll/wait and the runner's cleanup implementation.
        for child in self.children:
            try:
                os.kill(child.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            try:
                _, status = os.waitpid(child.pid, 0)
                child.returncode = os.waitstatus_to_exitcode(status)
                print(json.dumps({"test": self.id(), "finally_reaped_pid": child.pid,
                                  "returncode": child.returncode}), flush=True)
            except ChildProcessError:
                pass

    def assert_reaped(self):
        self.assertEqual(len(self.children), 1)
        child = self.children[0]
        try:
            os.kill(child.pid, 0)
            alive = True  # Includes zombies; both violate ownership after return.
        except ProcessLookupError:
            alive = False
        try:
            pid, status = os.waitpid(child.pid, os.WNOHANG)
            waitable = True
            if pid:
                child.returncode = os.waitstatus_to_exitcode(status)
        except ChildProcessError:
            waitable = False
        print(json.dumps({"test": self.id(), "pid": child.pid, "alive": alive,
                          "waitable": waitable, "returncode": child.returncode}), flush=True)
        self.assertFalse(alive, f"owned child {child.pid} remains alive/unreaped")
        self.assertFalse(waitable, f"owned child {child.pid} remains waitable")
        self.assertIsNotNone(child.returncode)

    def run_child(self, code="import time; time.sleep(30)"):
        return time_v.run([sys.executable, "-c", code], self.folder,
                          env=self.env, cwd=self.root)

    def expect_failure(self, operation, failure, callback=None):
        def fail(*args):
            if callback:
                callback(*args)
            raise failure

        with mock.patch.object(time_v, operation, side_effect=fail):
            with self.assertRaises(type(failure)) as caught:
                self.run_child()
        print(json.dumps({"test": self.id(), "exception": repr(caught.exception)}), flush=True)
        self.assertIs(caught.exception, failure)
        self.assert_reaped()
        self.assertFalse((self.folder / "time.json").exists())

    def test_capture_failure_after_spawn(self):
        self.expect_failure("capture_resource_evidence", RuntimeError("injected capture failure"))

    def test_capture_result_failure_after_spawn(self):
        with mock.patch.object(time_v, "capture_resource_evidence", return_value=None):
            with self.assertRaises(TypeError):
                self.run_child()
        self.assert_reaped()
        self.assertFalse((self.folder / "time.json").exists())

    def test_telemetry_failure(self):
        self.expect_failure("telemetry", RuntimeError("injected telemetry failure"))

    def test_keyboard_interrupt(self):
        self.expect_failure("telemetry", KeyboardInterrupt("injected interrupt"))

    def test_wait_interrupted(self):
        failure = KeyboardInterrupt("injected wait interrupt")
        real_wait4 = os.wait4
        calls = []

        def wait4(pid, options):
            calls.append((pid, options))
            if len(calls) == 1:
                raise failure
            return real_wait4(pid, options)

        with mock.patch.object(time_v.os, "wait4", side_effect=wait4):
            with self.assertRaises(KeyboardInterrupt) as caught:
                self.run_child()
        self.assertIs(caught.exception, failure)
        self.assert_reaped()

    def test_already_exited_child(self):
        failure = RuntimeError("capture failed after child exit")

        def after_exit(pid):
            deadline = time.monotonic() + 4
            while time.monotonic() < deadline:
                # Observe exit without reaping, leaving cleanup the actual race.
                if Path(f"/proc/{pid}/stat").read_text().split(") ", 1)[1].startswith("Z"):
                    print(json.dumps({"test": self.id(), "exited_pid": pid}), flush=True)
                    raise failure
                time.sleep(0.01)
            self.fail("short child did not exit")

        with mock.patch.object(time_v, "capture_resource_evidence", side_effect=after_exit):
            with self.assertRaises(RuntimeError) as caught:
                self.run_child("raise SystemExit(7)")
        self.assertIs(caught.exception, failure)
        self.assert_reaped()
        self.assertEqual(self.children[0].returncode, 7)

    def test_success_receipt_and_io(self):
        code = ("import json, os, sys; "
                "sum(i*i for i in range(200000)); "
                "print(json.dumps([os.getcwd(), os.environ['TEST_MARKER'], "
                "os.getpriority(os.PRIO_PROCESS, 0)])); "
                "print('child stderr', file=sys.stderr)")
        receipt = self.run_child(code)
        self.assert_reaped()
        self.assertEqual(self.children[0].returncode, 0)
        self.assertEqual(receipt["exit_code"], 0)
        self.assertEqual(receipt["command"], [sys.executable, "-c", code])
        cwd, marker, priority = json.loads((self.folder / "stdout.txt").read_text())
        self.assertEqual(cwd, str(self.root))
        self.assertEqual(marker, "child-environment")
        self.assertEqual(priority, min(19, os.getpriority(os.PRIO_PROCESS, 0) + 10))
        self.assertEqual((self.folder / "stderr.txt").read_text(), "child stderr\n")
        self.assertEqual(receipt["resource_evidence"]["gomaxprocs"], "2")
        self.assertGreater(receipt["user_seconds"] + receipt["system_seconds"], 0)
        self.assertGreater(receipt["peak_rss_kib"], 0)
        self.assertGreaterEqual(receipt["elapsed_seconds"], 0)
        self.assertEqual(receipt["measurement_interval_seconds"], receipt["elapsed_seconds"])
        self.assertEqual(json.loads((self.folder / "time.json").read_text()), receipt)
        self.assertTrue(json.loads((self.folder / "telemetry.json").read_text()))
        self.assertEqual(set(receipt), {
            "command", "exit_code", "elapsed_seconds", "user_seconds", "system_seconds",
            "peak_rss_kib", "voluntary_context_switches", "involuntary_context_switches",
            "measurement_interval_seconds", "resource_evidence", "host_busy_cpu_seconds",
            "non_owned_busy_cpu_seconds_estimate"})

    def test_nonzero_exit(self):
        with self.assertRaisesRegex(RuntimeError, "exit 7:"):
            self.run_child("raise SystemExit(7)")
        self.assert_reaped()
        self.assertEqual(self.children[0].returncode, 7)
        receipt = json.loads((self.folder / "time.json").read_text())
        self.assertEqual(receipt["exit_code"], 7)
        self.assertGreater(receipt["peak_rss_kib"], 0)

    def test_unavailable_resources_remain_evidence(self):
        evidence = {"files": {"/unavailable/cpu.max": {"unavailable": "denied"}}}
        with mock.patch.object(time_v, "capture_resource_evidence", return_value=evidence):
            receipt = self.run_child("pass")
        self.assert_reaped()
        self.assertEqual(receipt["resource_evidence"]["files"], evidence["files"])

    def test_unavailable_telemetry_remains_evidence(self):
        real_telemetry = time_v.telemetry

        def unavailable(pid):
            sample = real_telemetry(pid)
            sample[f"/proc/{pid}/status"] = {"unavailable": "denied"}
            return sample

        with mock.patch.object(time_v, "telemetry", side_effect=unavailable):
            self.run_child("pass")
        self.assert_reaped()
        samples = json.loads((self.folder / "telemetry.json").read_text())
        for sample in samples:
            self.assertEqual(sample[f"/proc/{self.children[0].pid}/status"],
                             {"unavailable": "denied"})

    def test_receipt_failure_after_native_reap(self):
        failure = RuntimeError("injected receipt failure")
        with mock.patch.object(time_v, "external_busy_cpu_estimate", side_effect=failure):
            with self.assertRaises(RuntimeError) as caught:
                self.run_child("pass")
        self.assertIs(caught.exception, failure)
        self.assert_reaped()
        self.assertEqual(self.children[0].returncode, 0)
        self.assertFalse((self.folder / "time.json").exists())


if __name__ == "__main__":
    unittest.main()
