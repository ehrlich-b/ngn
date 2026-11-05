#!/usr/bin/env python3
"""Run the small residual format proof with bounded CPU concurrency."""
import os
from pathlib import Path


root = Path(__file__).resolve().parents[3]
env = {k: v for k, v in os.environ.items()
       if not any(word in k.lower() for word in ('credential', 'token', 'secret', 'password'))}
env.update(GOMAXPROCS='2', OMP_NUM_THREADS='2', OPENBLAS_NUM_THREADS='2',
           MKL_NUM_THREADS='2', NUMEXPR_NUM_THREADS='2', VECLIB_MAXIMUM_THREADS='2',
           GOCACHE='/work/cache/go-build', GOMODCACHE='/work/cache/go-mod', GOPATH='/work/cache/go')
os.chdir(root)
command = ['nice', '-n', '10', 'go', 'test', '-p', '1', '-count=1', './engine',
           '-run', '^TestWave2ResidualFormat', '-timeout', '15s']
print('GOMAXPROCS=2 ' + ' '.join(command), flush=True)
os.execvpe(command[0], command, env)
