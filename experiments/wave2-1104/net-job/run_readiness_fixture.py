#!/usr/bin/env python3
"""Reproduce readiness/search rejection with real harness and owned transcripts."""
import argparse
import json
import os
from pathlib import Path
import sys
import unittest

REPO = Path(__file__).resolve().parents[3]
FIX = Path(__file__).resolve().parent
sys.path.insert(0, str(REPO))
from scripts.wave2 import net_job as nj  # noqa: E402
from scripts.wave2.test_net_job import StrictCampaignTests  # noqa: E402


def run(output, record=False):
    output = output.resolve()
    if output == REPO or REPO in output.parents:
        raise ValueError("evidence must be outside the checkout")
    output.mkdir(parents=True, exist_ok=True)
    os.environ["NET_JOB_PROOF_OUTPUT"] = str(output)
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(StrictCampaignTests)
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    if not result.wasSuccessful():
        raise SystemExit(1)
    root = StrictCampaignTests.root
    cases = []
    for path in sorted(root.glob("*/match/result.json")):
        value = nj.read(path)
        log = (path.parent / "match.log").read_text()
        kind = "aa" if path.parents[1].name.endswith("-aa") else "net"
        count = 200 if kind == "aa" else 1000
        cases.append({"name": path.parents[1].name, "harness_exit": value["harness_exit"],
                      "verdict": value["verdict"], "fault_count": len(value["faults"]),
                      "games": value.get("games", 0), "pairs": value.get("pairs", 0),
                      "readiness_rejection_logged": "engine rejected configuration:" in log,
                      "strict_pass": nj.strict_result(value, count, "AA_PASS" if kind == "aa" else "KEEP")})
    assert len(cases) == 8
    assert all(c["verdict"] == "UNSCORED" and c["fault_count"] and not c["strict_pass"] for c in cases)
    assert not list(root.glob("*-aa/controls.json")), "failed A/A was registered"
    assert not list(root.glob("*/adoption")), "faulty gate produced an adoption"
    report = {"version": 1, "fixture_only": True, "elo_claim": False,
              "passing_controls_registered": 0, "adoptions": 0, "cases": cases}
    nj.write(root / "readiness-result.json", report)
    expected = FIX / "readiness-result.json"
    if record:
        nj.write(expected, report)
    elif report != nj.read(expected):
        raise nj.Invalid("readiness regression differs from committed result: " + str(root))
    print(json.dumps({"status": "pass", "cases": len(cases), "evidence": str(root)}))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--record", action="store_true")
    args = parser.parse_args()
    run(args.output, args.record)
