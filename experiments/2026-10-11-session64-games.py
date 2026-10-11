"""Manual current-package SPSA nomination, own A/A40 then fixed400."""
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import subprocess
from zoneinfo import ZoneInfo

p = Path.cwd()
now = lambda: datetime.datetime.now(datetime.timezone.utc)
sha = lambda name: hashlib.sha256((p / name).read_bytes()).hexdigest()

def write(name, data):
    (p / name).write_text(json.dumps(data, indent=2, allow_nan=False) + '\n')

def status(event, detail):
    with (p / 'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=now().isoformat(), stage='session64-current-spsa', event=event, detail=detail)) + '\n')

def main():
    deadline = datetime.datetime(2026, 10, 11, 16, 0, tzinfo=ZoneInfo('America/New_York'))
    cutoff = deadline.replace(hour=13, minute=50)
    assert now() < cutoff, 'window held; do not start this sample'
    assert (p / 'checks.exit').read_text().strip() == '0'
    parity = json.loads((p / 'default-parity.json').read_text())
    assert parity['searches'] == 12 and parity['compiled_defaults_equal_current_overrides']
    assert parity['compiled_defaults_equal_accepted_old_source_tune'] and parity['advertised_defaults_match_all14']
    common = ['taskset', '-c', '4-11', 'nice', '-n', '10', str(p / 'sprt'),
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
    names = ['baseline-ngn', 'candidate-ngn', 'sprt', 'base.nnue', 'openings.txt',
             'vector.json', 'default-parity.json', 'checks.exit', 'paired-parser.py',
             'games.py', 'controller.sh', 'manifest.md', 'candidate.patch', 'build.sha256']
    hashes = {name: sha(name) for name in names}
    build_hashes = {line.split()[1]: line.split()[0] for line in (p / 'build.sha256').read_text().splitlines()}
    for name in ['baseline-ngn', 'candidate-ngn']:
        assert hashes[name] == build_hashes[name], 'copied binary differs from checked build'
    assert hashes['base.nnue'] == 'a4d90b41f40228a2ce709efb8d8809b8f69ff7566f81d8f9f1651e0e9b9a6e86'
    assert hashes['sprt'] == '9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a'
    assert hashes['openings.txt'] == '974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'
    write('game-manifest.json', dict(created_utc=now().isoformat(),base_source='14f7635/engine7bbb2bf',
          candidate='final14 compiled defaults only',candidate_source='5023b1d',candidate_patch_sha256=hashes['candidate.patch'],
          vector=json.loads((p / 'vector.json').read_text()),machine='forge Debian13/Ryzen6900HX',
          os=platform.platform(),host_pinning='on',cpus='4-11',concurrency=2,nice=10,
          memory_max_bytes=3*1024**3,memory_swap_max=0,environment=dict(GOMAXPROCS='1',GOAMD64='v3'),
          build_machine='rig WSL2 Ubuntu',go_version='go version go1.25.5 linux/amd64',pgo_sha256='616b1ebe276a6ee208f849bb1e48208973d98c6714bdebf7d8febd45cd5ee3f0',
          aa_argv=aa,gate_argv=gate,aa_timeout_seconds=1800,absolute_deadline_edt=deadline.isoformat(),
          files_sha256=hashes,decision='ownAA40 intervals contain0; fixed400 positive printed AND unrounded lower95, clean operations/pins; no pooling/rescue/automatic promotion'))
    hashes['game-manifest.json'] = sha('game-manifest.json')
    (p / 'game-inputs.sha256').write_text(''.join(pin + '  ' + name + '\n' for name, pin in hashes.items()))
    spec = importlib.util.spec_from_file_location('paired_parser', p / 'paired-parser.py')
    parser = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(parser)
    processes = subprocess.check_output(['ps', '-eo', 'pid,ppid,psr,pcpu,rss,args'], text=True).splitlines()
    (p / 'background-before-aa.txt').write_text('\n'.join([processes[0], *[line for line in processes[1:] if '/work/lanes/ngn-' in line]]) + '\n')
    for mode, argv in [('aa', aa), ('gate', gate)]:
        seconds = int((deadline - now()).total_seconds())
        assert seconds > 0, 'window ended; incomplete sample unscored'
        if mode == 'aa':
            seconds = min(seconds, 1800)
        subprocess.run(['sha256sum','-c','game-inputs.sha256'], check=True,
                       stdout=(p / ('input-check-before-' + mode + '.txt')).open('w'))
        status(mode + '-start', 'current package only; forge4-11/c2/nice10/host pinningON; no competing NGN CPU work')
        with (p / (mode + '.log')).open('w') as log:
            subprocess.run(['timeout','--signal=TERM','--kill-after=15s',str(seconds),*argv],check=True,
                           stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GOMAXPROCS='1',GOAMD64='v3'))
        result = parser.gate((p / (mode + '.log')).read_text(),mode)
        crashes = p / 'ngn_crashes.log'
        if crashes.exists():
            assert all('Crash Handler Initialized' in line or 'Crash log:' in line for line in crashes.read_text().splitlines())
        write(mode + '-summary.json', result)
        subprocess.run(['sha256sum','-c','game-inputs.sha256'],check=True,
                       stdout=(p / ('input-check-after-' + mode + '.txt')).open('w'))
        status(mode + '-done',result)

if __name__ == '__main__':
    main()
