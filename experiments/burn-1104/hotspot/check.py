#!/usr/bin/env python3
"""Build a patch-only candidate and run bounded profile/identity proofs."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
BASE = "6d3168b8c5f15532cc7a352d05465a77ab50f361"


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", required=True)
    args = parser.parse_args()
    out = Path(args.out).resolve()
    if out == ROOT or ROOT in out.parents:
        parser.error("evidence must be outside the checkout")
    out.mkdir(parents=True, exist_ok=True)
    env = {k: v for k, v in os.environ.items()
           if not any(s in k.lower() for s in ("credential", "token", "secret", "password"))}
    scratch = Path(env.get("TMPDIR", str(ROOT.parent / "tmp")))
    if ROOT.parent != scratch and ROOT.parent not in scratch.parents:
        parser.error("TMPDIR must be inside the assigned directory")
    scratch.mkdir(parents=True, exist_ok=True)
    env.update(GOCACHE="/work/cache/go-build", GOMODCACHE="/work/cache/go-mod",
               GOPATH="/work/cache/go", GOAMD64="v3", GOMAXPROCS="2", TMPDIR=str(scratch))
    work = Path(tempfile.mkdtemp(prefix="hotspot-", dir=scratch))
    records = []

    def run(name, command, cwd=ROOT, timeout=120, extra=None):
        started = time.monotonic()
        process = subprocess.run(["nice", "-n", "10", *map(str, command)], cwd=cwd,
                                 env=dict(env, **(extra or {})), text=True,
                                 stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout)
        (out / (name + ".stdout")).write_text(process.stdout)
        (out / (name + ".stderr")).write_text(process.stderr)
        records.append(dict(name=name, command=list(map(str, command)), cwd=str(cwd),
                            elapsed_seconds=time.monotonic()-started, exit_code=process.returncode))
        (out / "commands.json").write_text(json.dumps(records, indent=2) + "\n")
        if process.returncode:
            raise RuntimeError(f"{name} failed: {process.stderr}")
        return process.stdout

    manifest = json.loads((HERE / "manifest.json").read_text())
    assert manifest["base"] == BASE
    assert manifest["patch_sha256"] == digest(HERE / "candidate.patch")
    changed = run("default-tree", ["git", "diff", "--exit-code", "2e7bc9da", "HEAD", "--", "engine/cache.go"])
    assert not changed
    head = run("head", ["git", "rev-parse", "HEAD"]).strip()
    run("toolchain", ["go", "version"])
    trees = {}
    for side, revision in [("before", head), ("after", head)]:
        tree = work / side
        run(side + "-checkout", ["git", "worktree", "add", "--detach", tree, revision])
        trees[side] = tree
        if side == "after":
            run("patch-check", ["git", "apply", "--check", HERE / "candidate.patch"], tree)
            run("patch-apply", ["git", "apply", HERE / "candidate.patch"], tree)
        shutil.copyfile(HERE / "cache_get_test.go.txt", tree / "engine/hotspot_get_test.go")
        run(side + "-focused", ["go", "test", "-p", "2", "-short", "-count=1", "-run",
                                "^(TestHotspotGetEquivalence|TestSynchronizedCacheReturnsOnlyCoherentRecords)$", "-v", "./engine"], tree)
        run(side + "-build", ["go", "build", "-p", "2", "-trimpath", "-o", work / (side + "-probe"), HERE / "probe.go"], tree)

    corpus = HERE / "positions.json"
    assert len(json.loads(corpus.read_text())) == 12
    deadline = time.monotonic() + 120
    identity = {}
    cpu = 3 if 3 in os.sched_getaffinity(0) else min(os.sched_getaffinity(0))
    for backend in ("hce", "nnue"):
        pair = {}
        for side in ("before", "after"):
            output = run(side + "-" + backend + "-depth4", ["taskset", "-c", cpu, work / (side + "-probe"),
                         "-corpus", corpus, "-depth", "4", "-backend", backend], trees[side],
                         timeout=max(.01, deadline-time.monotonic()), extra={"GOMAXPROCS": "1"})
            pair[side] = [json.loads(line) for line in output.splitlines()]
            assert len(pair[side]) == 12
            assert all(row["nodes"] > 0 and row["threads"] == 1 for row in pair[side])
        fields = ("position", "fen", "nodes", "score", "move", "threads")
        a, b = [[{key: row[key] for key in fields} for row in pair[side]] for side in ("before", "after")]
        identity[backend] = dict(equal=a == b, baseline=a, candidate=b)
        (out / "identity.json").write_text(json.dumps(identity, indent=2) + "\n")
        assert a == b, f"{backend} changed play"
    identity_seconds = 120 - (deadline-time.monotonic())
    assert identity_seconds < 120

    profiles = {}
    for side in ("before", "after"):
        output = run(side + "-profile", ["taskset", "-c", cpu, work / (side + "-probe"), "-corpus", corpus,
                     "-position", "0", "-depth", "20", "-backend", "nnue", "-profile", out / (side + ".pprof")],
                     trees[side], extra={"GOMAXPROCS": "1"})
        profiles[side] = json.loads(output)
        top = run(side + "-top", ["go", "tool", "pprof", "-top", "-nodecount=80", work / (side + "-probe"), out / (side + ".pprof")])
        (out / (side + "-top.txt")).write_text(top)
    fields = ("nodes", "score", "move", "threads")
    assert all(profiles["before"][k] == profiles["after"][k] for k in fields)
    # The deliverable leaves the cache implementation untouched, so also run the
    # registered Go gates against the private patch-applied engine.
    run("candidate-vet", ["go", "vet", "-p", "2", "./..."], trees["after"], timeout=900)
    run("candidate-short", ["go", "test", "-p", "2", "-short", "-count=1", "./..."], trees["after"], timeout=900)
    result = dict(base=BASE, head=head, patch_sha256=digest(HERE / "candidate.patch"),
                  cache_engine_unchanged=True, identity_depth=4, identity_positions=12,
                  identity_seconds=identity_seconds, identity=identity,
                  profile_depth=20, profiles=profiles, cpu=cpu,
                  candidate_gates="vet and short tests passed",
                  fixed_pair_gain_pct=100*(profiles["before"]["seconds"]/profiles["after"]["seconds"]-1),
                  rig_qualification="pending; fixed-pair elapsed time is diagnostic, not a KEEP decision")
    (out / "result.json").write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps({k: v for k, v in result.items() if k != "identity"}, indent=2))


if __name__ == "__main__":
    main()
