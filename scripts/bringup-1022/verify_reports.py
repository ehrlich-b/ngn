#!/usr/bin/env python3
"""Check the bringup reports against their retained inputs without running a rig."""
import ast
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
EVIDENCE = ROOT.parent / "evidence"


def read_json(path):
    return json.loads(path.read_text())


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def report(name, limited=False):
    text = (ROOT / name).read_text()
    if limited:
        assert len(text.split()) <= 500, (name, len(text.split()))
    for path in re.findall(r"\]\(([^)]+)\)", text):
        assert (ROOT / path.split('#')[0]).is_file(), path
    return text


def baseline():
    text = report("BASELINE.md", limited=True)
    data = read_json(EVIDENCE / "measurement-summary.json")
    assert data["backend"] == "hce"
    assert data["binary_sha256"] == sha(EVIDENCE / "bin/ngn")
    assert data["binary_sha256"] in text
    for name in ("build", "test", "vet"):
        status = int((EVIDENCE / (name + ".exit")).read_text())
        log = EVIDENCE / (name + ".log")
        assert log.is_file()
        assert f"{name} exit {status}" in text, (name, status)
        print(name, "exit", status, "log_sha256", sha(log))
    assert [x["threads"] for x in data["legacy"]] == [1, 2, 4, 8]
    for row in data["legacy"]:
        n = row["threads"]
        assert row["positions"] == row["successful"] == 24
        assert str(row["avg_nps"]) in text
        raw = read_json(EVIDENCE / f"legacy-threads{n}.json")
        assert raw["avg_nps"] == row["avg_nps"]
        assert all(r["success"] for r in raw["results"])
        assert row["exit"] == (1 if "performance regression detected" in (EVIDENCE / f"legacy-threads{n}.log").read_text() else 0)
    assert all(row["exit"] == 0 for row in data["snapshots"] + data["comparisons"])
    digests = {sha(EVIDENCE / f"snapshot-threads{n}.json") for n in [1, 2, 4, 8]}
    assert len(digests) == 1
    snapshot = read_json(EVIDENCE / "snapshot-threads1.json")
    assert len(snapshot["fixtures"]) == 11
    rows = read_json(EVIDENCE / "fixed-results-verified.json")
    assert len(rows) == 24
    base = {x["name"]: x for x in rows if x["threads"] == 1}
    for row in rows:
        assert row["exit"] == 0 and row["nodes"] == 400000
        assert row["effective_threads"] == 1
        for key in ["depth", "bestmove", "last_completed_score"]:
            assert row[key] == base[row["name"]][key]
        assert row["bestmove"] in text
        raw = (EVIDENCE / f"fixed-{row['threads']}-{row['name']}.uci.log").read_text()
        receipts = re.findall(r"< info string threads configured (\d+) effective (\d+)", raw)
        assert receipts[-1] == (str(row["threads"]), "1")
        assert "nodes 400000 " in raw
    print("A1: 96 timed positions, 24 exact node caps, 44 cold/warm fixtures checked")


def state():
    text = report("STATE.md", limited=True)
    h1 = read_json(ROOT / "experiments/2026-09-19-h1-history-policy/result-v1.json")
    assert h1["decision"] == "ACCEPT" and h1["accepted_policy"] == "11"
    assert round(h1["phases"]["confirmation_11_vs_00"]["elo"], 2) == 56.96
    admission = read_json(ROOT / "experiments/2026-09-19-p1-staged-picker-profile/result-v1.json")
    assert admission["decision"] == "ADMIT_P1_REFERENCE_STAGED_PICKER"
    for token in ["56.96", "6.91%", "No implementation", "3300", "3600", "s39–s46", "not an official"]:
        assert token in text, token
    assert "67150c8e7811a2ca76cb94250d17f6503a6e248f" == subprocess.check_output(["git", "rev-parse", "HEAD:engine"], cwd=ROOT, text=True).strip()
    print("A2: word cap, all local citations, H1 verdict, P1 admission and unchanged engine tree checked")


def rig():
    text = report("RIG-PLAN.md")
    blocks = re.findall(r"```bash\n(.*?)```", text, re.S)
    assert len(blocks) == 3
    for block in blocks:
        subprocess.run(["bash", "-n"], input=block, text=True, check=True)
    python = re.search(r"<<'PY'\n(.*?)\nPY\n", blocks[-1], re.S)[1]
    ast.parse(python)
    held = read_json(ROOT / "experiments/2026-09-19-h1-history-policy/manifest-release-v1.json")
    driver = ROOT / "experiments/2026-09-19-e0-v12-anand-strength/driver.py"
    assert sha(driver) == held["frozen_sources"]["e0_driver.py"]["sha256"]
    literal_names = ast.literal_eval(re.search(r"names = (\[.*?\])\n", python, re.S)[1])
    assert set(literal_names) <= set(held["frozen_sources"])
    assert held["frozen_sources"]["v12_model"]["sha256"] in text
    for token in ["P1_CANDIDATE:?", "Blocked on implementation", "10+0.1", "0,2,4,6", "628", "100,000", "lower95 Elo >0", "e0.supervise", "e0.audit", "e0.preflight", "<=0.97", "<=1.05", "1.15"]:
        assert token in text, token
    print("A3: all shell/Python blocks parse; frozen driver and input names match; one conditional candidate and full protocol checked; no rig contacted")


def queue():
    rows = read_json(ROOT / "QUEUE-ITEMS.json")
    assert 1 <= len(rows) <= 4
    keys = set("id product title why acceptance gates value cost lane_class".split())
    assert len({x["id"] for x in rows}) == len(rows)
    for row in rows:
        assert set(row) == keys and row["product"] == "ngn"
        assert type(row["cost"]) is int and 1 <= row["cost"] <= 3
        assert row["lane_class"] in ["light", "heavy"]
        assert all(row[k] for k in ["id", "title", "why", "acceptance", "value"])
        assert isinstance(row["gates"], list) and len(row["gates"]) >= 2
        for command in row["gates"]:
            assert "nice -n 10" in command and "GOMAXPROCS=8" in command
            subprocess.run(["bash", "-n", "-c", command], check=True)
    assert rows[0]["id"] == "ngn-p1-r1"
    print("A4:", len(rows), "ranked items; exact key sets, costs, thread limits and complete gate syntax checked")


if __name__ == "__main__":
    {"A1": baseline, "A2": state, "A3": rig, "A4": queue}[sys.argv[1]]()
