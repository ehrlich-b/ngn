#!/usr/bin/env python3
"""Replay the small correction-margin fixture with bounded CPU resources."""
import json
import os
from pathlib import Path
import re
import shlex
import subprocess
import sys
import time

COMMAND = ["nice", "-n", "10", "go", "test", "-p", "1", "-count=1", "./engine", "-run", "^TestWave2CorrMargin", "-timeout", "15s", "-v"]

def main():
    command = sys.argv[1:] or COMMAND
    env = {k: v for k, v in os.environ.items() if not re.search(r"credential|token|secret|password", k, re.I)}
    scratch = Path(os.environ.get("TMPDIR", str(Path.cwd().parent / "tmp")))
    scratch.mkdir(parents=True, exist_ok=True)
    env.update(GOCACHE="/work/cache/go-build", GOMODCACHE="/work/cache/go-mod", GOPATH="/work/cache/go",
               GOMAXPROCS="2", OMP_NUM_THREADS="2", OPENBLAS_NUM_THREADS="2", MKL_NUM_THREADS="2",
               CUDA_VISIBLE_DEVICES="", TMPDIR=str(scratch), PYTHONDONTWRITEBYTECODE="1",
               npm_config_cache="/work/cache/npm", UV_CACHE_DIR="/work/cache/uv",
               UV_PYTHON_INSTALL_DIR="/work/cache/uv-python", PLAYWRIGHT_BROWSERS_PATH="/work/cache/ms-playwright")
    env["PATH"] = "/usr/local/go/bin:" + env.get("PATH", os.defpath)
    if hasattr(os, "sched_setaffinity"):
        os.sched_setaffinity(0, sorted(os.sched_getaffinity(0))[:2])
    print("$ " + shlex.join(command), flush=True)
    started = time.monotonic()
    child = subprocess.Popen(command, env=env)
    _, status, usage = os.wait4(child.pid, 0)
    rc = os.waitstatus_to_exitcode(status)
    child.returncode = rc
    print(json.dumps(dict(exit_code=rc, elapsed_seconds=time.monotonic()-started,
                          user_seconds=usage.ru_utime, system_seconds=usage.ru_stime,
                          peak_rss_kib=usage.ru_maxrss, cpu_limit=2, GOMAXPROCS=2,
                          tensor_threads=2, argv=command), sort_keys=True), flush=True)
    return rc

if __name__ == "__main__":
    sys.exit(main())
