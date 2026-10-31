"""Frozen singular fail-low correction gate on the actual final14 playing package."""
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import subprocess

p = Path.cwd()
sha = lambda name: hashlib.sha256((p / name).read_bytes()).hexdigest()


def write(name, value):
    (p / name).write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (p / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                    stage='session69-singular-correction', event=event, detail=detail)) + '\n')


def main():
    nomination = json.loads((p / 'nomination.json').read_text())
    for name, pin in nomination['binary_inputs_sha256'].items():
        assert sha(name) == pin, name
    # The scope launcher already sets nice10; retain that actual priority.
    common = ['taskset', '-c', '4-11', 'nice', '-n', '0', str(p / 'sprt'),
              '-new', str(p / 'baseline-ngn'), '-base', str(p / 'baseline-ngn'),
              '-tc', '10+0.1', '-concurrency', '2', '-maxmoves', '400',
              '-gametimeout', '180', '-openings', str(p / 'openings.txt'),
              '-lowpower=false', '-mingames', '1000000']
    for side in ['new', 'base']:
        for key, value in [('EvalFile', str(p / 'base.nnue')), ('UseNNUE', 'true'),
                           ('Hash', '64'), ('Threads', '1'), ('OwnBook', 'false')]:
            common += ['-' + side + 'option', key + '=' + value]
    aa = common + ['-maxgames', '40']
    gate = list(common)
    gate[gate.index('-new') + 1] = str(p / 'candidate-ngn')
    gate += ['-maxgames', '400']
    names = [*nomination['binary_inputs_sha256'], 'nomination.json', 'candidate.patch',
             'checks-review.json', 'paired-parser.py', 'games.py', 'controller.sh', 'manifest.md']
    hashes = {name: sha(name) for name in names}
    write('game-manifest.json', dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
          nomination=nomination, machine='forge Debian13/Ryzen6900HX', os=platform.platform(),
          host_pinning='plan ON; unverified after20:37 reboot; guest now16vCPU', cpus='4-11', concurrency=2, nice=10, memory_max_bytes=3 * 1024**3,
          memory_swap_max=0, tc='10+0.1', environment=dict(GOMAXPROCS='1', GOAMD64='v3'),
          build_machine='rig WSL2', go_version='go1.25.5 linux/amd64',
          pgo_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',
          aa_argv=aa, gate_argv=gate, files_sha256=hashes,
          aa_timeout_seconds=7200, gate_timeout_seconds=21600,
          decision='ownAA40 contains0; fixed400/200 printed AND unrounded lower95>0, zero faults/intact pins; else shelf; operational failure unscored',
          automatic_adoption=False))
    hashes['game-manifest.json'] = sha('game-manifest.json')
    (p / 'game-inputs.sha256').write_text(''.join(pin + '  ' + name + '\n' for name, pin in hashes.items()))
    spec = importlib.util.spec_from_file_location('parser', p / 'paired-parser.py')
    parser = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(parser)
    (p / 'background-before-aa.txt').write_text(subprocess.check_output(
        ['ps', '-eo', 'pid,ppid,psr,pcpu,rss,args'], text=True))
    for mode, argv, limit in [('aa', aa, 7200), ('gate', gate, 21600)]:
        with (p / ('input-check-before-' + mode + '.txt')).open('w') as checks:
            subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], stdout=checks, check=True)
        status(mode + '-start', 'forge4-11/c2/3GiB/swap0/nice10/taskset; postreboot16vCPU; external load recorded; no competing NGN CPU jobs')
        with (p / (mode + '.log')).open('w') as log:
            subprocess.run(['timeout', '--signal=TERM', '--kill-after=15s', str(limit), *argv],
                           stdout=log, stderr=subprocess.STDOUT, check=True,
                           env=dict(os.environ, GOMAXPROCS='1', GOAMD64='v3'))
        result = parser.gate((p / (mode + '.log')).read_text(), mode)
        crashes = p / 'ngn_crashes.log'
        if crashes.exists():
            assert all('Crash Handler Initialized' in line or 'Crash log:' in line
                       for line in crashes.read_text().splitlines())
        with (p / ('input-check-after-' + mode + '.txt')).open('w') as checks:
            subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], stdout=checks, check=True)
        write(mode + '-summary.json', result)
        status(mode + '-done', result)


if __name__ == '__main__':
    main()
