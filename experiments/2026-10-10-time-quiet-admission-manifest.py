"""Freeze checked actual-base time-admission artifacts before any games."""
import datetime
import hashlib
import json
import platform
from pathlib import Path
import subprocess

p = Path.cwd()
sha = lambda f: hashlib.sha256(f.read_bytes()).hexdigest()
names = ['base.tar', 'candidate.patch', 'baseline-ngn', 'candidate-ngn',
         'base.nnue', 'sprt', 'openings.txt', 'controller.sh', 'manifest.py',
         'aa-parser.py', 'paired-parser.py', 'gate-review.py', 'manifest.md']
hashes = {n: sha(p / n) for n in names}
assert hashes['base.nnue'] == 'be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
assert hashes['sprt'] == '9b0002f856849ebaf5e1d39487303d8e8f5406a8cc26d22fc2b409f9761aa83a'
assert hashes['openings.txt'] == '974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'
common = ['-tc', '60+0.6', '-concurrency', '2', '-maxmoves', '400',
          '-gametimeout', '900', '-openings', str(p / 'openings.txt'), '-lowpower=false']
def argv(kind, gate):
    a = ['taskset', '-c', '13,15', 'nice', '-n', '19', str(p / 'sprt'),
         '-new', str(p / (kind + '-ngn')), '-base', str(p / 'baseline-ngn'), *common]
    a += (['-elo0', '0', '-elo1', '6', '-alpha', '.05', '-beta', '.05',
           '-mingames', '400', '-maxgames', '4000'] if gate
          else ['-mingames', '1000000', '-maxgames', '40'])
    for side in ['new', 'base']:
        for n, v in [('EvalFile', str(p / 'base.nnue')), ('UseNNUE', 'true'),
                     ('Hash', '64'), ('Threads', '1'), ('OwnBook', 'false')]:
            a += ['-' + side + 'option', n + '=' + v]
    return a
j = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
         base_source='56b35efaaa147d594add941d82448779c27d4a72',
         candidate='base plus candidate.patch only; clock at new-depth admission',
         base_net_sha256=hashes['base.nnue'], files_sha256=hashes,
         machine='Ryzen9800X3D/WSL2', os=platform.platform(),
         go_version=subprocess.check_output(['go', 'version'], text=True).strip(),
         cpus=[13, 15], nice=19, concurrency=2, memory_max_bytes=2621440000,
         environment={'GOMAXPROCS': '1', 'GOAMD64': 'v3'},
         profile_sha256=sha(p / 'baseline/default.pgo'),
         concurrent_workloads=['provenance0-11/3GiB', 'data trainer12,14/4GiB', 'hopper untouched'],
         checks=['four existing controls', 'full-short', 'full-race', 'vet', 'explicit-PGO builds'],
         source_sha256={f: sha(p / 'candidate' / f) for f in ['engine/time.go', 'engine/time_admission_test.go']},
         aa_argv=argv('baseline', False), gate_argv=argv('candidate', True),
         decision='A/A40 contains0; completedH1 AND printed/unrounded lower95>0 AND zero faults/intact inputs; otherwise shelve/no extension',
         automatic_promotion=False)
(p / 'game-manifest.json').write_text(json.dumps(j, indent=2) + '\n')
hashes['game-manifest.json'] = sha(p / 'game-manifest.json')
(p / 'game-inputs.sha256').write_text(''.join(h + '  ' + n + '\n' for n, h in hashes.items()))
