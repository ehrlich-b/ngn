"""One native Rodent-completion wait, then repaired-package certification."""
import ctypes
import datetime
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import select
import subprocess
import time

p = Path.cwd()
rodent = p.parent / 'session68-rodent'


def write(name, value):
    (p / name).write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (p / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(
            utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session71-provenance-repaired', event=event, detail=detail)) + '\n')


def digest(name):
    return hashlib.sha256((p / name).read_bytes()).hexdigest()


def commands():
    common = ['taskset', '-c', '0-11', 'nice', '-n', '0', str(p / 'sprt'),
              '-new', str(p / 'baseline-ngn'), '-base', str(p / 'baseline-ngn'),
              '-tc', '60+0.6', '-concurrency', '6', '-maxmoves', '400',
              '-gametimeout', '900', '-openings', str(p / 'openings.txt'),
              '-lowpower=false', '-elo0', '0', '-elo1', '6',
              '-alpha', '.05', '-beta', '.05']
    vector = json.loads((p / 'vector.json').read_text())
    for side in ['new', 'base']:
        for name, value in dict(EvalFile=str(p / 'base.nnue'), UseNNUE='true',
                                Hash=64, Threads=1, OwnBook='false',
                                **vector).items():
            common += ['-' + side + 'option', str(name) + '=' + str(value)]
    aa = common + ['-mingames', '1000000', '-maxgames', '40']
    gate = list(common)
    gate[gate.index('-new') + 1] = str(p / 'candidate-ngn')
    gate += ['-mingames', '400', '-maxgames', '4000']
    return aa, gate


def prepare():
    nomination = json.loads((p / 'nomination.json').read_text())
    for name, expected in nomination['binary_inputs_sha256'].items():
        assert digest(name) == expected, name
    assert json.loads((p / 'baseline-readiness.json').read_text())['ready']
    aa, gate = commands()
    names = [*nomination['binary_inputs_sha256'], 'nomination.json', 'vector.json',
             'baseline-readiness.json', 'manifest.md', 'controller.sh', 'games.py',
             'aa-parser.py', 'paired-parser.py']
    hashes = {name: digest(name) for name in names}
    write('game-manifest.json', dict(
        created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        nomination=nomination, machine='Ryzen9800X3D / WSL2', os=platform.platform(),
        cpus='0-11', concurrency=6, nice=19, memory_max_bytes=3 * 1024**3,
        memory_swap_max=0, tc='60+0.6', go_version='go1.25.5 linux/amd64',
        environment=dict(GOMAXPROCS='1', GOAMD64='v3'),
        pgo_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',
        aa_argv=aa, gate_argv=gate, files_sha256=hashes,
        aa_timeout_seconds=10800, gate_timeout_seconds=172800,
        predecessor=str(rodent), automatic_adoption=False,
        decision='ownAA40 contains0; natural H1 AND printed/unrounded lower95>0 AND zero faults/intact inputs certifies; H0/cap requires diagnosis; faults unscored; no retries'))
    hashes['game-manifest.json'] = digest('game-manifest.json')
    (p / 'game-inputs.sha256').write_text(''.join(
        pin + '  ' + name + '\n' for name, pin in hashes.items()))


def wait_for_rodent():
    libc = ctypes.CDLL(None, use_errno=True)
    fd = libc.inotify_init1(os.O_CLOEXEC | os.O_NONBLOCK)
    assert fd >= 0
    try:
        assert libc.inotify_add_watch(fd, os.fsencode(rodent), 0x8 | 0x80) >= 0
        status('rodent-wait', 'native controller.done notification; no polling or engine load')
        deadline = time.monotonic() + 36 * 3600
        while not (rodent / 'controller.done').exists():
            remaining = deadline - time.monotonic()
            assert remaining > 0, 'predecessor has no terminal; left untouched'
            readable, _, _ = select.select([fd], [], [], remaining)
            assert readable, 'predecessor has no terminal; left untouched'
            os.read(fd, 65536)
    finally:
        os.close(fd)
    status('rodent-terminal', dict(
        done=(rodent / 'controller.done').read_text().strip(),
        exit=(rodent / 'controller.exit').read_text().strip(),
        score_used_for_admission=False))


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
    wait_for_rodent()
    if not capacity():
        return
    aa, gate = commands()
    for mode, argv, limit in [('aa', aa, 10800), ('gate', gate, 172800)]:
        with (p / ('input-check-before-' + mode + '.txt')).open('w') as checks:
            subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], stdout=checks, check=True)
        if mode == 'gate':
            (p / 'background-before-gate.txt').write_text(subprocess.check_output(
                ['ps', '-eo', 'pid,ppid,ni,rss,args'], text=True))
        status(mode + '-start', dict(cpus='0-11', concurrency=6, tc='60+0.6', nice=19))
        with (p / (mode + '.log')).open('w') as log:
            result = subprocess.run(['timeout', '--signal=TERM', '--kill-after=15s',
                                     str(limit), *argv], stdout=log, stderr=subprocess.STDOUT)
        with (p / ('input-check-after-' + mode + '.txt')).open('w') as checks:
            subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], stdout=checks, check=True)
        assert result.returncode == 0, (mode, 'operational failure', result.returncode)
        crashes = p / 'ngn_crashes.log'
        if crashes.exists():
            assert all('Crash Handler Initialized' in line or 'Crash log:' in line
                       for line in crashes.read_text().splitlines())
        parser = ['python3', 'aa-parser.py', 'gate', 'aa.log', '--gate-mode', 'aa'] if mode == 'aa' else ['python3', 'paired-parser.py']
        parsed = subprocess.run(parser, text=True, stdout=subprocess.PIPE, check=True)
        summary = json.loads(parsed.stdout)
        if mode == 'gate':
            lower = float(re.search(r'Penta Elo:.*?95% CI \[([-+\d.]+),',
                                    (p / 'gate.log').read_text()).group(1))
            summary['printed_lower95'] = lower
            summary['verdict'] = 'CERTIFIED' if (
                summary['harness_verdict'].startswith('H1 ACCEPTED') and
                lower > 0 and summary['exact_elo_ci95'][1] > 0) else 'REQUIRES DIAGNOSIS'
        write(mode + '-summary.json', summary)
        status(mode + '-done', summary)


if __name__ == '__main__':
    if len(os.sys.argv) > 1 and os.sys.argv[1] == 'prepare':
        prepare()
    else:
        run()
