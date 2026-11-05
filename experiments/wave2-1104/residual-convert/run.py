#!/usr/bin/env python3
"""Execute bounded conversion/parity proofs and write receipts outside the checkout."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

REPO = Path(__file__).resolve().parents[3]
HERE = Path(__file__).resolve().parent
BASE = "9209acb927dcb2d7d8f6355373f9504f1aea0c9b"


def setup():
    if hasattr(os, "sched_getaffinity"):
        os.sched_setaffinity(0, sorted(os.sched_getaffinity(0))[:2])
    for key in list(os.environ):
        if any(word in key.lower() for word in ("credential", "token", "secret", "password")):
            del os.environ[key]
    os.environ.update(GOMAXPROCS="2", OMP_NUM_THREADS="2", OPENBLAS_NUM_THREADS="2",
                      MKL_NUM_THREADS="2", NUMEXPR_NUM_THREADS="2", VECLIB_MAXIMUM_THREADS="2",
                      GOCACHE="/work/cache/go-build", GOMODCACHE="/work/cache/go-mod",
                      GOPATH="/work/cache/go", npm_config_cache="/work/cache/npm",
                      UV_CACHE_DIR="/work/cache/uv", UV_PYTHON_INSTALL_DIR="/work/cache/uv-python",
                      TMPDIR=str(REPO.parent/"tmp"), PYTHONDONTWRITEBYTECODE="1")
    Path(os.environ["TMPDIR"]).mkdir(exist_ok=True)
    sys.path.insert(0, str(REPO))


def execute(command, extra_env=None):
    env = os.environ.copy()
    env.update(extra_env or {})
    print("GOMAXPROCS=2 " + " ".join(command), flush=True)
    start = time.monotonic()
    process = subprocess.Popen(["nice", "-n", "10", *command], cwd=REPO, env=env)
    _, status, usage = os.wait4(process.pid, 0)
    code = os.waitstatus_to_exitcode(status)
    process.returncode = code
    print(json.dumps({"exit_code": code, "elapsed_seconds": round(time.monotonic()-start, 3),
                      "cpu_seconds": round(usage.ru_utime+usage.ru_stime, 3),
                      "peak_rss_kib": usage.ru_maxrss}), flush=True)
    if code:
        raise SystemExit(code)


def h768(output, bootstrap):
    import numpy as np
    from trainer.ngnp import RECORD_DTYPE, decode_record, to_fen
    from trainer.refeval import Network as LegacyReference
    from trainer.residual_export import lift_bytes, receipt
    from trainer.residual_refeval import Network
    source_path = REPO/"engine/nets/ngn-h768-2dcb6d93.nnue"
    source = source_path.read_bytes()
    assert hashlib.sha256(source).hexdigest() == "2dcb6d936c0275927cb9e881f42e22558e618ad7e6fd998df6c4e5046d42c443"
    model = output/"h768-zero.ngn6"
    if not model.exists():
        model.write_bytes(lift_bytes(source))
        print("Converted accepted H768 into scratch", flush=True)
    converted = model.read_bytes()
    conversion = receipt(source, converted)
    base, zero = Network(source), Network(converted)
    legacy = LegacyReference(source_path)
    records = np.fromfile(HERE/"records.ngnp", dtype=RECORD_DTYPE)
    cases = []
    assert len(records) == 32
    for rec in records:
        p = decode_record(rec)
        score = base.evaluate(p)
        assert zero.evaluate(p) == score == legacy.evaluate(p)
        assert np.array_equal(base.accumulators(p), zero.accumulators(p))
        cases.append({"fen": to_fen(p), "bucket": base.bucket(p), "stm": p.stm, "cp": score})
    conversion["integer_parity"] = cases
    (output/"h768-conversion.json").write_text(json.dumps(conversion, indent=2)+"\n")
    search_receipt = output/"h768-search.json"
    execute(["go", "test", "-p", "1", "-count=1", "./engine", "-run", "^TestWave2ResidualParity", "-timeout", "15s", "-v"],
            {"NGN_RESIDUAL_H768": str(model), "NGN_RESIDUAL_SEARCH_RECEIPT": str(search_receipt),
             "NGN_RESIDUAL_INTEGER_RECEIPT": str(output/"h768-conversion.json")})
    combined = {"conversion": conversion, "search": json.loads(search_receipt.read_text())}
    actual = output/"h768-receipt.json"
    actual.write_text(json.dumps(combined, indent=2)+"\n")
    if not bootstrap:
        assert json.loads((HERE/"h768-receipt.json").read_text()) == combined, "H768 receipt changed"
    print(json.dumps({"h768_output_sha256": conversion["output"]["sha256"],
                      "preserved_payload_bytes": conversion["preserved_payload_bytes"],
                      "numpy_integer_records": len(cases), "exact_depth4_searches": 4}), flush=True)


def policy():
    def git(*args):
        return subprocess.check_output(["git", *args], cwd=REPO, text=True).strip()
    subprocess.run(["git", "merge-base", "--is-ancestor", BASE, "HEAD"], cwd=REPO, check=True)
    commits = git("rev-list", BASE+"..HEAD").splitlines()
    assert commits, "candidate must be committed"
    identity = "Bryan Ehrlich <ehrlich.bryan@gmail.com>"
    for commit in commits:
        assert git("show", "-s", "--format=%an <%ae>", commit) == identity
        assert git("show", "-s", "--format=%cn <%ce>", commit) == identity
    subprocess.run(["git", "diff", "--exit-code", BASE, "HEAD", "--", "engine/default_net.go", "engine/nets"], cwd=REPO, check=True)
    print(json.dumps({"base": BASE, "head": git("rev-parse", "HEAD"), "commits": len(commits),
                      "author_committer": identity, "embedded_default_unchanged": True,
                      "cpu_affinity": sorted(os.sched_getaffinity(0)), "gpu": "unused",
                      "claim": "integer parity only; trained nonzero weights require the same oracle and later strength gate"}))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("check", choices=("python", "parity", "packet", "h768", "policy", "vet", "short", "diff-stack", "diff-main"))
    parser.add_argument("--output-dir", type=Path, default=REPO.parent/"scratch/residual-convert")
    parser.add_argument("--bootstrap", action="store_true", help="Capture the first H768 receipt before committing it")
    args = parser.parse_args()
    setup()
    output = args.output_dir.resolve()
    if output == REPO or REPO in output.parents:
        raise ValueError("Evidence and converted H768 must be outside the checkout")
    output.mkdir(parents=True, exist_ok=True)
    if args.check == "h768":
        h768(output, args.bootstrap)
    elif args.check == "policy":
        policy()
    elif args.check == "python":
        execute(["python3", "-m", "unittest", "trainer.tests.test_residual_export"])
    elif args.check == "packet":
        execute(["python3", str(HERE/"generate.py"), "--check"])
    elif args.check == "parity":
        execute(["go", "test", "-p", "1", "-count=1", "./engine", "-run", "^TestWave2ResidualParity", "-timeout", "15s", "-v"])
    elif args.check == "vet":
        execute(["go", "vet", "-p", "2", "./..."])
    elif args.check == "short":
        if os.uname().nodename == "forge":
            print("BLOCKED: kernel tests prohibited on forge; run the short gate on a suitable host.")
            raise SystemExit(75)
        execute(["go", "test", "-p", "2", "-short", "-count=1", "./..."])
    else:
        base = "8ba878cb03a7eb81ca465b3d562953d0f45e81c7" if args.check == "diff-stack" else "3517a353a99bbe627a505ccc5463aadce047431b"
        execute(["git", "diff", "--check", base, "HEAD"])


if __name__ == "__main__":
    main()
