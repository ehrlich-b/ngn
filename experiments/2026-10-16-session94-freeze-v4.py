"""Freeze the admitted v4 pool; WSL CPUs13,15 only, no engine launch."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import subprocess

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session94'
V = Path('/home/ehrli/ngn-data/claim-v4-20261016')
S = D / 'source'
T = V / 'tooling-s94'
F = T / 'frozen'
F.mkdir(parents=True)
argv = ['/usr/local/go/bin/go', 'build', '-buildvcs=true', '-p', '2', '-o', str(F / 'claimtool'), './cmd/claimtool']
(F / 'claimtool-build.json').write_text(json.dumps(argv) + '\n')
env = dict(os.environ, GOTOOLCHAIN='local', GOPROXY='off', GOAMD64='v3', GOMAXPROCS='2', GOCACHE=str(R / 'session17f/go-cache'))
assert not subprocess.check_output(['git', 'status', '--porcelain'], cwd=S)
with (T / 'build.log').open('w') as log:
    subprocess.run(argv, cwd=S, env=env, stdout=log, stderr=subprocess.STDOUT, check=True)
info = subprocess.check_output(['/usr/local/go/bin/go', 'version', '-m', str(F / 'claimtool')], text=True)
(F / 'buildinfo.txt').write_text(info)
assert 'vcs.modified=false' in info
modes = {}
for mode in ['screen', 'confirmation', 'tooling-check', 'stability-screen', 'stability-confirmation']:
    command = [str(F / 'claimtool'), 'prepare', '--mode', mode,
               '--candidate', str(D / 'dry-inputs/candidate-identity.json'),
               '--source', str(S), '--tool-build-argv', str(F / 'claimtool-build.json'),
               '--out', str(T / ('dry-' + mode)), '--dry-run']
    result = subprocess.run(command, env=env, capture_output=True, text=True)
    (T / ('dry-' + mode + '.log')).write_text(result.stdout + result.stderr)
    assert result.returncode == 0, (mode, result.stdout, result.stderr)
    modes[mode] = dict(status='PASS', argv=command)
receipt = dict(state='READY_TOOLING', source_commit=subprocess.check_output(['git','rev-parse','HEAD'],cwd=S,text=True).strip(),
               source_tree=subprocess.check_output(['git','rev-parse','HEAD^{tree}'],cwd=S,text=True).strip(),source_clean=True,
               runner=dict(path=str(F/'claimtool'),sha256=hashlib.sha256((F/'claimtool').read_bytes()).hexdigest()),
               go_version='go1.25.5',GOAMD64='v3',GOMAXPROCS=2,cpus='13,15',nice=19,
               dry_identity='Current accepted9ec5d31/81215229 metadata rebuild; no engine launch or scored games',
               dry_modes=modes, ready_for_games=True, games=0,
               caveat='Tool and anchor preparation ready; candidate identity and live exclusive-rig receipt must follow net gates and transfer.',
               focused_tests=str(D/'focused.log'),
               reused_full_checks=str(R/'session92/checks-result.json'),
               finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
(T/'build-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
with (R/'status.jsonl').open('a') as stream:
    stream.write(json.dumps(dict(utc=receipt['finished_utc'],stage='session94-v4',event='ready',detail=receipt))+'\n')
print(json.dumps(dict(state=receipt['state'],runner=receipt['runner'],source_commit=receipt['source_commit'],dry_modes=list(modes))))
