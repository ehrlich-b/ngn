"""Bounded owned-process proofs; synthetic match transcripts make no Elo claim."""
import copy
import io
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

from scripts.wave2 import net_job as nj


REPO = Path(__file__).resolve().parents[2]
FIX = REPO / "experiments/wave2-1104/net-job"


class NetJobTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="net-job-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.cpus = sorted(os.sched_getaffinity(0))[:2]
        self.source = self.root / "source.tar.gz"
        with tarfile.open(self.source, "w:gz") as archive:
            text = ('package engine\nconst DefaultNetSHA256 = "' + nj.BASE_NET +
                    '"\n//go:embed nets/default.nnue\n').encode()
            entry = tarfile.TarInfo("engine/default_net.go")
            entry.size = len(text)
            archive.addfile(entry, io.BytesIO(text))
        self.net = nj.artifact(FIX / "tiny-owned.nnue")
        self.train = {"version": 1, "id": "training", "stage": "train",
                      "inputs": {"source": nj.artifact(self.source)},
                      "argv": [sys.executable, str(FIX / "fake_training.py"),
                               self.net["path"], str(self.source), "success"]}

    def admit(self, job):
        path = self.root / "reservations.json"
        reservations = nj.read(path) if path.exists() else {"version": 1, "jobs": {}}
        reservations["jobs"][job["id"]] = {"status": "ADMITTED", "stage": job["stage"],
            "input_sha256": nj.identity(job), "cpus": self.cpus, "ticket": "owned-fixture"}
        nj.write(path, reservations)

    def completed_training(self):
        self.admit(self.train)
        receipt = nj.step(self.root, self.train)
        self.assertEqual(receipt["status"], "SUCCEEDED", receipt)
        return {"id": self.train["id"], "input_sha256": nj.identity(self.train), "export": "selected",
                "net_sha256": self.net["sha256"], "source_sha256": nj.sha(self.source)}

    def await_job(self, selection):
        return {"version": 1, "id": "consumer", "stage": "await-result", "inputs": {}, "training": selection}

    def gate_job(self):
        selection = self.completed_training()
        return {"version": 1, "id": "gate", "stage": "gate", "training": selection,
                "inputs": {"base_source": nj.artifact(self.source),
                           "base_net": nj.artifact(REPO / "engine/nets/ngn-h768-2dcb6d93.nnue")},
                "book": nj.artifact(FIX / "development-book.txt"),
                "claim_books": [nj.artifact(FIX / "claim-book.txt")], "retention_bytes": 1}

    def fake_build(self, source, output, target, log, cpus):
        shutil.copyfile(FIX / ("fake_match.py" if target == "./cmd/sprt" else "fake_uci.py"), output)
        output.chmod(0o700)
        Path(log).write_text("owned fake build; real consumer is a separate fixture\n")
        return nj.artifact(output)

    def fake_parity(self, argv, cwd, log, cpus, extra=None):
        self.assertEqual(argv[:4], ["go", "test", "-p", "1"])
        Path(log).write_text("owned fake parity; real parity is a separate fixture\n")
        return 0

    def run_gate(self, mode="keep"):
        job = self.gate_job()
        self.admit(job)
        original = nj.campaign_job

        def campaign_job(*args, **kwargs):
            value = original(*args, **kwargs)
            # Leave the A/A neutral; vary only the synthetic net transcript.
            if value["class"] == "net":
                value["candidate_options"].append("FixtureMode=" + mode)
            return value

        with patch.object(nj, "build", side_effect=self.fake_build), \
                patch.object(nj, "execute", side_effect=self.fake_parity), \
                patch.object(nj, "campaign_job", side_effect=campaign_job):
            return job, nj.step(self.root, job)

    def test_reservation_wait_then_train(self):
        self.assertEqual(nj.step(self.root, self.train)["reason"], "reservation")
        self.assertFalse((self.root / "training/train.log").exists())
        self.completed_training()

    def test_reservation_identity_is_bound(self):
        self.admit(self.train)
        path = self.root / "reservations.json"
        value = nj.read(path)
        value["jobs"]["training"]["input_sha256"] = "0" * 64
        nj.write(path, value)
        self.assertEqual(nj.step(self.root, self.train)["status"], "BLOCKED")

    def test_failed_and_missing_training_receipts_block_without_games(self):
        for mode in ("fail", "no-receipt", "failed-receipt", "wrong-source"):
            with self.subTest(mode=mode):
                job = copy.deepcopy(self.train)
                job["id"] = mode
                job["argv"][-1] = mode
                self.admit(job)
                receipt = nj.step(self.root, job)
                self.assertEqual(receipt["status"], "FAILED", receipt)
                selection = {"id": mode, "input_sha256": nj.identity(job), "export": "selected",
                             "net_sha256": self.net["sha256"], "source_sha256": nj.sha(self.source)}
                waiter = self.await_job(selection)
                waiter["id"] = "await-" + mode
                self.assertEqual(nj.step(self.root, waiter)["status"], "BLOCKED")
        selection["id"] = "absent"
        waiter = self.await_job(selection)
        self.assertEqual(nj.step(self.root, waiter)["status"], "BLOCKED")
        self.assertFalse(list(self.root.rglob("games.pgn")))

    def test_waits_for_explicit_terminal_success(self):
        selection = self.completed_training()
        path = self.root / "training/receipt.json"
        terminal = nj.read(path)
        for status in ("WAITING", "RUNNING"):
            nj.write(path, dict(terminal, status=status))
            self.assertEqual(nj.step(self.root, self.await_job(selection))["status"], "WAITING")
        nj.write(path, terminal)
        self.assertEqual(nj.step(self.root, self.await_job(selection))["status"], "READY")

    def test_selected_net_source_and_receipt_hashes(self):
        selection = self.completed_training()
        for field in ("net_sha256", "source_sha256", "input_sha256"):
            with self.subTest(field=field):
                wrong = dict(selection, **{field: "0" * 64})
                waiter = self.await_job(wrong)
                waiter["id"] = field
                self.assertEqual(nj.step(self.root, waiter)["status"], "BLOCKED")
        self.source.write_bytes(b"mutated source")
        self.assertEqual(nj.step(self.root, self.await_job(selection))["status"], "BLOCKED")

    def test_duplicate_start_is_idempotent_and_rejects_changed_inputs(self):
        self.completed_training()
        path = self.root / "training/train.log"
        before = path.stat().st_mtime_ns
        with patch.object(nj, "execute", side_effect=AssertionError("restarted")):
            first = nj.step(self.root, self.train)
            self.assertEqual(nj.step(self.root, self.train), first)
        self.assertEqual(path.stat().st_mtime_ns, before)
        changed = copy.deepcopy(self.train)
        changed["argv"][-1] = "fail"
        with self.assertRaises(nj.Invalid):
            nj.step(self.root, changed)

    def test_running_work_is_never_restarted(self):
        self.admit(self.train)
        nj.write(self.root / "training/receipt.json", {"status": "RUNNING"})
        with patch.object(nj, "execute", side_effect=AssertionError("restarted")):
            self.assertEqual(nj.step(self.root, self.train)["status"], "BLOCKED")

    def test_training_and_gate_locks_are_separate(self):
        self.admit(self.train)
        with nj.lock(self.root / "training.lock") as acquired:
            self.assertTrue(acquired)
            self.assertEqual(nj.step(self.root, self.train)["reason"], "training.lock")
            with nj.lock(self.root / "gate.lock") as other:
                self.assertTrue(other)
        job = self.gate_job()
        self.admit(job)
        with nj.lock(self.root / "gate.lock"):
            self.assertEqual(nj.step(self.root, job)["reason"], "gate.lock")
        with nj.lock(self.root / "gate/job.lock"):
            with patch.object(nj, "gate", side_effect=AssertionError("duplicate start")):
                self.assertEqual(nj.step(self.root, job)["reason"], "gate.lock")

    def test_failed_prerequisite_blocks_training(self):
        path = self.root / "prerequisite.json"
        nj.write(path, {"status": "FAILED"})
        self.train["prerequisite"] = nj.artifact(path)
        self.admit(self.train)
        self.assertEqual(nj.step(self.root, self.train)["status"], "BLOCKED")

    def test_book_overlap_and_source_mismatch_are_rejected(self):
        book = nj.artifact(FIX / "development-book.txt")
        with self.assertRaises(nj.Invalid):
            nj.frozen_book(book, [book], self.root / "book.txt")
        a, b = self.root / "a", self.root / "b"
        a.mkdir()
        b.mkdir()
        (a / "main.go").write_text("a")
        (b / "main.go").write_text("b")
        with self.assertRaises(nj.Invalid):
            nj.matching_sources(a, b, None)

    def test_control_changes_with_new_source_or_net(self):
        job = self.gate_job()
        selected = {"net": self.net, "source": nj.artifact(self.source)}
        binary = nj.artifact(FIX / "fake_uci.py")
        args = (job, selected, job["book"], {"base": binary}, self.cpus)
        original = nj.control_identity(*args)
        for kind in ("source", "net"):
            changed = copy.deepcopy(selected)
            changed[kind]["sha256"] = "0" * 64
            self.assertNotEqual(original, nj.control_identity(job, changed, *args[2:]))

    def test_keep_writes_private_candidate_and_matching_embedded_file(self):
        job, receipt = self.run_gate()
        self.assertEqual(receipt["status"], "SUCCEEDED", receipt)
        self.assertEqual(receipt["verdict"], "KEEP")
        adoption = self.root / "gate/adoption"
        manifest = nj.read(adoption / "manifest.json")
        nj.check_artifact(receipt["adoption"])
        nj.check_artifact(manifest["bundle"])
        self.assertEqual(manifest["status"], "READY")
        self.assertTrue(manifest["private"])
        self.assertFalse(manifest["main_updated"])
        self.assertEqual(nj.read(adoption / "identity.json")["cases"], 4)
        embedded = adoption / "source/engine/nets" / ("candidate-" + self.net["sha256"] + ".nnue")
        self.assertEqual(nj.sha(embedded), self.net["sha256"])
        self.assertEqual(manifest["retained_bytes"], nj.retained_bytes(self.root / "gate"))
        self.assertGreater(manifest["retained_bytes"], job["retention_bytes"])
        self.assertIn("preserved", manifest["keep_reason"])
        with patch.object(nj, "gate", side_effect=AssertionError("restarted gate")):
            self.assertEqual(nj.step(self.root, job), receipt)

    def test_negative_gate_is_shelved(self):
        _, receipt = self.run_gate("negative")
        self.assertEqual(receipt["status"], "SUCCEEDED", receipt)
        self.assertEqual(receipt["verdict"], "SHELVED")
        self.assertFalse((self.root / "gate/adoption").exists())

    def test_rounded_zero_gate_is_shelved(self):
        _, receipt = self.run_gate("rounded-zero")
        self.assertEqual(receipt["verdict"], "SHELVED", receipt)
        self.assertFalse((self.root / "gate/adoption").exists())

    def test_incomplete_pair_blocks_adoption(self):
        _, receipt = self.run_gate("incomplete")
        self.assertEqual(receipt["status"], "FAILED", receipt)
        self.assertFalse((self.root / "gate/adoption").exists())

    def test_fault_blocks_adoption(self):
        _, receipt = self.run_gate("fault")
        self.assertEqual(receipt["status"], "FAILED", receipt)
        self.assertFalse((self.root / "gate/adoption").exists())

    def test_same_color_pair_blocks_adoption(self):
        _, receipt = self.run_gate("same-color")
        self.assertEqual(receipt["status"], "FAILED", receipt)

    def test_owned_uci_consumer_identity(self):
        result = nj.consumer_identity(FIX / "fake_uci.py", self.net["path"], self.cpus)
        self.assertEqual((result["cases"], result["depth"], result["identical"]), (4, 4, True))


