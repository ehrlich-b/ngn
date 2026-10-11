"""One post-conversion forge A/A200, then the frozen final-vector gate if admitted."""
import ctypes
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import select
import subprocess
import time
from zoneinfo import ZoneInfo

p = Path.cwd()
data = Path('/work/lanes/ngn-data-step2-20261011')
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
now = lambda: datetime.datetime.now(datetime.timezone.utc).isoformat()


def status(event, detail):
    with (p / 'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=now(), stage='session61-forge-games', event=event, detail=detail)) + '\n')


def write(name, value):
    (p / name).write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def wait_conversion():
    # Subscribe before examining the receipt, avoiding a completion race.
    libc = ctypes.CDLL(None, use_errno=True)
    fd = libc.inotify_init1(os.O_CLOEXEC)
    assert fd >= 0
    try:
        assert libc.inotify_add_watch(fd, os.fsencode(data), 0x8 | 0x80) >= 0
        poll = select.poll()
        poll.register(fd, select.POLLIN)
        deadline = time.monotonic() + 12 * 3600
        while not (data / 'recovery-session59.done').exists():
            remaining = deadline - time.monotonic()
            assert remaining > 0 and poll.poll(int(remaining * 1000)), 'conversion wait expired'
            os.read(fd, 65536)
    finally:
        os.close(fd)
    assert (data / 'recovery-session59.exit').read_text().strip() == '0', 'data stage failed; no games'
    coverage = json.loads((data / 'coverage.json').read_text())
    assert (data / 't74.archive.json').exists()
    return coverage


def parse(mode):
    name = 'aa200-parser.py' if mode == 'aa' else 'paired-parser.py'
    spec = importlib.util.spec_from_file_location('parser', p / name)
    parser = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(parser)
    result = parser.gate((p / (mode + '.log')).read_text(), mode)
    crashes = p / 'ngn_crashes.log'
    if crashes.exists():
        assert all('Crash Handler Initialized' in line or 'Crash log:' in line
                   for line in crashes.read_text().splitlines()), 'unexpected crash log'
    write(mode + '-summary.json', result)
    return result


def main():
    expected_vector = dict(NullMoveR=4, NullMoveMinDepth=2, LMRFullDepth=3, LMRMoveThreshold=2,
                          FutilityMargin=104, FutilityMaxDepth=8, ReverseFutilityMargin=63,
                          DeltaMargin=202, SingularDepth=7, SingularMargin=57,
                          ExtensionBudget=29, AspInit=17, AspMult=163, LMPBase=3)
    expected = dict(ngn='c14f03d8db4d00a3c43a1944e389b60cd5c52b8649c665e370cea1dcb3c0b848',
                    **{'base.nnue': 'a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86',
                       'sprt': '9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a',
                       'openings.txt': '974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'})
    for name, pin in expected.items():
        assert sha(p / name) == pin, name
    vector = json.loads((p / 'vector.json').read_text())
    live = json.loads((p / 'live-receipt.json').read_text())
    endpoint = json.loads((p / 'endpoint.json').read_text())
    assert vector == expected_vector == live['endpoint']['rounded'] == endpoint['rounded']
    assert live['source'] == 'ed34480' and live['searches'] == 12 and live['normal_exits']
    assert endpoint['iter'] == 2000 and live['handler_test'] == 'PASS all14 pointers match exact final values'
    assert live['binary_sha256'] == expected['ngn'] and live['base_net_sha256'] == expected['base.nnue']
    common = ['taskset', '-c', '4-11', 'nice', '-n', '10', str(p / 'sprt'),
              '-new', str(p / 'ngn'), '-base', str(p / 'ngn'), '-tc', '10+0.1',
              '-concurrency', '2', '-maxmoves', '400', '-gametimeout', '180',
              '-openings', str(p / 'openings.txt'), '-lowpower=false', '-mingames', '1000000']
    for side in ['new', 'base']:
        for key, value in [('EvalFile', str(p / 'base.nnue')), ('UseNNUE', 'true'),
                           ('Hash', '64'), ('Threads', '1'), ('OwnBook', 'false')]:
            common += ['-' + side + 'option', key + '=' + value]
    aa = common + ['-maxgames', '200']
    gate = common + ['-maxgames', '400']
    for key, value in vector.items():
        gate += ['-newoption', key + '=' + str(value)]
    hashes = {name: sha(p / name) for name in [*expected, 'vector.json', 'endpoint.json',
              'live-receipt.json', 'games.py', 'controller.sh', 'aa200-parser.py', 'paired-parser.py', 'manifest.md']}
    write('game-manifest.json', dict(created_utc=now(), base_source='ed34480',
          base_net_sha256=expected['base.nnue'], candidate='final SPSA14 options only',
          source_checks='session56 accepted-package handler and12 bounded searches; already PASS',
          machine='forge Debian13/Ryzen6900HX', os=platform.platform(), host_pinning='on',
          cpus='4-11', concurrency=2, nice=10, memory_max_bytes=3*1024**3, memory_swap_max=0,
          environment=dict(GOMAXPROCS='1', GOAMD64='v3'), go_version='go version go1.25.5 linux/amd64',
          pgo_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',
          aa_argv=aa, gate_argv=gate, aa_timeout_seconds=7200, gate_timeout_seconds=14400,
          gate_launch_before_edt='2026-10-11T12:00:00-04:00', finalizer_reserved_from_edt='2026-10-11T16:00:00-04:00',
          files_sha256=hashes, vector=vector, automatic_promotion=False,
          decision='A/A200 interval contains0,zero faults,intact pins; fixed400 gate printed AND unrounded lower95>0/clean pins only; no retry/pooling',
          conditions='native conversion/coverage completion first; finish admitted timed sample before any parity/source work'))
    hashes['game-manifest.json'] = sha(p / 'game-manifest.json')
    (p / 'game-inputs.sha256').write_text(''.join(pin + '  ' + name + '\n' for name, pin in hashes.items()))
    status('conversion-wait', 'native completion notification; no competing CPU work; A/A200 then conditional final14 gate')
    coverage = wait_conversion()
    write('conversion-ready.json', dict(utc=now(), coverage=coverage))
    processes = subprocess.check_output(['ps', '-eo', 'pid,ppid,psr,pcpu,rss,args'], text=True).splitlines()
    (p / 'background-before-aa.txt').write_text('\n'.join([processes[0], *[line for line in processes[1:] if '/work/lanes/ngn-' in line]]) + '\n')
    write('capacity-before-aa.json', dict(utc=now(), load_average=os.getloadavg(),
          memory=subprocess.check_output(['free', '-m'], text=True), host_pinning='on'))
    for mode, argv, seconds in [('aa', aa, 7200), ('gate', gate, 14400)]:
        if mode == 'gate':
            cutoff = datetime.datetime(2026, 10, 11, 12, 0, tzinfo=ZoneInfo('America/New_York'))
            if datetime.datetime.now(datetime.timezone.utc) >= cutoff:
                write('gate-window-held.json', dict(utc=now(), reason='reserve forge finalizer capacity from16:00; manual admission needed'))
                status('gate-window-held', 'A/A passed; SPSA not started after12:00; review next finalizer window')
                return
        subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], check=True,
                       stdout=(p / ('input-check-before-' + mode + '.txt')).open('w'))
        status(mode + '-start', 'forge4-11/c2/nice10/host pinningON; no concurrent NGN conversion, tests or parity')
        with (p / (mode + '.log')).open('w') as log:
            subprocess.run(['timeout', '--signal=TERM', '--kill-after=15s', str(seconds), *argv],
                           check=True, stdout=log, stderr=subprocess.STDOUT,
                           env=dict(os.environ, GOMAXPROCS='1', GOAMD64='v3'))
        result = parse(mode)
        subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], check=True,
                       stdout=(p / ('input-check-after-' + mode + '.txt')).open('w'))
        status(mode + '-done', json.dumps(result))


if __name__ == '__main__':
    main()
