#!/usr/bin/env python3
"""Run bounded residual training proofs on CPU using the existing Torch venv."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[3]
HERE = Path(__file__).resolve().parent
BASE = '9bf6038dcac96f8cdb3bd59036c5d5ed1d49ad55'
PYTHON = '/work/cache/ngn-nnue-training/venv/bin/python'


def setup():
    os.sched_setaffinity(0, sorted(os.sched_getaffinity(0))[:2])
    for key in list(os.environ):
        if any(word in key.lower() for word in ('credential', 'token', 'secret', 'password')):
            del os.environ[key]
    numpy_path = os.environ.get('NGN_RESIDUAL_NUMPY', '/work/cache/ngn-nnue-training/numpy-cp311')
    os.environ.update(GOMAXPROCS='2', OMP_NUM_THREADS='2', OPENBLAS_NUM_THREADS='2',
                      MKL_NUM_THREADS='2', NUMEXPR_NUM_THREADS='2', VECLIB_MAXIMUM_THREADS='2',
                      GOCACHE='/work/cache/go-build', GOMODCACHE='/work/cache/go-mod',
                      GOPATH='/work/cache/go', npm_config_cache='/work/cache/npm',
                      UV_CACHE_DIR='/work/cache/uv', UV_PYTHON_INSTALL_DIR='/work/cache/uv-python',
                      TMPDIR=str(ROOT.parent/'tmp'), PYTHONDONTWRITEBYTECODE='1',
                      PYTHONPATH=os.pathsep.join((str(ROOT), numpy_path)), CUDA_VISIBLE_DEVICES='')
    Path(os.environ['TMPDIR']).mkdir(exist_ok=True)


def execute(command):
    print(json.dumps({'command': command, 'affinity': sorted(os.sched_getaffinity(0)),
                      'GOMAXPROCS': os.environ['GOMAXPROCS'], 'host': os.uname().nodename}), flush=True)
    start = time.monotonic()
    process = subprocess.Popen(['nice', '-n', '10', *command], cwd=ROOT, env=os.environ.copy())
    _, status, usage = os.wait4(process.pid, 0)
    code = os.waitstatus_to_exitcode(status)
    process.returncode = code
    print(json.dumps({'exit_code': code, 'elapsed_seconds': round(time.monotonic()-start, 3),
                      'cpu_seconds': round(usage.ru_utime+usage.ru_stime, 3),
                      'peak_rss_kib': usage.ru_maxrss}), flush=True)
    if code:
        raise SystemExit(code)


def policy():
    def git(*args):
        return subprocess.check_output(['git', *args], cwd=ROOT, text=True).strip()
    subprocess.run(['git', 'merge-base', '--is-ancestor', BASE, 'HEAD'], cwd=ROOT, check=True)
    commits = git('rev-list', BASE+'..HEAD').splitlines()
    assert commits, 'candidate must be committed'
    identity = 'Bryan Ehrlich <ehrlich.bryan@gmail.com>'
    for commit in commits:
        assert git('show', '-s', '--format=%an <%ae>', commit) == identity
        assert git('show', '-s', '--format=%cn <%ce>', commit) == identity
    assert not git('status', '--porcelain')
    subprocess.run(['git', 'diff', '--exit-code', BASE, 'HEAD', '--',
                    'engine/default_net.go', 'engine/nets', 'engine/evaluator_selection.go'], cwd=ROOT, check=True)
    print(json.dumps(dict(base=BASE, head=git('rev-parse', 'HEAD'), commits=len(commits),
                          author_committer=identity, embedded_default_unchanged=True,
                          host=os.uname().nodename, affinity=sorted(os.sched_getaffinity(0)),
                          claim='CPU fixtures only; no default-play change, speed or Elo claim.')), flush=True)


def main():
    cli = argparse.ArgumentParser(description=__doc__)
    cli.add_argument('check', choices=('python', 'parity', 'packet', 'policy', 'host', 'vet', 'short', 'diff-stack', 'diff-main'))
    args = cli.parse_args()
    setup()
    if args.check == 'python':
        execute([PYTHON, '-m', 'unittest', 'trainer.tests.test_residual_model'])
    elif args.check == 'packet':
        execute([PYTHON, str(HERE/'generate.py')])
    elif args.check == 'parity':
        execute(['go', 'test', '-p', '1', '-count=1', './engine', '-run', '^TestWave2ResidualParity', '-timeout', '15s', '-v'])
    elif args.check == 'policy':
        policy()
    elif args.check == 'host':
        print(json.dumps(dict(host=os.uname().nodename, kernel_or_device_checks='none required or performed',
                              affinity=sorted(os.sched_getaffinity(0)), gpu='unused')), flush=True)
        execute([PYTHON, '-c', 'import torch; torch.set_num_threads(2); assert torch.empty(1, device="cpu").device.type == "cpu"; print(torch.__version__, "CUDA build", torch.version.cuda, "CPU selected; threads", torch.get_num_threads())'])
    elif args.check == 'vet':
        execute(['go', 'vet', '-p', '2', './...'])
    elif args.check == 'short':
        if os.uname().nodename == 'forge':
            print('BLOCKED: run the registered short gate on a suitable host.')
            raise SystemExit(75)
        execute(['go', 'test', '-p', '2', '-short', '-count=1', './...'])
    else:
        base = '8ba878cb03a7eb81ca465b3d562953d0f45e81c7' if args.check == 'diff-stack' else '3517a353a99bbe627a505ccc5463aadce047431b'
        execute(['git', 'diff', '--check', base, 'HEAD'])


if __name__ == '__main__':
    main()
