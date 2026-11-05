#!/usr/bin/env python3
"""Regression for auditing the experiment without a parent BRIEF.md."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]


class ContractAuditTest(unittest.TestCase):
    def test_contract_without_parent_brief(self):
        env = {k: v for k, v in os.environ.items()
               if not any(s in k.lower() for s in ("credential", "token", "secret", "password"))}
        scratch = ROOT.parent / "tmp"
        scratch.mkdir(exist_ok=True)
        parent = Path(tempfile.mkdtemp(prefix="audit-contract-", dir=scratch))
        checkout = parent / "repo"
        # Share only local Git objects; deliberately do not copy any parent files.
        subprocess.run(["git", "clone", "--shared", "--no-checkout", "--quiet",
                        str(ROOT), str(checkout)], env=env, check=True, timeout=10)
        experiment = checkout / HERE.relative_to(ROOT)
        experiment.mkdir(parents=True)
        for name in ("audit.py", "manifest.json", "candidate.patch", "positions.json",
                     "qualification.txt"):
            shutil.copyfile(HERE / name, experiment / name)
        self.assertFalse((parent / "BRIEF.md").exists())
        self.assertFalse((checkout / "BRIEF.md").exists())
        result = subprocess.run([sys.executable, str(experiment / "audit.py"), "contract"],
                                cwd=checkout, env=env, text=True, capture_output=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        report = json.loads(result.stdout)
        self.assertTrue(report["default_engine_unchanged"])
        self.assertEqual(report["corpus_positions"], 12)
        self.assertTrue(report["qualification"].startswith("pending"))


if __name__ == "__main__":
    unittest.main(verbosity=2)
