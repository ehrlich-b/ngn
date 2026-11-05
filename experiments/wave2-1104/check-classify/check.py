#!/usr/bin/env python3
"""Bounded checking-classification proof; all execution logs stay outside Git."""
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import socket
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[3]
HERE = Path(__file__).resolve().parent
BASE = "3517a353a99bbe627a505ccc5463aadce047431b"
FRAGMENT = ["nice", "-n", "10", "go", "test", "-p", "1", "-count=1",
            "./engine", "-run", "^TestWave2GivesCheck", "-timeout", "15s", "-v"]


def environment():
    env = {k: v for k, v in os.environ.items()
           if not re.search(r"credential|token|secret|password", k, re.I)}
    env.update(GOCACHE="/work/cache/go-build", GOMODCACHE="/work/cache/go-mod",
               GOPATH="/work/cache/go", GOMAXPROCS="2",
               npm_config_cache="/work/cache/npm", UV_CACHE_DIR="/work/cache/uv",
               UV_PYTHON_INSTALL_DIR="/work/cache/uv-python",
               TMPDIR=str(ROOT.parent / "tmp"), OMP_NUM_THREADS="2",
               OPENBLAS_NUM_THREADS="2", MKL_NUM_THREADS="2",
               NUMEXPR_NUM_THREADS="2", CUDA_VISIBLE_DEVICES="",
               PLAYWRIGHT_BROWSERS_PATH="/work/cache/ms-playwright",
               PYTHONDONTWRITEBYTECODE="1")
    env["PATH"] = "/usr/local/go/bin:" + env.get("PATH", os.defpath)
    Path(env["TMPDIR"]).mkdir(exist_ok=True)
    return env


def output_dir():
    path = Path(os.environ.get("CHECK_CLASSIFY_OUTPUT", ROOT.parent / "evidence/check-classify")).resolve()
    if path == ROOT or ROOT in path.parents:
        raise ValueError("execution evidence must be outside the checkout")
    path.mkdir(parents=True, exist_ok=True)
    return path


def run(label, args):
    """Native wait4 accounting, avoiding a dependency on /usr/bin/time."""
    folder = output_dir()
    log = folder / (label + ".log")
    started = time.monotonic()
    with log.open("w") as stream:
        stream.write("$ GOMAXPROCS=2 " + shlex.join(args) + "\n")
        stream.flush()
        process = subprocess.Popen(args, cwd=ROOT, env=environment(),
                                   stdout=stream, stderr=subprocess.STDOUT)
        _, status, usage = os.wait4(process.pid, 0)
        rc = os.waitstatus_to_exitcode(status)
        process.returncode = rc
    result = dict(command=shlex.join(args), exit_code=rc, head=git("rev-parse", "HEAD"),
                  elapsed_seconds=time.monotonic() - started,
                  user_seconds=usage.ru_utime, system_seconds=usage.ru_stime,
                  peak_rss_kib=usage.ru_maxrss,
                  log_sha256=hashlib.sha256(log.read_bytes()).hexdigest())
    (folder / (label + ".json")).write_text(json.dumps(result, indent=2) + "\n")
    print(log.read_text(), end="")
    print(json.dumps(result, sort_keys=True), flush=True)
    return rc, log.read_text()


def git(*args):
    return subprocess.check_output(["git", *args], cwd=ROOT, env=environment(), text=True).strip()


def main():
    if hasattr(os, "sched_setaffinity"):
        os.sched_setaffinity(0, sorted(os.sched_getaffinity(0))[:2])
    mode = sys.argv[1]
    if mode == "fragment":
        return run("fragment", FRAGMENT)[0]
    if mode == "mutation":
        # Go's overlay replaces only the candidate predicate; neither source
        # nor any other file is moved, renamed or removed.
        scratch = Path(tempfile.mkdtemp(prefix="check-mutation-", dir=environment()["TMPDIR"]))
        overlay = scratch / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {
            str(ROOT / "engine/wave2_check.go"): str(HERE / "legacy_check.go.txt")}}))
        command = FRAGMENT.copy()
        command[5:5] = ["-overlay", str(overlay)]
        rc, log = run("legacy-mutation", command)
        assert rc == 1 and "--- FAIL: TestWave2GivesCheckCorpus" in log
        assert "predicate=false, make/isInCheck/unmake=true" in log
        assert "discovered-and-double" in log and "ep-rank-discovery" in log
        assert "[build failed]" not in log
        print("PASS: historical moved-piece-only predicate rejected by retained witnesses")
        return 0
    if mode == "scope":
        changed = git("diff", "--name-only", BASE, "--", "engine").splitlines()
        assert set(changed) == {"engine/wave2_check.go", "engine/wave2_check_test.go"}, changed
        src = (ROOT / "engine/wave2_check.go").read_text()
        for token in ["pseudo-legal", "own king safety", "IsEnPassant", "IsCastle", "PromoType",
                      "GetBishopAttacks", "GetRookAttacks"]:
            assert token in src, token
        assert "MakeMove(" not in src and "applyMoveBoard(" not in src
        print("PASS: board-local predicate; only the two new engine files differ; no search/evasion hook")
        return 0
    if mode == "host":
        host = socket.gethostname()
        print(json.dumps(dict(host=host, uid=os.geteuid(),
                              allowed_cpus=sorted(os.sched_getaffinity(0)),
                              kernel_device_checks="none required or executed",
                              exact_forge_host=(host == "forge"))))
        assert os.geteuid() != 0
        return 0
    if mode == "identity":
        commits = git("log", "--format=%an <%ae>|%cn <%ce>", BASE + "..HEAD").splitlines()
        expected = "Bryan Ehrlich <ehrlich.bryan@gmail.com>"
        assert commits and all(c == expected + "|" + expected for c in commits), commits
        assert git("merge-base", BASE, "HEAD") == BASE
        assert not git("diff", BASE, "HEAD", "--", "engine/search.go", "engine/movegen.go")
        print("PASS: all new commits authored/committed by " + expected + "; production search unchanged")
        return 0
    if mode == "artifacts":
        for name in ["corpus.json", "legacy_check.go.txt", "check.py", "README.md"]:
            assert git("ls-files", "--error-unmatch", str(HERE.relative_to(ROOT) / name))
        assert os.access(__file__, os.X_OK)
        data = json.loads((HERE / "corpus.json").read_text())
        assert len(data) == 46 and sum(len(c["Witnesses"]) for c in data) == 110
        folder = output_dir()
        results = {name: json.loads((folder / (name + ".json")).read_text())
                   for name in ["fragment", "legacy-mutation"]}
        for name, result in results.items():
            assert result["head"] == git("rev-parse", "HEAD"), name
            assert result["log_sha256"] == hashlib.sha256((folder / (name + ".log")).read_bytes()).hexdigest()
        assert results["fragment"]["exit_code"] == 0
        assert results["legacy-mutation"]["exit_code"] == 1
        (folder / "result.json").write_text(json.dumps(results, indent=2) + "\n")
        print("PASS: committed fixed corpus and executable command; generated result.json/logs outside checkout; affinity=2, GOMAXPROCS=2, tensor threads=2, no speed sample")
        return 0
    if mode == "run":
        label = sys.argv[2]
        assert sys.argv[3] == "--"
        return run(label, sys.argv[4:])[0]
    raise ValueError("unknown mode: " + mode)


if __name__ == "__main__":
    sys.exit(main())
