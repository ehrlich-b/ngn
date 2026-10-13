"""Prospective session77 SEE gate after clean horizon shelf; no adoption."""
import ctypes
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import re
import select
import subprocess
import time

p = Path.cwd()
predecessor = p.parent / 'session75-horizon-gate'


def write(name, value):
    (p / name).write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (p / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(
            utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session77-see-rig', event=event, detail=detail)) + '\n')


def digest(name):
    return hashlib.sha256((p / name).read_bytes()).hexdigest()


def prepare():
    nomination = json.loads((p / 'nomination.json').read_text())
    for name, expected in nomination['binary_inputs_sha256'].items():
        assert digest(name) == expected, name
    common = ['taskset', '-c', '0-11', 'nice', '-n', '0', str(p / 'sprt'),
              '-new', str(p / 'baseline-ngn'), '-base', str(p / 'baseline-ngn'),
              '-tc', '10+0.1', '-concurrency', '6', '-maxmoves', '400',
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
    hashes = {name: digest(name) for name in names}
    write('game-manifest.json', dict(
        created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        nomination=nomination, machine='Ryzen9800X3D / WSL2', os=platform.platform(),
        cpus='0-11', concurrency=6, nice=19, memory_max_bytes=3 * 1024**3,
        memory_swap_max=0, tc='10+0.1', go_version='go1.25.5 linux/amd64',
        environment=dict(GOMAXPROCS='1', GOAMD64='v3'),
        pgo_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',
        aa_argv=aa, gate_argv=gate, files_sha256=hashes,
        aa_timeout_seconds=7200, gate_timeout_seconds=21600,
        predecessor=str(predecessor), predecessor_required='clean horizon shelf plus manual actual-base review',
        decision='ownAA40 contains0; fixed400/200 printed AND unrounded lower95>0, zero faults/intact pins; else shelf; operational failure unscored',
        automatic_adoption=False))
    hashes['game-manifest.json'] = digest('game-manifest.json')
    (p / 'game-inputs.sha256').write_text(''.join(
        pin + '  ' + name + '\n' for name, pin in hashes.items()))


def wait_for_predecessor():
    assert (predecessor / 'controller.exit').read_text().strip() == '0'
    gate = json.loads((predecessor / 'gate-summary.json').read_text())
    assert gate['verdict'] == 'shelved' and gate['games'] == 400
    review = json.loads((p / 'nomination.json').read_text())
    assert review['manual_actual_base_review'] == 'f1d0a03 unchanged after horizon shelf'
    status('predecessor-review', dict(horizon='clean full shelf', actual_base='f1d0a03',
                                    plan='October12 22:55 prospective SEE nomination'))
    return True


def capacity():
    processes = subprocess.check_output(['ps', '-eo', 'pid,ppid,ni,rss,args'], text=True)
    (p / 'background-before-aa.txt').write_text(processes)
    busy = [line for line in processes.splitlines()
            if re.search(r'trainer\.train|(?:^|/)sprt(?:\s|$)|(?:^|/)datagen(?:\s|$)|(?:^|/)pairmatch(?:\s|$)', line)]
    memory = {name: int(value) * 1024 for name, value in
              re.findall(r'^(MemTotal|MemAvailable):\s+(\d+) kB', Path('/proc/meminfo').read_text(), re.M)}
    used = memory['MemTotal'] - memory['MemAvailable']
    disk = os.statvfs('/mnt/c')
    c_free = disk.f_bavail * disk.f_frsize
    receipt = dict(nonavailable_memory_bytes=used, new_job_budget_bytes=3 * 1024**3,
                   total_budget_limit_bytes=12 * 1024**3, c_free_bytes=c_free,
                   competing_chess_processes=busy)
    write('capacity-before-aa.json', receipt)
    if busy or c_free < 25.5 * 1024**3 or used + 3 * 1024**3 > 12 * 1024**3:
        status('capacity-hold', receipt)
        return False
    return True


def run():
    if not wait_for_predecessor() or not capacity():
        return
    manifest = json.loads((p / 'game-manifest.json').read_text())
    spec = importlib.util.spec_from_file_location('parser', p / 'paired-parser.py')
    parser = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(parser)
    for mode in ['aa', 'gate']:
        with (p / ('input-check-before-' + mode + '.txt')).open('w') as checks:
            subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], stdout=checks, check=True)
        status(mode + '-start', dict(cpus='0-11', concurrency=6, tc='10+0.1', nice=19))
        with (p / (mode + '.log')).open('w') as log:
            result = subprocess.run(['timeout', '--signal=TERM', '--kill-after=15s',
                                     str(manifest[mode + '_timeout_seconds']),
                                     *manifest[mode + '_argv']], stdout=log, stderr=subprocess.STDOUT)
        with (p / ('input-check-after-' + mode + '.txt')).open('w') as checks:
            subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], stdout=checks, check=True)
        assert result.returncode == 0, (mode, 'operational failure', result.returncode)
        crashes = p / 'ngn_crashes.log'
        if crashes.exists():
            assert all('Crash Handler Initialized' in line or 'Crash log:' in line
                       for line in crashes.read_text().splitlines())
        summary = parser.gate((p / (mode + '.log')).read_text(), mode)
        write(mode + '-summary.json', summary)
        status(mode + '-done', summary)


if __name__ == '__main__':
    if len(os.sys.argv) > 1 and os.sys.argv[1] == 'prepare':
        prepare()
    else:
        run()
