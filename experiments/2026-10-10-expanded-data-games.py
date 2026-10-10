"""Freeze and play one reviewed expanded-data gate on a nominated playing base."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess

root = Path('/work/lanes/ngn-s47')
p = Path.cwd()
t = root / 'expanded-data'
sha = lambda f: hashlib.sha256(f.read_bytes()).hexdigest()
nom = json.loads((p / 'nomination.json').read_text())
training = json.loads((t / 'training-receipt.json').read_text())
assert (t / 'postprocess.done').exists() and int((t / 'postprocess.exit').read_text()) == 0
assert training['postprocessing_machine'] == 'forge'
assert training['epochs'] == 58 and training['total_positions'] == 14848000000
assert training['reexport_identical'] and training['parity_cases'] == 20
assert nom['reviewed_training_receipt_sha256'] == sha(t / 'training-receipt.json')
assert nom['base_net_sha256'] == 'be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
assert nom['base_engine_sha256'] == sha(p / 'ngn')
assert sha(p / 'base.nnue') == nom['base_net_sha256']
assert sha(t / 'trained/best.nnue') == training['net_sha256']
(p / 'candidate.nnue').write_bytes((t / 'trained/best.nnue').read_bytes())
(p / 'training-receipt.json').write_bytes((t / 'training-receipt.json').read_bytes())
assert sha(p / 'sprt') == '9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a'
assert sha(p / 'openings.txt') == '974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'
cpus, concurrency = nom['cpu_mask'], nom['concurrency']
common = ['-tc', '10+0.1', '-concurrency', str(concurrency), '-mingames', '1000000',
          '-maxmoves', '400', '-gametimeout', '180', '-openings', str(p / 'openings.txt'), '-lowpower=false']
def argv(net, count):
    a = ['taskset', '-c', cpus, 'nice', '-n', '10', str(p / 'sprt'),
         '-new', str(p / 'ngn'), '-base', str(p / 'ngn'), *common, '-maxgames', str(count)]
    for side, file in [('new', net), ('base', 'base.nnue')]:
        for name, value in [('EvalFile', str(p / file)), ('UseNNUE', 'true'),
                            ('Hash', '64'), ('Threads', '1'), ('OwnBook', 'false')]:
            a += ['-' + side + 'option', name + '=' + value]
    return a
files = ['ngn', 'base.nnue', 'candidate.nnue', 'sprt', 'openings.txt',
         'nomination.json', 'training-receipt.json', 'controller.sh', 'games.py',
         'gate-parser.py', 'manifest.md']
hashes = {f: sha(p / f) for f in files}
j = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
         base_source=nom['base_source'], change='expanded data only; identical accepted engine on both sides',
         training_source='56b35ef plus reviewed exact archive IO09d53c7',
         training_recipe_record='experiments/2026-10-10-expanded-data.md',
         training_review=nom['training_review'], candidate_net_sha256=training['net_sha256'],
         base_net_sha256=nom['base_net_sha256'], tc='10+0.1', cpu_mask=cpus,
         concurrency=concurrency, nice=10, host_pinning=nom['host_pinning'], memory_max_bytes=nom['memory_max_bytes'],
         environment={'GOMAXPROCS': '1', 'GOAMD64': 'v3'}, machine='forge Debian13 / Ryzen6900HX VM',
         os=platform.platform(), profile_sha256=nom['profile_sha256'],
         go_version=nom['go_version'], concurrent_workloads=nom['concurrent_workloads'],
         aa_argv=argv('base.nnue', 40), gate_argv=argv('candidate.nnue', 400), files_sha256=hashes,
         decision='freshA/A40 contains0/zero faults; exactly400/200 pairs, printedANDunrounded lower95>0/zero faults/intact inputs; otherwise shelve/no extension',
         automatic_promotion=False)
(p / 'game-manifest.json').write_text(json.dumps(j, indent=2) + '\n')
hashes['game-manifest.json'] = sha(p / 'game-manifest.json')
(p / 'game-inputs.sha256').write_text(''.join(h + '  ' + f + '\n' for f, h in hashes.items()))
env = dict(os.environ, GOMAXPROCS='1')
for mode, net, count, limit in [('aa', 'base.nnue', 40, 3600), ('gate', 'candidate.nnue', 400, 14400)]:
    if mode == 'gate':
        subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], check=True, stdout=(p/'input-check-after-aa.txt').open('w'))
    with (root / 'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                stage='session47-expanded-data-games-forge', event=mode+'-start',
                                detail='reviewed data-only gate; actual accepted source frozen'))+'\n')
    with (p / (mode + '.log')).open('w') as log:
        subprocess.run(['timeout', '--signal=TERM', '--kill-after=15s', str(limit), *argv(net, count)],
                       stdout=log, stderr=subprocess.STDOUT, env=env, check=True)
    with (p / (mode + '-summary.json')).open('w') as log:
        subprocess.run(['python3', 'gate-parser.py', 'gate', mode+'.log', '--gate-mode', mode],
                       stdout=log, check=True)
subprocess.run(['sha256sum', '-c', 'game-inputs.sha256'], check=True, stdout=(p/'input-check-after.txt').open('w'))
