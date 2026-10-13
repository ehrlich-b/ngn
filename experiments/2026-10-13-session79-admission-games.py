"""Native SMP completion -> changed-base checks -> original isolated clock policy."""
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
predecessor = p.parent / 'session77-helper-depths-gate'


def write(name, value):
    (p / name).write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (p / 'status.jsonl').open('a') as out:
        out.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                 event=event, detail=detail)) + '\n')


def digest(name):
    return hashlib.sha256((p / name).read_bytes()).hexdigest()


def wait():
    status('predecessor-wait', str(predecessor / 'controller.done') + ' native completion; no polling')
    if not (predecessor / 'controller.done').exists():
        libc = ctypes.CDLL(None)
        fd = libc.inotify_init1(os.O_NONBLOCK | os.O_CLOEXEC)
        assert fd >= 0 and libc.inotify_add_watch(fd, os.fsencode(predecessor), 0x8 | 0x80) >= 0
        try:
            deadline = time.monotonic() + 24 * 3600
            while not (predecessor / 'controller.done').exists():
                remaining = deadline - time.monotonic()
                assert remaining > 0 and select.select([fd], [], [], remaining)[0]
                os.read(fd, 65536)
        finally:
            os.close(fd)
    gate_file = predecessor / 'gate-summary.json'
    gate = json.loads(gate_file.read_text()) if gate_file.exists() else {}
    allowed = ((predecessor / 'controller.exit').read_text().strip() == '0'
               and gate.get('verdict') == 'shelved' and gate.get('games') == 400
               and gate.get('pairs') == 200 and gate.get('flags') == [0, 0])
    status('predecessor-review', dict(allowed=allowed, gate=gate,
                                    action='SEE-base admission only' if allowed else 'manual actual-base hold'))
    return allowed


def capacity():
    processes = subprocess.check_output(['ps', '-eo', 'pid,ppid,rss,args'], text=True)
    busy = [line for line in processes.splitlines()
            if re.search(r'trainer\.train|(?:^|/)sprt(?:\s|$)|(?:^|/)datagen(?:\s|$)|(?:^|/)pairmatch(?:\s|$)', line)]
    memory = {k: int(v) * 1024 for k, v in
              re.findall(r'^(MemTotal|MemAvailable):\s+(\d+) kB', Path('/proc/meminfo').read_text(), re.M)}
    disk = os.statvfs('/mnt/c')
    receipt = dict(used_bytes=memory['MemTotal'] - memory['MemAvailable'],
                   budget_bytes=3 * 1024**3, c_free_bytes=disk.f_bavail * disk.f_frsize, busy=busy)
    write('capacity-before-checks.json', receipt)
    allowed = not busy and receipt['c_free_bytes'] >= 25.5 * 1024**3 and receipt['used_bytes'] + receipt['budget_bytes'] <= 12 * 1024**3
    if not allowed:
        status('capacity-hold', receipt)
    return allowed


def prepare():
    common = ['taskset', '-c', '0-11', str(p / 'sprt'), '-new', str(p / 'baseline-ngn'),
              '-base', str(p / 'baseline-ngn'), '-tc', '120+1', '-concurrency', '6',
              '-maxmoves', '400', '-gametimeout', '1800', '-openings', str(p / 'openings.txt'),
              '-lowpower=false', '-mingames', '1000000']
    for side in ['new', 'base']:
        for key, value in [('EvalFile', str(p / 'base.nnue')), ('UseNNUE', 'true'),
                           ('Hash', '64'), ('Threads', '1'), ('OwnBook', 'false'), ('Move Overhead', '100')]:
            common += ['-' + side + 'option', key + '=' + value]
    aa = common + ['-maxgames', '40']
    gate = list(common)
    gate[gate.index('-new') + 1] = str(p / 'candidate-ngn')
    gate += ['-maxgames', '400']
    files = ['baseline-ngn', 'candidate-ngn', 'base.nnue', 'sprt', 'openings.txt',
             'paired-parser.py', 'games.py', 'controller.sh', 'manifest.md', 'checks.sh',
             'candidate.patch', 'nomination.json', 'source.tar', 'candidate-inputs.sha256']
    hashes = {name: digest(name) for name in files}
    nomination = json.loads((p / 'nomination.json').read_text())
    for name, expected in nomination['files_sha256'].items():
        assert digest(name) == expected, name
    write('game-manifest.json', dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        base_source='f203e2f5c37861fee324a1cf486b5bfd5f6308d7',
        candidate_source='f203e2f plus original session69 admission patch',
        change='immediate per-iteration clock admission only',
        net='a4d90b41', final14='compiled identical defaults',
        machine='rig Ryzen9800X3D WSL2', os=platform.platform(),
        tc='120+1', concurrency=6, cpus='0-11', nice=19, GOMAXPROCS=1, GOAMD64='v3',
        go_version=(p / 'environment.txt').read_text().strip(),
        pgo_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',
        memory_max_bytes=3 * 1024**3, memory_swap_max=0,
        aa_argv=aa, gate_argv=gate, aa_timeout_seconds=10800, gate_timeout_seconds=43200,
        files_sha256=hashes, predecessor='clean full helper-depth shelf on original f1d0a03; SEE correction separately kept; changed-base checks PASS',
        decision='ownAA40 printed/exact contains0; fixed400/200 printed/exact lower95>0, zero faults/intact inputs; otherwise shelf; faults/control failure unscored',
        automatic_adoption=False, extension=False, pooling=False, claim=False))
    hashes['game-manifest.json'] = digest('game-manifest.json')
    (p / 'game-inputs.sha256').write_text(''.join(pin + '  ' + name + '\n' for name, pin in hashes.items()))


def run():
    if not wait() or not capacity():
        return
    status('checks-start', dict(cpus=[13, 15], base='f203e2f', candidate='same base plus admission only'))
    subprocess.run(['systemd-run', '--user', '--scope', '-p', 'AllowedCPUs=13,15',
                    '-p', 'CPUQuota=200%', '-p', 'MemoryMax=2G', '-p', 'MemorySwapMax=0',
                    'taskset', '-c', '13,15', 'bash', str(p / 'checks.sh')], check=True)
    assert (p / 'checks.exit').read_text().strip() == '0'
    status('checks-done', 'focused, engine short/race, full short/race, vet, PGO build PASS')
    prepare()
    spec = importlib.util.spec_from_file_location('existing_parser', p / 'paired-parser.py')
    parser = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(parser)
    manifest = json.loads((p / 'game-manifest.json').read_text())
    for mode in ['aa', 'gate']:
        with (p / ('input-check-before-' + mode + '.txt')).open('w') as out:
            subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], stdout=out, check=True)
        status(mode + '-start', dict(tc='120+1', cpus='0-11', concurrency=6, nice=19))
        with (p / (mode + '.log')).open('w') as out:
            result = subprocess.run(['timeout', '--signal=TERM', '--kill-after=15s',
                                     str(manifest[mode + '_timeout_seconds']), *manifest[mode + '_argv']],
                                    stdout=out, stderr=subprocess.STDOUT)
        with (p / ('input-check-after-' + mode + '.txt')).open('w') as out:
            subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], stdout=out, check=True)
        assert result.returncode == 0, (mode, 'operationally unscored', result.returncode)
        crashes = p / 'ngn_crashes.log'
        if crashes.exists():
            assert all('Crash Handler Initialized' in line or 'Crash log:' in line for line in crashes.read_text().splitlines())
        summary = parser.gate((p / (mode + '.log')).read_text(), mode)
        write(mode + '-summary.json', summary)
        status(mode + '-done', summary)


if __name__ == '__main__':
    run()