class StrictCampaignTests(unittest.TestCase):
    """The real cmd/sprt and campaign caller must reject a recovered search fault."""
    @classmethod
    def setUpClass(cls):
        cls.temporary = tempfile.TemporaryDirectory(prefix="net-job-strict-")
        cls.addClassCleanup(cls.temporary.cleanup)
        cls.root = Path(cls.temporary.name)
        cls.harness = cls.root / "sprt"
        subprocess.run(["nice", "-n", "10", "go", "build", "-p", "1", "-o", str(cls.harness), "./cmd/sprt"],
                       cwd=REPO, env=nj.environment(), check=True)

    def test_recovered_search_error_cannot_pass_aa_or_net(self):
        for mode in ("warmup", "search"):
            for kind in ("aa", "net"):
                with self.subTest(mode=mode, kind=kind):
                    directory = self.root / (mode + "-" + kind)
                    directory.mkdir()
                    registry, status = directory / "controls.json", directory / "status.jsonl"
                    if kind == "net":
                        nj.write(registry, {"fixture": {"result": "owned prerequisite fixture"}})
                    job = {"name": mode + "-" + kind, "class": kind, "output": str(directory / "match"),
                           "control_key": "fixture", "artifacts": {str(self.harness): nj.sha(self.harness),
                               str(FIX / "fake_uci.py"): nj.sha(FIX / "fake_uci.py")},
                           "harness": str(self.harness), "candidate": str(FIX / "fake_uci.py"),
                           "base": str(FIX / "fake_uci.py"), "cpus": str(min(os.sched_getaffinity(0))),
                           "tc": "10+0.1", "concurrency": 1, "min_games": 200, "max_games": 200,
                           "book": str(FIX / "development-book.txt"), "candidate_options": ["FixtureError=" + mode],
                           "base_options": [], "no_retries": True, "strict_pairs": True}
                    result = nj.campaign.run(job, registry, status)
                    self.assertEqual(result["verdict"], "UNSCORED", result)
                    self.assertTrue(result["faults"])
                    self.assertNotEqual(result["harness_exit"], 0)
                    self.assertIn("harness exit" + str(result["harness_exit"]), result["faults"])
                    log = (directory / "match/match.log").read_text()
                    self.assertIn("warmup" if mode == "warmup" else "engine-error", log)
                    self.assertNotIn("=== RESULT", log)
                    if kind == "aa":
                        self.assertFalse(registry.exists(), "failed control was registered")


if __name__ == "__main__":
    unittest.main()
