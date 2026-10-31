"""One manually nominated fixed data gate; no automatic adoption."""
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


def status(event, detail):
    with (p / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(
            utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session66-data-gate', event=event, detail=detail)) + '\n')


expected = {
    'ngn': '4f8f631fe29eb010605b6eaec2d8c1fac769e39c416f3ff7f41e997feae7a61d',
    'base.nnue': 'a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86',
    'candidate.nnue': '4636de46c8afb33227cff72831d3fa80b86b1c84d9394661f70b29f10ddc945f',
    'sprt': '9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a',
    'openings.txt': '974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222',
    'paired-parser.py': 'ed10e8511cc6a42e754f9f30407bbcfc4c4f24f38ab3568ea16d0036e725aea7',
}
assert all(sha(name) == digest for name, digest in expected.items())
receipt = json.loads((p / 'training-receipt.json').read_text())
assert receipt['epochs'] == 58 and receipt['total_positions'] == 14848000000
assert receipt['best_epoch'] == 58 and receipt['reexport_identical']
assert receipt['parity_cases'] == 20
assert receipt['net_sha256'] == expected['candidate.nnue']
spec = importlib.util.spec_from_file_location('parser', p / 'paired-parser.py')
parser = importlib.util.module_from_spec(spec)
spec.loader.exec_module(parser)
aa = parser.gate((p / 'aa.log').read_text(), 'aa')
assert aa == json.loads((p / 'aa-summary.json').read_text())
for name in ['aa-input-check-before.txt', 'aa-input-check-after.txt']:
    lines = (p / name).read_text().splitlines()
    assert len(lines) == 9 and all(line.endswith(': OK') for line in lines)
aa_manifest = json.loads((p / 'aa-manifest.json').read_text())
assert all(aa_manifest['files_sha256'][name] == digest
           for name, digest in expected.items() if name != 'candidate.nnue')
assert (aa_manifest['tc'], aa_manifest['cpus'], aa_manifest['concurrency'],
        aa_manifest['host_pinning']) == ('10+0.1', '4-11', 2, 'on')
argv = ['taskset', '-c', '4-11', 'nice', '-n', '10', str(p / 'sprt'),
        '-new', str(p / 'ngn'), '-base', str(p / 'ngn'), '-tc', '10+0.1',
        '-concurrency', '2', '-maxmoves', '400', '-gametimeout', '180',
        '-openings', str(p / 'openings.txt'), '-lowpower=false',
        '-mingames', '1000000', '-maxgames', '400']
for side, net in [('new', 'candidate.nnue'), ('base', 'base.nnue')]:
    for key, value in [('EvalFile', str(p / net)), ('UseNNUE', 'true'),
                       ('Hash', '64'), ('Threads', '1'), ('OwnBook', 'false')]:
        argv += ['-' + side + 'option', key + '=' + value]
names = [*expected, 'training-receipt.json', 'aa.log', 'aa-summary.json',
         'aa-manifest.json', 'aa-input-check-before.txt', 'aa-input-check-after.txt',
         'gate.py', 'controller.sh', 'manifest.md']
hashes = {name: sha(name) for name in names}
manifest = dict(
    created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
    source='f1d0a03/topic5023b1d exact engine on BOTH sides',
    change='net only: expanded corpus step2, unchanged training recipe',
    machine='forge Debian13/Ryzen6900HX', os=platform.platform(),
    host_pinning='on', cpus='4-11', concurrency=2, nice=10,
    memory_max_bytes=3 * 1024**3, memory_swap_max=0, tc='10+0.1',
    environment=dict(GOMAXPROCS='1', GOAMD64='v3'),
    go_version='go version go1.25.5 linux/amd64',
    profile_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',
    aa_reused_from='/work/lanes/ngn-s65-data-aa', aa=aa, gate_argv=argv,
    files_sha256=hashes, operational_timeout_seconds=21600,
    background='immutable readers; trainer terminal; rig tune-transfer separate',
    decision='fixed400/200; printed AND unrounded lower95>0; zero faults/intact pins; otherwise shelf; incomplete unscored')
(p / 'game-manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
hashes['game-manifest.json'] = sha('game-manifest.json')
(p / 'game-inputs.sha256').write_text(''.join(
    digest + '  ' + name + '\n' for name, digest in hashes.items()))
processes = subprocess.check_output(['ps', '-eo', 'pid,ppid,psr,pcpu,rss,args'], text=True).splitlines()
(p / 'background-before-gate.txt').write_text('\n'.join(
    [processes[0], *[line for line in processes[1:] if '/work/lanes/ngn-' in line]]) + '\n')
with (p / 'input-check-before-gate.txt').open('w') as stream:
    subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], check=True, stdout=stream)
status('gate-start', 'manual selected4636de46 vs a4d90b41; final14 identical both sides; ownAA40 reused')
with (p / 'gate.log').open('w') as stream:
    subprocess.run(['timeout', '--signal=TERM', '--kill-after=15s', '21600', *argv],
                   check=True, stdout=stream, stderr=subprocess.STDOUT,
                   env=dict(os.environ, GOMAXPROCS='1', GOAMD64='v3'))
result = parser.gate((p / 'gate.log').read_text(), 'gate')
crashes = p / 'ngn_crashes.log'
if crashes.exists():
    assert all('Crash Handler Initialized' in line or 'Crash log:' in line
               for line in crashes.read_text().splitlines())
(p / 'gate-summary.json').write_text(json.dumps(result, indent=2) + '\n')
with (p / 'input-check-after-gate.txt').open('w') as stream:
    subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], check=True, stdout=stream)
status('gate-done', result)
