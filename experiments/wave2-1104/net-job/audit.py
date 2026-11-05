#!/usr/bin/env python3
"""Read-only host, resource, provenance and default-play fixture assertions."""
import argparse
import json
import os
from pathlib import Path
import socket
import subprocess
import sys

REPO = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(REPO))
from scripts.wave2 import net_job as nj  # noqa: E402

BASE = "3517a353a99bbe627a505ccc5463aadce047431b"
FIX = Path(__file__).resolve().parent


def git(*args):
    return subprocess.check_output(["git", *args], cwd=REPO, text=True).strip()


def main(mode):
    if mode == "host":
        hostname = socket.gethostname()
        print(json.dumps({"hostname": hostname, "is_forge": hostname == "forge",
                          "proofs": "ordinary Python and Go processes only"}))
    elif mode == "resources":
        env = nj.environment()
        assert env["GOMAXPROCS"] == "2" and env["GOFLAGS"] == "-p=1"
        assert all(env[k] == "2" for k in ("OMP_NUM_THREADS", "MKL_NUM_THREADS", "OPENBLAS_NUM_THREADS", "NUMEXPR_NUM_THREADS"))
        assert env["CUDA_VISIBLE_DEVICES"] == ""
        assert len(os.sched_getaffinity(0)) <= 2, "run this assertion through time_v.py"
        result = nj.read(FIX / "result.json")
        net = FIX / "tiny-owned.nnue"
        assert nj.net_format(net) == (1, 16) and net.stat().st_size == 24704
        assert nj.sha(net) == result["net_sha256"] == result["embedded_sha256"]
        assert result["fixture_only"] and not result["elo_claim"] and result["games_played"] == 0
        assert result["identity"]["cases"] == 4 and result["identity"]["depth"] == 4
        assert result["parity_passed"] and result["startup_identity_passed"]
        print(json.dumps({"cpus": sorted(os.sched_getaffinity(0)), "net": nj.artifact(net),
                          "net_bytes": net.stat().st_size, "fixture_result": nj.artifact(FIX / "result.json"),
                          "format": "NGNN1", "hidden": 16, "games": 0}))
    elif mode == "policy":
        commits = git("log", "--format=%H|%an|%ae|%cn|%ce", BASE + "..HEAD").splitlines()
        assert commits, "commit the candidate first"
        for row in commits:
            head, author, email, committer, committer_email = row.split("|")
            assert author == committer == "Bryan Ehrlich"
            assert email == committer_email == "ehrlich.bryan@gmail.com"
        assert not git("diff", BASE, "HEAD", "--", "engine/default_net.go", "engine/nets")
        assert not git("status", "--porcelain", "--untracked-files=all")
        print(json.dumps({"head": git("rev-parse", "HEAD"), "branch": git("branch", "--show-current"),
                          "commits": [row.split("|")[0] for row in commits],
                          "author_and_committer": "Bryan Ehrlich <ehrlich.bryan@gmail.com>",
                          "default_net_unchanged": nj.BASE_NET}))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("host", "resources", "policy"))
    main(parser.parse_args().mode)
