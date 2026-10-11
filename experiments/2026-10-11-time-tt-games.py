"""Manually nominated immediate-admission SPRT on the accepted TT source."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess

p = Path.cwd()
root = p.parent
sha = lambda file: hashlib.sha256(file.read_bytes()).hexdigest()
nom = json.loads((p / 'nomination.json').read_text())
checks = json.loads((p / 'checks-receipt.json').read_text())
assert checks['source'] == nom['base_source'] == 'ed34480'
assert nom['base_engine_sha256'] == checks['files_sha256']['baseline-ngn'] == sha(p / 'baseline-ngn')
assert nom['candidate_engine_sha256'] == checks['files_sha256']['candidate-ngn'] == sha(p / 'candidate-ngn')
assert nom['base_net_sha256'] == sha(p / 'base.nnue')
assert sha(p / 'candidate.patch') == '76efa5266d49c5bf3f8106d2f9907a1e55066efa0637ac8c84b1cb0bcdc01dd7'
assert sha(p / 'sprt') == '9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a'
assert sha(p / 'openings.txt') == '974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'
cpus, concurrency = '0-11', 6
common = ['-tc', '60+0.6', '-concurrency', str(concurrency), '-maxmoves', '400',
          '-gametimeout', '900', '-openings', str(p / 'openings.txt'), '-lowpower=false']
options = []
for side in ['new', 'base']:
    for name, value in [('EvalFile', str(p / 'base.nnue')), ('UseNNUE', 'true'),
                        ('Hash', '64'), ('Threads', '1'), ('OwnBook', 'false')]:
        options += ['-' + side + 'option', name + '=' + value]

def argv(mode):
    candidate = 'baseline-ngn' if mode == 'aa' else 'candidate-ngn'
    a = ['taskset', '-c', cpus, 'nice', '-n', '19', str(p / 'sprt'),
         '-new', str(p / candidate), '-base', str(p / 'baseline-ngn'), *common, *options]
    if mode == 'aa':
        return a + ['-mingames', '1000000', '-maxgames', '40']
    return a + ['-elo0', '0', '-elo1', '6', '-alpha', '.05', '-beta', '.05',
                '-mingames', '400', '-maxgames', '4000']

files = ['baseline-ngn', 'candidate-ngn', 'base.nnue', 'candidate.patch', 'checks-receipt.json',
         'sprt', 'openings.txt', 'nomination.json', 'controller.sh', 'games.py',
         'aa-parser.py', 'paired-parser.py', 'gate-review.py', 'manifest.md']
hashes = {f: sha(p / f) for f in files}
j = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
         base_source=nom['base_source'], base_net_sha256=nom['base_net_sha256'],
         candidate='unchanged immediate-admission patch only on accepted TT baseline',
         machine='rig WSL2 / Ryzen7 9800X3D', os=platform.platform(), tc='60+0.6',
         cpu_mask=cpus, concurrency=concurrency, nice=19, memory_max_bytes=3*1024**3,
         environment={'GOMAXPROCS': '1', 'GOAMD64': 'v3'},
         profile_sha256=nom['profile_sha256'], go_version=checks['go_version'],
         concurrent_workloads=nom['concurrent_workloads'], checks=checks['checks'],
         aa_argv=argv('aa'), gate_argv=argv('gate'), files_sha256=hashes,
         decision='ownAA40 then SPRT0,+6/min400/cap4000; H1ANDprintedANDexactlower95>0/zero faults/intact pins only',
         automatic_promotion=False)
(p / 'game-manifest.json').write_text(json.dumps(j, indent=2) + '\n')
hashes['game-manifest.json'] = sha(p / 'game-manifest.json')
(p / 'game-inputs.sha256').write_text(''.join(h + '  ' + f + '\n' for f, h in hashes.items()))
for mode, limit in [('aa', 14400), ('gate', 259200)]:
    if mode == 'gate':
        subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], check=True,
                       stdout=(p / 'input-check-after-aa.txt').open('w'))
    with (root / 'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                               stage='session56-time-tt-games', event=mode+'-start',
                               detail='manual settled-package admission only; rig0-11/c6/3GiB'))+'\n')
    with (p / (mode + '.log')).open('w') as log:
        subprocess.run(['timeout', '--signal=TERM', '--kill-after=15s', str(limit), *argv(mode)],
                       stdout=log, stderr=subprocess.STDOUT, check=True,
                       env=dict(os.environ, GOMAXPROCS='1'))
    if mode == 'aa':
        with (p / 'aa-summary.json').open('w') as log:
            subprocess.run(['python3', 'aa-parser.py', 'gate', 'aa.log', '--gate-mode', 'aa'],
                           stdout=log, check=True)
    else:
        subprocess.run(['python3', 'gate-review.py'], check=True)
subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], check=True,
               stdout=(p / 'input-check-after.txt').open('w'))
