#!/usr/bin/env python3
"""Execute the bounded residual proof; results go outside the checkout by default."""
import argparse
import json
import os
from pathlib import Path
import resource
import subprocess
import time

repo = Path(__file__).resolve().parents[3]
lane = repo.parent
parser = argparse.ArgumentParser()
parser.add_argument('check', choices=['fragment', 'oracle', 'walk', 'integration', 'policy', 'vet', 'short'])
parser.add_argument('--verbose', action='store_true')
args = parser.parse_args()
if hasattr(os, 'sched_getaffinity'):
    os.sched_setaffinity(0, sorted(os.sched_getaffinity(0))[:2])
env = {k: v for k, v in os.environ.items()
       if not any(word in k.lower() for word in ('credential', 'token', 'secret', 'password'))}
env.update(GOMAXPROCS='2', OMP_NUM_THREADS='2', OPENBLAS_NUM_THREADS='2',
           MKL_NUM_THREADS='2', NUMEXPR_NUM_THREADS='2', VECLIB_MAXIMUM_THREADS='2',
           GOCACHE='/work/cache/go-build', GOMODCACHE='/work/cache/go-mod', GOPATH='/work/cache/go',
           TMPDIR=str(lane / 'tmp'), npm_config_cache='/work/cache/npm', UV_CACHE_DIR='/work/cache/uv',
           UV_PYTHON_INSTALL_DIR='/work/cache/uv-python')
Path(env['TMPDIR']).mkdir(exist_ok=True)
patterns = {
    'fragment': '^TestWave2ResidualEval',
    'oracle': '^TestWave2ResidualEval(Oracle|StorageBounds|DefaultH768Zero)$',
    'walk': '^TestWave2ResidualEvalWalk$',
    'integration': '^TestWave2ResidualEval(UCIIdentity|DefaultH768Zero)$',
}
if args.check in patterns:
    command = ['nice', '-n', '10', 'go', 'test', '-p', '1', '-count=1', './engine',
               '-run', patterns[args.check], '-timeout', '15s']
    if args.verbose:
        command.append('-v')
elif args.check == 'vet':
    command = ['nice', '-n', '10', 'go', 'vet', '-p', '2', './...']
elif args.check == 'short':
    command = ['nice', '-n', '10', 'go', 'test', '-p', '2', '-short', '-count=1', './...']
    # The brief explicitly prohibits kernel tests on forge. The unmodified short
    # suite includes them without a testing.Short guard. Do not pretend it passed.
    if os.uname().nodename.startswith('forge'):
        print('BLOCKED on ' + os.uname().nodename + ': brief prohibits kernel tests; run on a suitable host.', flush=True)
        print('Required command: GOMAXPROCS=2 ' + ' '.join(command), flush=True)
        raise SystemExit(75)
else:
    def git(*a):
        return subprocess.check_output(['git', *a], cwd=repo, env=env, text=True).strip()
    base = '4cab8e04fd8afd651cb11ca4199e40b3958ca20f'
    subprocess.run(['git', 'merge-base', '--is-ancestor', base, 'HEAD'], cwd=repo, env=env, check=True)
    commits = git('rev-list', base + '..HEAD').splitlines()
    assert commits, 'candidate must be committed'
    for commit in commits:
        expected = 'Bryan Ehrlich <ehrlich.bryan@gmail.com>'
        assert git('show', '-s', '--format=%an <%ae>', commit) == expected
        assert git('show', '-s', '--format=%cn <%ce>', commit) == expected
    # Fixtures are local and default bytes must match the prerequisite candidate.
    subprocess.run(['git', 'diff', '--exit-code', base, 'HEAD', '--',
                    'engine/default_net.go', 'engine/nets/'], cwd=repo, env=env, check=True)
    print(json.dumps({'base': base, 'head': git('rev-parse', 'HEAD'), 'commits': len(commits),
                      'author_committer': expected, 'embedded_default_unchanged': True,
                      'sampling': 'none; no speed or Elo claim'}))
    raise SystemExit(0)
print('GOMAXPROCS=2 ' + ' '.join(command), flush=True)
start = time.monotonic()
process = subprocess.Popen(command, cwd=repo, env=env)
_, status, usage = os.wait4(process.pid, 0)
code = os.waitstatus_to_exitcode(status)
process.returncode = code
print(json.dumps({'exit_code': code, 'elapsed_seconds': round(time.monotonic()-start, 3),
                  'cpu_seconds': round(usage.ru_utime+usage.ru_stime, 3),
                  'peak_rss_kib': usage.ru_maxrss, 'diagnostic_only': True}), flush=True)
raise SystemExit(code)
