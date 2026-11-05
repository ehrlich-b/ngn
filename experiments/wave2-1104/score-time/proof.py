#!/usr/bin/env python3
"""Run a bounded proof with sanitized credentials and native wait4 accounting."""
import json
import os
from pathlib import Path
import resource
import subprocess
import sys
import time

lane = Path.cwd().parent
env = {k: v for k, v in os.environ.items() if not any(w in k.lower() for w in ('credential', 'token', 'secret', 'password'))}
env.update(GOCACHE='/work/cache/go-build', GOMODCACHE='/work/cache/go-mod', GOPATH='/work/cache/go', GOMAXPROCS='2', OMP_NUM_THREADS='2', OPENBLAS_NUM_THREADS='2', MKL_NUM_THREADS='2', NUMEXPR_NUM_THREADS='2', CUDA_VISIBLE_DEVICES='', npm_config_cache='/work/cache/npm', UV_CACHE_DIR='/work/cache/uv', UV_PYTHON_INSTALL_DIR='/work/cache/uv-python', TMPDIR=str(lane/'tmp'), NGN_DEBUG_LOG='')
if hasattr(os, 'sched_getaffinity'):
    os.sched_setaffinity(0, sorted(os.sched_getaffinity(0))[:2])
command = sys.argv[1:]
print(json.dumps({'command': command, 'cwd': '.', 'cpu_affinity': sorted(os.sched_getaffinity(0)), 'gomaxprocs': env['GOMAXPROCS']}), flush=True)
started = time.monotonic()
process = subprocess.Popen(['nice', '-n', '10', *command], env=env)
_, status, usage = os.wait4(process.pid, 0)
process.returncode = os.waitstatus_to_exitcode(status)
print(json.dumps({'exit_code': process.returncode, 'elapsed_seconds': round(time.monotonic()-started, 6), 'user_seconds': usage.ru_utime, 'system_seconds': usage.ru_stime, 'max_rss_kib': usage.ru_maxrss}), flush=True)
sys.exit(process.returncode)
