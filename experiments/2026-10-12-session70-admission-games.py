"""Nominate the already checked immediate-admission change on final14."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess

p = Path.cwd()


def digest(name):
    return hashlib.sha256((p / name).read_bytes()).hexdigest()


def write(name, value):
    (p / name).write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def status(event, detail):
    with (p / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(
            utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session70-admission-final14', event=event, detail=detail)) + '\n')


def main():
    nomination = json.loads((p / 'nomination.json').read_text())
    checks = json.loads((p / 'checks-review.json').read_text())
    assert nomination['base_source'] == checks['base_source']
    assert checks['tested_source_matches_worktree']
    assert nomination['binary_inputs_sha256']['candidate-ngn'] == checks['candidate_binary_sha256']
    for name, pin in nomination['binary_inputs_sha256'].items():
        assert digest(name) == pin, name
    common = ['taskset', '-c', '4-11', 'nice', '-n', '0', str(p / 'sprt'),
              '-new', str(p / 'baseline-ngn'), '-base', str(p / 'baseline-ngn'),
              '-tc', '60+0.6', '-concurrency', '2', '-maxmoves', '400',
              '-gametimeout', '900', '-openings', str(p / 'openings.txt'),
              '-lowpower=false']
    for side in ['new', 'base']:
        for key, value in [('EvalFile', str(p / 'base.nnue')), ('UseNNUE', 'true'),
                           ('Hash', '64'), ('Threads', '1'), ('OwnBook', 'false')]:
            common += ['-' + side + 'option', key + '=' + value]
    aa = common + ['-mingames', '1000000', '-maxgames', '40']
    gate = list(common)
    gate[gate.index('-new') + 1] = str(p / 'candidate-ngn')
    gate += ['-elo0', '0', '-elo1', '6', '-alpha', '.05', '-beta', '.05',
             '-mingames', '400', '-maxgames', '4000']
    names = [*nomination['binary_inputs_sha256'], 'nomination.json',
             'candidate.patch', 'checks-review.json', 'aa-parser.py',
             'paired-parser.py', 'gate-review.py', 'games.py', 'controller.sh', 'manifest.md']
    hashes = {name: digest(name) for name in names}
    write('game-manifest.json', dict(
        created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        nomination=nomination, machine='forge Debian13/Ryzen6900HX',
        os=platform.platform(), host_pinning='plan ON; unverified after 20:37 reboot',
        guest_vcpus=os.cpu_count(), cpus='4-11', concurrency=2, nice=10,
        memory_max_bytes=3 * 1024**3, memory_swap_max=0, tc='60+0.6',
        environment=dict(GOMAXPROCS='1', GOAMD64='v3'),
        go_version=checks['go_version'], build_machine='forge',
        pgo_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',
        aa_argv=aa, gate_argv=gate, files_sha256=hashes,
        aa_timeout_seconds=14400, gate_timeout_seconds=259200,
        decision='ownAA40 then SPRT[0,+6]/min400/cap4000; H1 AND printed/unrounded lower95>0; zero faults/intact inputs; otherwise shelf; faults unscored',
        automatic_adoption=False))
    hashes['game-manifest.json'] = digest('game-manifest.json')
    (p / 'game-inputs.sha256').write_text(''.join(
        pin + '  ' + name + '\n' for name, pin in hashes.items()))
    (p / 'background-before-aa.txt').write_text(subprocess.check_output(
        ['ps', '-eo', 'pid,ppid,psr,pcpu,rss,args'], text=True))
    for mode, argv, limit in [('aa', aa, 14400), ('gate', gate, 259200)]:
        with (p / ('input-check-before-' + mode + '.txt')).open('w') as checks_stream:
            subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'],
                           stdout=checks_stream, check=True)
        status(mode + '-start', dict(cpus='4-11', concurrency=2, nice=10,
               tc='60+0.6', automatic_adoption=False))
        with (p / (mode + '.log')).open('w') as log:
            subprocess.run(['timeout', '--signal=TERM', '--kill-after=15s',
                            str(limit), *argv], stdout=log,
                           stderr=subprocess.STDOUT, check=True)
        if mode == 'aa':
            with (p / 'aa-summary.json').open('w') as summary:
                subprocess.run(['python3', 'aa-parser.py', 'gate', 'aa.log',
                                '--gate-mode', 'aa'], stdout=summary, check=True)
        else:
            subprocess.run(['python3', 'gate-review.py'], check=True)
        crashes = p / 'ngn_crashes.log'
        if crashes.exists():
            assert all('Crash Handler Initialized' in line or 'Crash log:' in line
                       for line in crashes.read_text().splitlines())
        with (p / ('input-check-after-' + mode + '.txt')).open('w') as checks_stream:
            subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'],
                           stdout=checks_stream, check=True)
        status(mode + '-done', json.loads((p / (mode + '-summary.json')).read_text()))


if __name__ == '__main__':
    main()
