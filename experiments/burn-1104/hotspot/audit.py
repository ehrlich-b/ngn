#!/usr/bin/env python3
"""Read-only checks for the patch-only experiment's provenance and contract."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import socket
import subprocess

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
BASE = "6d3168b8c5f15532cc7a352d05465a77ab50f361"


def git(*args):
    env = {k: v for k, v in os.environ.items()
           if not any(s in k.lower() for s in ("credential", "token", "secret", "password"))}
    return subprocess.check_output(["git", *args], cwd=ROOT, env=env, text=True).strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("check", choices=("host", "authors", "contract"))
    args = parser.parse_args()
    if args.check == "host":
        assert os.geteuid() != 0, "ordinary proof must not run as root"
        print(json.dumps(dict(host=socket.gethostname(), uid=os.geteuid(),
                              checks="ordinary Go vet, short tests, user-space pprof and fixed searches",
                              blocked_kernel_checks=[], reason="no OS-kernel operations are required")))
    elif args.check == "authors":
        expected = "Bryan Ehrlich <ehrlich.bryan@gmail.com>"
        rows = git("log", "--format=%H%x09%an <%ae>%x09%cn <%ce>%x09%B%x00", BASE + "..HEAD").split("\0")
        count = 0
        for row in rows:
            if not row.strip():
                continue
            commit, author, committer, message = row.strip().split("\t", 3)
            assert author == committer == expected, (commit, author, committer)
            assert len(message.strip().splitlines()) == 1
            count += 1
            print(json.dumps(dict(commit=commit, author=author, committer=committer, subject=message.strip())))
        assert count > 0, "commit the candidate before the readiness run"
    else:
        m = json.loads((HERE / "manifest.json").read_text())
        patch = (HERE / "candidate.patch").read_bytes()
        assert m["base"] == BASE
        assert hashlib.sha256(patch).hexdigest() == m["patch_sha256"]
        assert patch.decode().count("diff --git ") == 1
        assert patch.decode().startswith("diff --git a/engine/cache.go b/engine/cache.go\n")
        assert not git("diff", "2e7bc9da", "HEAD", "--", "engine/cache.go")
        assert hashlib.sha256((HERE / "qualification.txt").read_bytes()).hexdigest() == "6711573eece1ce3e081fcb2708901998bac07ae90c7e556f62639bc2e666809d"
        corpus = json.loads((HERE / "positions.json").read_text())
        assert len(corpus) == len(set(corpus)) == 12
        assert m["bounded_check"]["depth"] == 4 and m["bounded_check"]["threads"] == 1
        assert m["bounded_check"]["timeout_seconds"] == 120
        pending = m["full_qualification"]
        exact = (HERE / "qualification.txt").read_text().rstrip("\n")
        assert pending["pending_requirements_verbatim"] == exact
        assert pending["status"].startswith("pending")
        assert pending["speedcheck_argv"] in exact
        print(json.dumps(dict(head=git("rev-parse", "HEAD"), cache_engine_unchanged=True,
                              patch_sha256=m["patch_sha256"], corpus_positions=len(corpus),
                              qualification=pending["status"], headless=True,
                              decision="Keep the single candidate as an opt-in patch; defer timing/adoption to the rig as the brief requires; no questions or dialogs are needed."), indent=2))


if __name__ == "__main__":
    main()
