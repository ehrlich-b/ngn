#!/usr/bin/env python3
"""Check the committed prior-bonus experiment handoff; never run the pre-screen."""
import json
from pathlib import Path
import re
import subprocess
import sys
import shlex
import tempfile

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
BASE = "6d3168b8c5f15532cc7a352d05465a77ab50f361"


def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT, text=True).strip()


def run(args, cwd=ROOT):
    print("+", shlex.join(map(str, args)), flush=True)
    subprocess.run(list(map(str, args)), cwd=cwd, check=True)


def identity():
    output = ROOT.parent / "evidence"
    builds = ROOT.parent / "artifacts"
    output.mkdir(exist_ok=True)
    builds.mkdir(exist_ok=True)
    source = Path(tempfile.mkdtemp(prefix="prior-baseline-", dir=ROOT.parent / "tmp"))
    archive = source / "baseline.tar"
    run(["git", "archive", "--output=" + str(archive), BASE])
    run(["tar", "-xf", archive, "-C", source])
    run(["nice", "-n", "10", "go", "build", "-trimpath", "-buildvcs=false", "-o", builds / "sp-base", "./cmd/speedprobe"], cwd=source)
    run(["nice", "-n", "10", "go", "build", "-trimpath", "-buildvcs=false", "-o", builds / "sp-new", "./cmd/speedprobe"])
    run(["nice", "-n", "10", "python3", "scripts/speedcheck.py", builds / "sp-base", builds / "sp-new",
         "--net", "engine/nets/ngn-h512-81215229.nnue", "--cpu", "3", "--depth", "10", "--identity-only",
         "--out", output / "replay-identity.json"])
    committed = json.loads((HERE / "identity.json").read_text())
    current = json.loads((output / "replay-identity.json").read_text())
    for key in ("baseline_sha256", "candidate_sha256", "net_sha256", "cpu", "depth", "identity"):
        assert current[key] == committed[key], "identity artifact differs: " + key
    print("Rebuilt starting head and current candidate: committed identity reproduced.")


def smoke():
    output = ROOT.parent / "evidence"
    builds = ROOT.parent / "artifacts"
    output.mkdir(exist_ok=True)
    builds.mkdir(exist_ok=True)
    run(["nice", "-n", "10", "go", "build", "-trimpath", "-buildvcs=false", "-o", builds / "ngn", "."])
    run(["nice", "-n", "10", "go", "build", "-trimpath", "-buildvcs=false", "-o", builds / "sprt", "./cmd/sprt"])
    args = ["nice", "-n", "10", builds / "sprt", "-lowpower=false", "-new", builds / "ngn",
            "-newoption", "T1104PriorBonus=1", "-base", builds / "ngn", "-baseoption", "T1104PriorBonus=0",
            "-nodes", "1000", "-openings", HERE / "smoke-openings.txt", "-concurrency", "1",
            "-elo0", "0", "-elo1", "5", "-mingames", "2", "-maxgames", "2", "-maxmoves", "40",
            "-pgn", output / "replay-smoke.pgn"]
    print("+", shlex.join(map(str, args)), flush=True)
    with (output / "replay-smoke.txt").open("w") as log:
        subprocess.run(list(map(str, args)), cwd=ROOT, check=True, stdout=log, stderr=subprocess.STDOUT)
    result = (output / "replay-smoke.txt").read_text()
    print(result)
    assert "Games: 2   W-D-L: 0-2-0" in result
    assert (output / "replay-smoke.pgn").read_text().count('[Result "1/2-1/2"]') == 2
    artifacts()


def artifacts():
    identity = json.loads((HERE / "identity.json").read_text())
    assert identity["cpu"] == 3 and identity["depth"] == 10
    assert set(identity["identity"]) == {"hce", "nnue"}
    for backend in identity["identity"].values():
        assert backend["equal"] and len(backend["searches"]) == 12
    nomination = json.loads((HERE / "nomination.json").read_text())
    assert nomination["verdict"] == "DEFERRED"
    assert nomination["option"] == {"name": "T1104PriorBonus", "new": 1, "base": 0}
    assert nomination["games"] == 0 and nomination["penta_elo"] is None and nomination["interval"] is None
    assert nomination["wdl"] == {"wins": 0, "draws": 0, "losses": 0}
    assert nomination["pentanomial"] == [0] * 5
    smoke = (HERE / "smoke.txt").read_text()
    assert "mode: nodes 1000/move" in smoke
    assert "Games: 2   W-D-L: 0-2-0" in smoke
    assert "T1104PriorBonus value 1" in smoke and "T1104PriorBonus value 0" in smoke
    assert (HERE / "smoke.pgn").read_text().count('[Result "1/2-1/2"]') == 2
    brief = (ROOT.parent / "BRIEF.md").read_text()
    requirement = next(line[2:] for line in brief.splitlines() if line.startswith("- Commit experiments/"))
    assert requirement in (HERE / "PRESCREEN.md").read_text()
    for name in ("identity.json", "smoke.txt", "smoke-openings.txt", "smoke.pgn", "nomination.json", "PRESCREEN.md"):
        assert git("ls-files", "--error-unmatch", str((HERE / name).relative_to(ROOT)))
    print("Committed identity, two-game smoke, DEFERRED nomination and verbatim rig handoff verified.")


def authors():
    commits = git("log", "--format=%H|%an|%ae|%cn|%ce|%s", BASE + "..HEAD").splitlines()
    assert commits
    for commit in commits:
        head, author, email, committer, cemail, subject = commit.split("|", 5)
        assert (author, email) == ("Bryan Ehrlich", "ehrlich.bryan@gmail.com")
        assert (committer, cemail) == (author, email)
        assert subject.endswith(".") and "\n" not in subject
        print("Verified author and committer:", commit)


def headless():
    changed = git("diff", "--name-only", BASE, "HEAD").splitlines()
    assert set(changed) <= {
        "engine/search.go", "engine/prior_bonus.go", "engine/prior_bonus_test.go",
        *(str(p.relative_to(ROOT)) for p in HERE.iterdir() if p.is_file()),
    }
    final = (ROOT.parent / "final.md").read_text()
    assert "Headless" in final and "DEFERRED" in final
    assert (ROOT.parent / "KEEP").is_file()
    assert not re.search(r"\b(input\(|tkinter|zenity|osascript)\b", (ROOT / "engine/prior_bonus.go").read_text())
    print("Headless decision and retained handoff recorded; change scope:", *changed, sep="\n")


if __name__ == "__main__":
    {"identity": identity, "smoke": smoke, "artifacts": artifacts, "authors": authors, "headless": headless}[sys.argv[1]]()
