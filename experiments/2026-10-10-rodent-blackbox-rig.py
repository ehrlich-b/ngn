"""Run the frozen black-box diagnostic after manual settled-package nomination."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess

root = Path('/home/ehrli/ngn-data/night-20261007')
p = root / 'session50-rodent-blackbox'
opponent = root / 'session19-rodent3/opponent-only/rodent-v12'
opponent_cwd = opponent.parent / 'rodent'
sha = lambda f: hashlib.sha256(f.read_bytes()).hexdigest()
nom = json.loads((p / 'nomination.json').read_text())
assert nom['playing_source'] == '9bfbae4'
assert sha(p / 'ngn') == '84431b35d3ae8ef92c6e9f1b967cc01ad35b400a1f812c347fe6f0dcbe55c2e1'
assert sha(p / 'net.nnue') == nom['playing_net_sha256']
assert sha(opponent) == '10dd722e730a70887fcaa07157b86c5ba3579618f858f7519d44f55036e00949'
assert sha(p / 'blackboxmatch') == 'c76498ade22ff41209cce86a37c6314f4cea82b355615a74b2594164cfc82fcc'
env = dict(os.environ, GOMAXPROCS='1', GOAMD64='v3')
ngn_options = [('EvalFile', str(p / 'net.nnue')), ('UseNNUE', 'true'),
               ('Hash', '64'), ('Threads', '1'), ('OwnBook', 'false')]
rodent_options = [('Hash', '64'), ('Threads', '1'), ('OwnBook', 'false'),
                  ('UCI_LimitStrength', 'false')]


def argv(stage, mode='match', aa=False, clock=False, **budget):
    engine = p / ('clock-ngn' if clock else 'ngn')
    command = ['taskset', '-c', '0-11', 'nice', '-n', '19', str(p / 'blackboxmatch'),
               '-ngn', str(engine), '-opponent', str(engine if aa else opponent),
               '-mode', mode, '-concurrency', '6', '-out', str(p / stage),
               '-openings', str(p / 'openings100.txt'),
               '-positions', str(p / 'cmd/blackboxmatch/positions.txt')]
    for side, options in [('ngn', ngn_options),
                           ('opponent', ngn_options if aa else rodent_options)]:
        for name, value in options:
            command += ['-' + side + 'option', name + '=' + value]
    for name, value in budget.items():
        command += ['-' + name, str(value)]
    return command


def status(event, detail):
    with (root / 'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                stage='session50-rodent-blackbox', event=event,
                                detail=detail)) + '\n')


def run(stage, command, timeout):
    (p / (stage + '-command.json')).write_text(json.dumps(
        dict(argv=command, cwd=str(opponent_cwd), environment=env_subset,
             timeout_seconds=timeout), indent=2) + '\n')
    status(stage + '-start', 'frozen diagnostic; no playing adoption')
    with (p / (stage + '.log')).open('w') as f:
        subprocess.run(['timeout', '--signal=TERM', '--kill-after=15s', str(timeout),
                        *command], cwd=opponent_cwd, env=env, stdout=f,
                       stderr=subprocess.STDOUT, check=True)
    status(stage + '-done', 'complete; preserve failures, no retries or extension')


def review(name, *arguments):
    with (p / name).open('w') as f:
        subprocess.run(['python3', str(p / 'cmd/blackboxmatch/review.py'), *arguments],
                       cwd=p, env=env, stdout=f, check=True)


env_subset = {k: env[k] for k in ('GOMAXPROCS', 'GOAMD64')}
files = ['ngn', 'clock-ngn', 'blackboxmatch', 'net.nnue', 'openings100.txt',
         'cmd/blackboxmatch/positions.txt', 'cmd/blackboxmatch/review.py',
         'nomination.json', 'controller.sh', 'controller.py', 'manifest.md']
hashes = {f: sha(p / f) for f in files}
manifest = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                nomination=nom, machine='rig WSL2 / Ryzen7 9800X3D', os=platform.platform(),
                environment=env_subset, cpu_mask='0-11', concurrency=6, nice=19,
                memory_max_bytes=4294967296, swap_max=0, host_pinning='not-applicable',
                files_sha256=hashes, opponent_binary_sha256=sha(opponent),
                opponent_cwd=str(opponent_cwd), opponent='existing Rodent V1.2 default net; black-box only',
                probe_argv=argv('probe', mode='probe', repetitions=3),
                node_budget_rule='integer median of96 NGN1s counts; frozen before scored games',
                sample='384 probes;64 node-compliance probes;fresh AA40 each cell;200 nodes;200 time;AA40/24 profiles120+1',
                decision='diagnostic only; complete legal pairs/normal exits/zero faults/intact inputs; no extension or adoption',
                throughput_rule='equal-node deficit improves >=10 Elo AND per-opening score-difference lower95>0',
                automatic_promotion=False)
(p / 'game-manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
hashes['game-manifest.json'] = sha(p / 'game-manifest.json')
(p / 'inputs.sha256').write_text(''.join(h + '  ' + f + '\n' for f, h in hashes.items()))


def check_inputs(name):
    with (p / name).open('w') as f:
        subprocess.run(['sha256sum', '-c', 'inputs.sha256'], cwd=p, stdout=f, check=True)


check_inputs('input-check-before.txt')
run('probe', argv('probe', mode='probe', repetitions=3), 3600)
review('node-budget.json', 'budget', str(p / 'probe'))
nodes = json.loads((p / 'node-budget.json').read_text())['common_node_budget']
run('compliance', argv('compliance', mode='compliance', repetitions=1, nodes=nodes), 1800)

# Concrete node/time/clock commands are frozen together before the first game.
commands = [
    ('aa-nodes', argv('aa-nodes', aa=True, pairs=20, nodes=nodes), 10800),
    ('nodes', argv('nodes', pairs=100, nodes=nodes), 21600),
    ('aa-time', argv('aa-time', aa=True, pairs=20, movetime=1000), 10800),
    ('time', argv('time', pairs=100, movetime=1000), 21600),
    ('aa-clock', argv('aa-clock', aa=True, clock=True, pairs=20, bank=120000, increment=1000), 14400),
    ('clock', argv('clock', clock=True, pairs=12, bank=120000, increment=1000), 10800),
]
(p / 'cell-manifest.json').write_text(json.dumps(dict(common_node_budget=nodes,
    commands=[dict(stage=s, argv=a, timeout_seconds=t) for s, a, t in commands]), indent=2) + '\n')
for stage, command, timeout in commands:
    check_inputs('input-check-' + stage + '.txt')
    run(stage, command, timeout)
    if stage.startswith('aa-'):
        mode, budget = {'aa-nodes': ('nodes', nodes), 'aa-time': ('movetime', 1000),
                        'aa-clock': ('clock', 120000)}[stage]
        review(stage + '-review.json', 'aa', str(p / stage), mode, str(budget))
    elif stage == 'time':
        review('comparison.json', 'compare', str(p / 'nodes'), str(p / 'time'), str(nodes))
    elif stage == 'clock':
        review('clock-review.json', 'clock', str(p / 'clock'))
check_inputs('input-check-after.txt')
status('complete', 'review comparison/profile records; no engine adoption or claim')
