"""WSL: natural net terminals -> transfer -> prospective v4 screen -> confirmation.

Runs in a separate unlimited-CPU service, never in the 600% net-gate scope.
No existing gate is interrupted. No remote publication or source adoption.
"""
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import select
import subprocess

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session94'
V = Path('/home/ehrli/ngn-data/claim-v4-20261016/tooling-s94')
PHYSICAL = '0,2,4,6,8,10,12,14'


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('x') as stream:
        stream.write(json.dumps(value, indent=2, allow_nan=False) + '\n')


def read(path):
    return json.loads(path.read_text())


def bind(path):
    return dict(Path=str(path), SHA256=hashlib.sha256(path.read_bytes()).hexdigest())


def status(event, detail):
    with (R / 'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=now(), stage='session94-claim-chain', event=event, detail=detail)) + '\n')


def wait_pid(pid):
    try:
        fd = os.pidfd_open(pid)
    except ProcessLookupError:
        return
    try:
        select.select([fd], [], [])
    finally:
        os.close(fd)


def load_module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def choose_net(results):
    # Prospective priority, independent of observed score ranking. Both nets
    # were tested against 81215229; no between-net gain is claimed or stacked.
    for name in ['moderated-t74', 'volume-control']:
        if results[name]['verdict'] == 'KEEP':
            return name
    return 'accepted-final115'


def transfer_job(name):
    template = read(R / 'session83/net-transfer/manifest.json')
    net_gate = read(R / ('session93/net-' + name) / 'manifest.json')
    for key in ['argv', 'environment', 'pid', 'started_utc', 'reused_control']:
        template.pop(key, None)
    template.update(name='s94-transfer-' + name, output=str(D / ('transfer-' + name)),
                    net_sha256=net_gate['net_sha256'],
                    candidate_options=net_gate['candidate_options'], base_options=net_gate['base_options'],
                    artifacts=net_gate['artifacts'], review_after=False,
                    selection='Predeclared moderated priority if both fixed1000 nets KEEP; no score ranking',
                    decision='fixed400/200 at60+0.6; exact point>0 and zero faults before v4 foreign games')
    return template


def identity(name, attempt):
    package = read(D / 'playing-build.json')
    if name == 'accepted-final115':
        net = R / 'session82/net-final115/candidate.nnue'
        cell = R / 'session80-training'
    else:
        cell = R / 'session87/mixture' / name
        net = cell / 'trained/latest.nnue'
    ready = read(cell / 'result.json')
    assert ready['epochs'] == 115 and ready['reexport_identical'] and ready['parity_cases'] == 20
    net_artifact = bind(net)
    assert net_artifact['SHA256'] == ready['net_sha256']
    provenance = {key: bind(path) for key, path in dict(
        manifest=cell / 'manifest.json', data=cell / 'manifest.json',
        training=cell / 'trained/history.json', export=cell / 'result.json',
        parity=cell / 'trained/parity.json').items()}
    path = attempt / 'candidate-identity.json'
    write(path, dict(Engine=package['engine'], Net=net_artifact,
                     SourceRoot=package['source_root'], SourceCommit=package['source_commit'],
                     GoVersion='go1.25.5', BuildArgv=package['argv'], Provenance=provenance))
    return path


def command(argv, log):
    with log.open('x') as stream:
        subprocess.run(argv, stdout=stream, stderr=subprocess.STDOUT, check=True,
                       env=dict(os.environ, GOMAXPROCS='8'))


def claim_stage(mode, candidate, attempt):
    stage = attempt / mode
    tool = str(V / 'frozen/claimtool')
    prefix = ['taskset', '-c', PHYSICAL, tool]
    argv = prefix + ['prepare', '--mode', mode, '--candidate', str(candidate),
                     '--source', str(D / 'source'), '--tool-build-argv', str(V / 'frozen/claimtool-build.json'),
                     '--out', str(stage)]
    if mode == 'confirmation':
        argv += ['--screen-receipt', str(attempt / 'screen/receipt.json'),
                 '--screen-analysis', str(attempt / 'screen/analysis.json')]
    status('starting-' + mode, dict(attempt=str(attempt), argv=argv))
    command(argv, attempt / (mode + '-prepare.log'))
    receipt, games = str(stage / 'receipt.json'), str(stage / 'games')
    command(prefix + ['run', '--receipt', receipt, '--games', games, '--out', str(stage / 'run-status.json')],
            attempt / (mode + '-run.log'))
    command(prefix + ['audit', '--receipt', receipt, '--games', games, '--out', str(stage / 'audit.json')],
            attempt / (mode + '-audit.log'))
    command(prefix + ['analyze', '--receipt', receipt, '--audit', str(stage / 'audit.json'),
                      '--games', games, '--out', str(stage / 'analysis.json')], attempt / (mode + '-analysis.log'))
    # Run/audit/analyze already bind and recheck artifacts. No extra custody pass.
    result = read(stage / 'analysis.json')
    status(mode + '-complete', dict(primary=result['Primary'], bracketed=result['Bracketed'],
                                    screen_eligible=result['ScreenEligible'], claim_eligible=result['ClaimEligible'],
                                    analysis=str(stage / 'analysis.json')))
    return result


def resume_work():
    pause = read(R / 'session93/spsa-planned-pause.json')
    argv = pause['resume_argv']
    assert argv[-1] == '-resume'
    command(['systemd-run', '--user', '--unit=ngn-s94-spsa-resume',
             '-p', 'AllowedCPUs=0-11', '-p', 'CPUQuota=600%', '-p', 'MemoryMax=3G', '-p', 'MemorySwapMax=0',
             '-p', 'Environment=GOMAXPROCS=1',
             '-p', 'StandardOutput=append:' + str(R / 'session88/spsa-warm/tune.log'), '-p', 'StandardError=inherit',
             'nice', '-n', '19', 'taskset', '-c', '0-11'] + argv, D / 'spsa-resume-launch.log')
    command(['systemd-run', '--user', '--unit=ngn-s94-gpu-next',
             '-p', 'AllowedCPUs=12-15', '-p', 'CPUQuota=200%', '-p', 'MemoryMax=4G', '-p', 'MemorySwapMax=0',
             '-p', 'Environment=PYTHONPATH=' + str(R / 'session87/training-source'),
             '-p', 'Environment=OMP_NUM_THREADS=2', '-p', 'Environment=MKL_NUM_THREADS=2',
             '-p', 'Environment=OPENBLAS_NUM_THREADS=2',
             '-p', 'StandardOutput=append:' + str(D / 'gpu-next/controller.log'), '-p', 'StandardError=inherit',
             'nice', '-n', '19', '/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python',
             str(D / 'gpu-next.py')], D / 'gpu-next-launch.log')
    status('resumed', dict(spsa='ngn-s94-spsa-resume.service', gpu='ngn-s94-gpu-next.service'))


def width_check(candidate, attempt, runner):
    """Only after a failed screen with D>40; no training/tune until complete."""
    directory = attempt / 'width8-vs4'
    directory.mkdir()
    ident = read(candidate)
    net = ident['Net']['Path']
    engine = ident['Engine']['Path']
    wrappers = {}
    for width in [4, 8]:
        mask = PHYSICAL if width == 8 else '0,2,4,6'
        path = directory / ('ngn' + str(width))
        # These campaign-owned paths contain no shell metacharacters.
        import shlex
        path.write_text('#!/bin/sh\nexport GOMAXPROCS=' + str(width) + '\nexec taskset -c ' + mask + ' ' + shlex.quote(engine) + ' "$@"\n')
        path.chmod(0o755)
        wrappers[width] = path
    template = read(R / 'session87/net-final174/manifest.json')
    for key in ['argv', 'environment', 'pid', 'started_utc', 'reused_control', 'source_checks', 'finalization']:
        template.pop(key, None)
    options = ['EvalFile=' + net, 'UseNNUE=true', 'Threads=8', 'Hash=256', 'OwnBook=false']
    template.update(engine=engine, candidate=str(wrappers[8]), base=str(wrappers[8]), cpus='0-15',
                    concurrency=1, candidate_options=options, base_options=options,
                    artifacts={engine: ident['Engine']['SHA256'], net: ident['Net']['SHA256'],
                               template['harness']: template['artifacts'][template['harness']],
                               template['book']: template['artifacts'][template['book']]},
                    control_key='rig/physical8/c1/10+0.1/Threads8/Hash256/s94-width', review_after=False)
    for path in wrappers.values():
        template['artifacts'][str(path)] = bind(path)['SHA256']
    aa = dict(template, name='s94-width8-aa', output=str(directory / 'aa'), max_games=40, **{'class': 'aa'})
    measurement = dict(template, name='s94-width8-vs4', output=str(directory / 'measurement'),
                       base=str(wrappers[4]), base_options=[x if x != 'Threads=8' else 'Threads=4' for x in options],
                       max_games=400, **{'class': 'measurement'})
    write(directory / 'nomination.json', dict(aa=aa, measurement=measurement,
          decision='fixed400,zero faults;point<25 opens an SMP profile;otherwise resume single-thread queue;no absolute claim'))
    probe = load_module('width_probe', R / 'session87/width-probe.py')
    probe.D = directory
    for job, name, widths, masks in [(aa, 'probe-aa', [8, 8], [list(range(0, 16, 2))] * 2),
                                   (measurement, 'probe-width', [8, 4], [list(range(0, 16, 2)), [0, 2, 4, 6]])]:
        observed = probe.probe(job, name, widths, masks)
        assert all(row['peak_5s_cpu_wall_ratio'] >= .875 * int(row['gomaxprocs']) for row in observed['children'])
    result = runner.run(aa, R / 'session80/controls.json', R / 'status.jsonl')
    assert result['verdict'] == 'AA_PASS' and not result['faults']
    result = runner.run(measurement, R / 'session80/controls.json', R / 'status.jsonl')
    assert result['verdict'] == 'MEASURED' and not result['faults']
    status('width-complete', result)
    return result['exact_elo_ci95'][0] >= 25


def main():
    config = read(D / 'chain-nomination.json')
    status('waiting-net-chain', dict(pid=config['net_chain_pid'], terminal=str(R / 'session93/net-chain-result.json')))
    wait_pid(config['net_chain_pid'])
    predecessor = read(R / 'session93/net-chain-result.json')
    assert predecessor['state'] == 'REVIEW_NET_GATES', predecessor
    results = {name: read(R / ('session93/net-' + name) / 'result.json') for name in ['volume-control', 'moderated-t74']}
    for result in results.values():
        assert result['games'] == 1000 and result['pairs'] == 500 and not result['faults']
        assert result['verdict'] in ['KEEP', 'SHELVED']
    assert read(D / 'playing-build.json')['identity']['verdict'] == 'PASS'
    assert read(V / 'build-receipt.json')['state'] == 'READY_TOOLING'
    runner = load_module('queue', R / 'session93/campaign_queue.py')
    name = choose_net(results)
    if name != 'accepted-final115':
        job = transfer_job(name)
        write(D / 'transfer-nomination.json', job)
        transfer = runner.run(job, R / 'session80/controls.json', R / 'status.jsonl')
        assert transfer['verdict'] == 'TRANSFER_PASS' and not transfer['faults'], transfer
    attempt = Path('/home/ehrli/ngn-data/claim-v4-20261016/attempt-001')
    attempt.mkdir()
    candidate = identity(name, attempt)
    write(D / 'selected-package.json', dict(net=name, identity=str(candidate),
          base='9ec5d31/81215229/compiled final14', source_changed=False,
          net_gates={key: str(R / ('session93/net-' + key) / 'result.json') for key in results},
          transfer='accepted-final115 prior transfer reused' if name == 'accepted-final115' else str(D / ('transfer-' + name) / 'result.json')))
    write(attempt / 'nomination.json', dict(authority=config['authority'], candidate=str(candidate), selected_net=name,
          selection=config['net_selection'], labels=str(Path('/home/ehrli/ngn-data/claim-v4-20261016/labels-koivisto.json')),
          occupancy='screen/occupancy.json; real game launcher, CPU/wall>=7 per engine, exact mask; inherited cpu.max recorded',
          monitor='resources.jsonl every600seconds,descriptive only;no stop/verdict effect',
          resources='unlimited CPUQuota in ngn-s94-claim-chain;GOMAX8;tasksetphysical8;nice19;exclusive',
          screen='fixed320/10+0.1;lower95>3600 andrawscores bracket50%',
          confirmation='automatic only on screen PASS;same identity;fixed640/120+1;strictlower95>3600 andbracketing',
          created_utc=now()))
    sidecar = subprocess.Popen(['python3', str(D / 'monitor.py'), str(os.getpid()), str(attempt / 'resources.jsonl')])
    try:
        screen = claim_stage('screen', candidate, attempt)
        if screen['ScreenEligible']:
            confirmation = claim_stage('confirmation', candidate, attempt)
            if confirmation['ClaimEligible']:
                write(D / 'claim-chain-result.json', dict(state='GOAL_MET', analysis=str(attempt / 'confirmation/analysis.json'), primary=confirmation['Primary']))
                status('GOAL_MET', confirmation['Primary'])
                return
        else:
            deficit = 3640 - screen['Primary']['Estimate']
            status('screen-failed', dict(primary=screen['Primary'], anchors=screen['Anchors'], shortfall=deficit))
            if deficit > 40 and not width_check(candidate, attempt, runner):
                write(D / 'claim-chain-result.json', dict(state='NEEDS_SMP_PROFILE', shortfall=deficit,
                      next='Profile Threads8 at this natural boundary before code;no tune/trainer launched'))
                return
    finally:
        sidecar.terminate()
        sidecar.wait()
    resume_work()
    write(D / 'claim-chain-result.json', dict(state='FAILED_CLAIM_WORK_RESUMED', attempt=str(attempt), finished_utc=now()))


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        if not (D / 'claim-chain-result.json').exists():
            write(D / 'claim-chain-result.json', dict(state='HELD_REVIEW', error=str(error), at_utc=now()))
        status('held-review', str(error))
        raise
