"""Run only the small CPU unittest fragment with explicit resource limits."""

import json
import os
from pathlib import Path
import re
import resource
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parent
REPO = ROOT.parents[2]


def main():
    if len(sys.argv) != 2:
        raise SystemExit('usage: run.sh /absolute/output-directory-outside-checkout')
    out = Path(sys.argv[1]).resolve()
    if out == REPO or REPO in out.parents:
        raise SystemExit('Execution evidence must be outside the checkout')
    out.mkdir(parents=True, exist_ok=True)
    expected_python = Path.home() / 'ngn-original-nnue-night-20261001/venv/bin/python'
    if Path(sys.executable).absolute() != expected_python:
        raise SystemExit('Use the existing rig interpreter: ' + str(expected_python))
    env = {k: v for k, v in os.environ.items()
           if not re.search('credential|token|secret|password', k, flags=re.I)}
    env.update(CUDA_VISIBLE_DEVICES='', OMP_NUM_THREADS='2', OPENBLAS_NUM_THREADS='2',
               MKL_NUM_THREADS='2', NUMEXPR_NUM_THREADS='2', GOMAXPROCS='2',
               PYTHONDONTWRITEBYTECODE='1', NGN_QAT_REPORT=str(out / 'result.json'))
    if not env.get('TMPDIR') or not Path(env['TMPDIR']).is_dir():
        raise SystemExit('Set TMPDIR to an existing writable scratch directory')
    cores = sorted(os.sched_getaffinity(0))[:2]
    os.sched_setaffinity(0, cores)

    def limit_cpu():
        resource.setrlimit(resource.RLIMIT_CPU, (10, 10))

    command = [str(expected_python), '-m', 'unittest', 'trainer.tests.test_quantized_forward', '-v']
    print(json.dumps({'command': command, 'cpu_affinity': cores, 'cpu_seconds_limit': 10,
                      'CUDA_VISIBLE_DEVICES': '', 'OMP_NUM_THREADS': 2,
                      'credential_variables_removed': True}), flush=True)
    started = time.perf_counter()
    process = subprocess.Popen(command, cwd=REPO, env=env, preexec_fn=limit_cpu)
    _, status, usage = os.wait4(process.pid, 0)
    process.returncode = os.waitstatus_to_exitcode(status)
    cost = {'command': command, 'exit_code': process.returncode, 'cpu_affinity': cores,
            'cpu_seconds_limit': 10, 'elapsed_seconds': time.perf_counter() - started,
            'user_seconds': usage.ru_utime, 'system_seconds': usage.ru_stime,
            'peak_rss_kib': usage.ru_maxrss, 'diagnostic_only': True}
    (out / 'cost.json').write_text(json.dumps(cost, indent=2) + '\n')
    print(json.dumps(cost, sort_keys=True), flush=True)
    if process.returncode:
        raise SystemExit(process.returncode if process.returncode > 0 else 1)
    measured = json.loads((out / 'result.json').read_text())
    if measured['tensor_threads'] != 2:
        raise SystemExit('Expected exactly two tensor threads')
    # The frozen result is actual execution output. Duration and interpreter
    # versions are diagnostic observations, not reproducibility assertions.
    committed = ROOT / 'result.json'
    if committed.exists():
        expected = json.loads(committed.read_text())
        observations = {'qat_forward_backward_seconds_diagnostic_only', 'python', 'torch', 'numpy'}
        stable = lambda report: {k: v for k, v in report.items() if k not in observations}
        if stable(measured) != stable(expected):
            raise SystemExit('Execution differs from the committed fixture result')
        print('Committed fixture result reproduced.', flush=True)


if __name__ == '__main__':
    main()
