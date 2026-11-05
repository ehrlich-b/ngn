#!/usr/bin/env python3
"""Run proof commands with the prescribed shared cache and credential-free environment."""
import os
import re
import sys
from pathlib import Path
root = Path(__file__).resolve().parent
(root / 'tmp').mkdir(exist_ok=True)
env = {k: v for k, v in os.environ.items() if not re.search(r'credential|token|secret|password', k, re.I)}
env.update(GOCACHE='/work/cache/go-build', GOMODCACHE='/work/cache/go-mod', GOPATH='/work/cache/go', GOMAXPROCS='2', npm_config_cache='/work/cache/npm', UV_CACHE_DIR='/work/cache/uv', UV_PYTHON_INSTALL_DIR='/work/cache/uv-python', TMPDIR=str(root / 'tmp'))
os.execvpe('nice', ['nice', '-n', '10', *sys.argv[1:]], env)
