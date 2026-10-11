"""Prepare and audit one factorizer trial; a reviewed base nomination is mandatory."""
import argparse
import hashlib
import json
import math
from pathlib import Path
import re
import subprocess
import sys
import tempfile

CAMPAIGN = Path('/home/ehrli/ngn-data/night-20261007')
ADMISSION = Path('/home/ehrli/ngn-data/claim-v3-20261008/admission-session3')
PYTHON = '/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python'
DATA = '/home/ehrli/nnue-owned-k4-20260920/archive-a1/corpus-a1/train-archive.bf'
DATA_HASH = '270f65bdf9f5c948160cae5330030a60f99e94383de2a290b565ae7aad7cde26'
BASE_HASH = 'be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
RECIPE = dict(data=[DATA], data_format='archive', archive_holdout_modulus=1024,
              archive_split_seed=20261006, hidden=512, buckets=8, format=3,
              kb_map='default', batch=16384, lr=.001, positions_per_epoch=256000000,
              decay='cosine', decay_steps=1000, gamma=.5, min_lr_ratio=.1,
              wdl=.75, k=400., weight_decay=.01, drop_flags=15, min_ply=0,
              block_records=262144, seed=20261006, device='cuda', threads=2,
              precision='bf16', keep_best_only=True, resume=None,
              validation_data=None, record_counts=None)


def digest(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(8 * 1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def load(path):
    return json.loads(Path(path).read_text())


def write(path, value):
    Path(path).write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def score_to_elo(score):
    # Match the pinned instrument's endpoint guards, independently of its code.
    if score <= 1e-4:
        return -800.
    if score >= 1 - 1e-4:
        return 800.
    return 400 * math.log10(score / (1 - score))


def gate(text, mode):
    expected = {'aa': 200, 'gate': 400}[mode]
    assert not re.search(r'(?im)panic|fatal|illegal|no.move|watchdog|timeout|crash|forfeit|engine.error', text)
    summaries = re.findall(r'Games: (\d+)\s+W-D-L: (\d+)-(\d+)-(\d+)', text)
    assert len(summaries) == 1, 'missing or repeated final summary'
    games, w, d, l = map(int, summaries[0])
    assert games == w + d + l == expected
    progress = re.findall(r'(?m)^G(\d+)\s+(\d+)W\s+(\d+)D\s+(\d+)L.*?\s+(\S+)\s*$', text)
    assert len(progress) == expected, 'incomplete raw game progress'
    previous = (0, 0, 0)
    reasons = {}
    for i, (n, wi, di, li, reason) in enumerate(progress, 1):
        counts = tuple(map(int, (wi, di, li)))
        assert int(n) == i == sum(counts)
        assert tuple(c - p for c, p in zip(counts, previous)) in ((1, 0, 0), (0, 1, 0), (0, 0, 1))
        assert reason in ('checkmate', 'draw-rule', 'max-moves', 'stalemate'), reason
        reasons[reason] = reasons.get(reason, 0) + 1
        previous = counts
    assert previous == (w, d, l)
    pentas = re.findall(r'Pentanomial \[LL (\d+)\s+LD (\d+)\s+\{LW,DD\} (\d+)\s+WD (\d+)\s+WW (\d+)\] over (\d+) pairs', text)
    assert len(pentas) == 1
    *penta, pairs = map(int, pentas[0])
    assert sum(penta) == pairs == expected // 2
    score = sum(k * c / 4 for k, c in enumerate(penta)) / pairs
    assert abs(score - (w + .5 * d) / games) < 1e-12
    variance = max(sum((k / 4) ** 2 * c for k, c in enumerate(penta)) / pairs - score ** 2, .001)
    se = math.sqrt(variance / pairs)
    exact = [score_to_elo(score), score_to_elo(score - 1.96 * se), score_to_elo(score + 1.96 * se)]
    estimates = re.findall(r'Penta Elo: ([-+\d.]+)\s+95% CI \[([-+\d.]+),\s*([-+\d.]+)\]', text)
    assert len(estimates) == 1
    reported = list(map(float, estimates[0]))
    assert all(math.isfinite(x) for x in reported)
    assert abs(exact[0] - reported[0]) <= .051
    assert all(abs(a - b) <= .501 for a, b in zip(exact[1:], reported[1:]))
    flags = re.findall(r'Flag-outs \(lost on time\): new (\d+), base (\d+)\s+of (\d+) games', text)
    assert len(flags) == 1 and tuple(map(int, flags[0])) == (0, 0, expected)
    if mode == 'aa':
        assert reported[1] <= 0 <= reported[2] and exact[1] <= 0 <= exact[2]
    return dict(mode=mode, games=games, pairs=pairs, wins=w, draws=d, losses=l,
                penta=penta, elo=reported[0], ci95=reported[1:], exact_elo_ci95=exact,
                flags=[0, 0], reasons=reasons,
                verdict='aa-pass' if mode == 'aa' else ('eligible' if reported[1] > 0 and exact[1] > 0 else 'shelved'))


def recipe(config, epochs):
    assert epochs in (58, 116) and config['epochs'] == epochs
    for key, value in RECIPE.items():
        assert config[key] == value, f'base recipe changed: {key}'
    assert not config.get('king_factorizer', False)
    assert config.get('ngnp_holdout_modulus', 0) == 0
    assert config.get('ngnp_split_seed') is None
    split = config['archive_split']
    assert (split['hash'], split['seed'], split['modulus'], split['raw_records'],
            split['training_records'], split['holdout_records']) == (
        'ngn-board-mix64-v1', 20261006, 1024, 2374290033, 2371970389, 2319644)


def command(root, epochs):
    return [PYTHON, '-m', 'trainer.train', '--data', DATA, '--data-format', 'archive',
            '--archive-holdout-modulus', '1024', '--archive-split-seed', '20261006',
            '--out', str(root / 'candidate'), '--format', '3', '--kb-map', 'default',
            '--hidden', '512', '--buckets', '8', '--batch', '16384', '--lr', '.001',
            '--epochs', str(epochs), '--positions-per-epoch', '256000000', '--wdl', '.75',
            '--k', '400', '--decay', 'cosine', '--min-lr-ratio', '.1', '--threads', '2',
            '--precision', 'bf16', '--seed', '20261006', '--keep-best-only', '--king-factorizer']


def plan(root, campaign=CAMPAIGN, admission=ADMISSION):
    n = load(root / 'nomination.json')
    assert n['reviewed'] is True
    assert re.fullmatch(r'[0-9a-f]{40}', n['review_commit'])
    assert n['review_record'].startswith('experiments/') and n['review_record'].endswith('.md')
    events = [json.loads(line) for line in (campaign / 'status.jsonl').read_text().splitlines()]
    assert any(e['stage'] == 'session1' and e['event'] == 'complete' for e in events)
    checks = (campaign / 'session1/post-gate-input-check.log').read_text().splitlines()
    assert checks and all(line.endswith(': OK') for line in checks)
    aa = gate((campaign / 'session1/aa.log').read_text(), 'aa')
    result = gate((campaign / 'session1/gate.log').read_text(), 'gate')
    accepted = result['verdict'] == 'eligible'
    assert n['longer_verdict'] == ('accepted' if accepted else 'shelved')
    selected = 'kb8h512-long' if accepted else 'kb8h512'
    assert n['base'] == selected, 'nomination must select the immediate accepted base'
    epochs = 116 if accepted else 58
    base = campaign / selected
    recipe(load(base / 'config.json'), epochs)
    assert digest(base / 'config.json') == digest(root / f'{selected}.config.json')
    parity = load(campaign / 'session1/parity.json')
    assert parity['parity_cases'] == 20 and parity['reexport_identical'] is True
    assert parity['network_sha256'] == digest(campaign / 'kb8h512-long/best.nnue')
    assert parity['checkpoint_sha256'] == digest(campaign / 'kb8h512-long/best.pt')
    assert n['base_net_sha256'] == digest(base / 'best.nnue')
    if not accepted:
        assert n['base_net_sha256'] == BASE_HASH
    evidence = dict(aa=campaign / 'session1/aa.log', gate=campaign / 'session1/gate.log',
                    parity=campaign / 'session1/parity.json',
                    post_gate=campaign / 'session1/post-gate-input-check.log',
                    admission=admission / 'admission-receipt.json',
                    cuda=campaign / 'session8b/cuda-receipt.json')
    assert n['evidence_sha256'] == {k: digest(v) for k, v in evidence.items()}
    admitted, cuda = load(evidence['admission']), load(evidence['cuda'])
    assert (admission / 'controller.done').is_file()
    assert admitted['admitted'] is True and admitted['games'] == 12
    assert (campaign / 'session8b/cuda-controller.done').is_file()
    assert (campaign / 'session8b/cuda-controller.exit').read_text().strip() == '0'
    assert cuda['passed'] is True
    assert cuda['log_sha256'] == digest(campaign / 'session8b/cuda-tests.log')
    assert 'skipped' not in (campaign / 'session8b/cuda-tests.log').read_text()
    build = load(root / 'build-receipt.json')
    assert build['source_reconciled'] is True and build['engine_inputs'] == 230 and build['trainer_inputs'] == 25
    for name, sha in build['binary_sha256'].items():
        assert digest(root / 'frozen' / name) == sha
    return dict(base=selected, base_net=str(base / 'best.nnue'), base_net_sha256=n['base_net_sha256'],
                epochs=epochs, trainer_argv=command(root, epochs), aa=aa, longer_gate=result,
                nomination_sha256=digest(root / 'nomination.json'), evidence_sha256=n['evidence_sha256'],
                source_commit=build['source_commit'], binary_sha256=build['binary_sha256'],
                inference_change=False, behavior_change='king_factorizer false -> true')


def parity(root):
    import numpy as np
    import torch
    from trainer.export import export_state
    from trainer.ngnp import RECORD_DTYPE, decode_record, to_fen
    from trainer.refeval import Network
    torch.set_num_threads(2)
    out = root / 'candidate'
    launch = load(root / 'launch-plan.json')
    history = load(out / 'history.json')
    epochs = launch['epochs']
    assert [row['epoch'] for row in history] == list(range(1, epochs + 1))
    assert all(row['positions'] == 256000000 and math.isfinite(row['validation_loss']) for row in history)
    checkpoint = torch.load(out / 'best.pt', map_location='cpu', weights_only=True)
    assert checkpoint['config']['king_factorizer'] is True
    baseline_config = load(root / f"{launch['base']}.config.json")
    for key in RECIPE:
        if key != 'resume':
            assert checkpoint['config'][key] == baseline_config[key], key
    assert checkpoint['config']['epochs'] == epochs
    assert checkpoint['config']['archive_split'] == baseline_config['archive_split']
    assert checkpoint['config']['king_map'] == baseline_config['king_map']
    selected = min(history, key=lambda row: row['validation_loss'])
    assert checkpoint['epoch'] == selected['epoch']
    assert checkpoint['step'] == selected['step'] and checkpoint['history'] == history[:selected['epoch']]
    assert checkpoint['model']['w1_shared'].count_nonzero().item() > 0
    assert all(torch.isfinite(tensor).all().item() for tensor in checkpoint['model'].values())
    reexport = out / 'best-reexport.nnue'
    export_state(checkpoint['model'], reexport, 3)
    assert digest(reexport) == digest(out / 'best.nnue')
    net = Network(out / 'best.nnue')
    records = np.fromfile(root / 'frozen/parity-records.ngnp', dtype=RECORD_DTYPE)
    assert len(records) == 20 and (net.version, net.hidden, net.king_buckets, net.buckets) == (3, 512, 8, 8)
    cases = []
    for record in records:
        pos = decode_record(record)
        cases.append(dict(fen=to_fen(pos), eval_cp=net.evaluate(pos), bucket=net.bucket(pos)))
    payload = dict(format='NGNN3', hidden=512, king_buckets=8, buckets=8,
                   network_sha256=digest(out / 'best.nnue'), cases=cases)
    write(root / 'parity-evals.json', payload)
    return dict(network_sha256=payload['network_sha256'], checkpoint_sha256=digest(out / 'best.pt'),
                parity_cases=20, reexport_identical=True, best_epoch=checkpoint['epoch'],
                full_epochs=epochs, shared_rows_nonzero=True)


def selftest():
    # Historical complete logs supply independent positive/neutral/A/A controls.
    r = CAMPAIGN
    assert gate((r / 'gate-kb8h512.log').read_text(), 'gate')['verdict'] == 'eligible'
    assert gate((r / 'gate-kb8h1024.log').read_text(), 'gate')['verdict'] == 'shelved'
    aa = (r / 'aa-kb8h1024.log').read_text()
    assert gate(aa, 'aa')['verdict'] == 'aa-pass'
    rejected = 0
    mutations = [aa.replace('Games: 40', 'Games: 38'), aa.replace('base 0  of', 'base 1  of'),
                 aa + '\nengine crash\n', aa + aa, '\n'.join(aa.splitlines()[1:]).replace('G1 ', 'G2 '),
                 aa.replace('checkmate', 'no-move'), aa.replace('over 20 pairs', 'over 19 pairs'),
                 re.sub(r'(?m)^G1\s+.*\n', '', aa)]
    for text in mutations:
        try:
            gate(text, 'aa')
        except (AssertionError, ValueError):
            rejected += 1
        else:
            raise AssertionError('malformed gate admitted')
    config = load(r / 'kb8h512/config.json')
    recipe(config, 58)
    long = load(r / 'kb8h512-long/config.json')
    recipe(long, 116)
    recipe_rejections = 0
    for key, value in [('epochs', 59), ('seed', 1), ('precision', 'fp32'), ('wdl', .5),
                       ('king_factorizer', True), ('ngnp_holdout_modulus', 1024)]:
        changed = dict(config, **{key: value})
        try:
            recipe(changed, 58)
        except AssertionError:
            recipe_rejections += 1
        else:
            raise AssertionError('changed base recipe admitted')
    for epochs in (58, 116):
        argv = command(Path('/tmp/factorizer-controls'), epochs)
        assert argv.count('--king-factorizer') == 1 and '--resume' not in argv
        assert argv[argv.index('--epochs') + 1] == str(epochs)
    assert [score_to_elo(s) for s in (-1., 0., .0001, .5, .9999, 1., 2.)] == [-800., -800., -800., 0., 800., 800., 800.]
    assert abs(score_to_elo(.75) - 190.848501887865) < 1e-10
    plan_rejections = nomination_controls()
    return dict(passed=True, historical_gate_controls=3, gate_rejections=rejected,
                recipe_rejections=recipe_rejections, nomination_rejections=plan_rejections,
                score_boundary_controls=8, schedule_plans=[58, 116])


def nomination_controls():
    """Synthetic custody fixtures never launch engines, training or matches."""
    import shutil
    rejected = 0
    with tempfile.TemporaryDirectory(prefix='ngn-factorizer-controls-') as directory:
        top = Path(directory)
        for accepted in (False, True):
            campaign, admission, root = (top / str(accepted) / p for p in ('campaign', 'admission', 'trial'))
            for path in (campaign / 'session1', campaign / 'session8b', root / 'frozen', admission,
                         campaign / 'kb8h512', campaign / 'kb8h512-long'):
                path.mkdir(parents=True)
            (campaign / 'status.jsonl').write_text(json.dumps(dict(stage='session1', event='complete'))+'\n')
            (campaign / 'session1/post-gate-input-check.log').write_text('fixture: OK\n')
            for name in ('kb8h512', 'kb8h512-long'):
                shutil.copyfile(CAMPAIGN / name / 'config.json', campaign / name / 'config.json')
                shutil.copyfile(CAMPAIGN / name / 'config.json', root / f'{name}.config.json')
                shutil.copyfile(CAMPAIGN / 'kb8h512/best.nnue', campaign / name / 'best.nnue')
            (campaign / 'kb8h512-long/best.pt').write_bytes(b'synthetic checkpoint custody')
            shutil.copyfile(CAMPAIGN / 'aa-kb8h1024.log', campaign / 'session1/aa.log')
            shutil.copyfile(CAMPAIGN / ('gate-kb8h512.log' if accepted else 'gate-kb8h1024.log'), campaign / 'session1/gate.log')
            parity = dict(parity_cases=20, reexport_identical=True,
                          network_sha256=digest(campaign / 'kb8h512-long/best.nnue'),
                          checkpoint_sha256=digest(campaign / 'kb8h512-long/best.pt'))
            write(campaign / 'session1/parity.json', parity)
            write(admission / 'admission-receipt.json', dict(admitted=True, games=12))
            (admission / 'controller.done').touch()
            (campaign / 'session8b/cuda-controller.done').touch()
            (campaign / 'session8b/cuda-controller.exit').write_text('0\n')
            (campaign / 'session8b/cuda-tests.log').write_text('Ran 1 test\nOK\n')
            write(campaign / 'session8b/cuda-receipt.json', dict(passed=True, log_sha256=digest(campaign / 'session8b/cuda-tests.log')))
            (root / 'frozen/ngn').write_bytes(b'synthetic binary custody')
            build = dict(source_reconciled=True, engine_inputs=230, trainer_inputs=25,
                         source_commit='f'*40, binary_sha256=dict(ngn=digest(root / 'frozen/ngn')))
            write(root / 'build-receipt.json', build)
            evidence = dict(aa=campaign / 'session1/aa.log', gate=campaign / 'session1/gate.log',
                            parity=campaign / 'session1/parity.json', post_gate=campaign / 'session1/post-gate-input-check.log',
                            admission=admission / 'admission-receipt.json', cuda=campaign / 'session8b/cuda-receipt.json')
            selected = 'kb8h512-long' if accepted else 'kb8h512'
            n = dict(reviewed=True, review_commit='f'*40, review_record='experiments/synthetic-review.md',
                     longer_verdict='accepted' if accepted else 'shelved', base=selected, base_net_sha256=BASE_HASH,
                     evidence_sha256={k:digest(p) for k,p in evidence.items()})
            write(root / 'nomination.json', n)
            assert plan(root, campaign, admission)['epochs'] == (116 if accepted else 58)
            for field, value in [('reviewed', False), ('review_commit', 'pending'), ('base', 'other'),
                                 ('base', 'kb8h512' if accepted else 'kb8h512-long'),
                                 ('longer_verdict', 'shelved' if accepted else 'accepted'),
                                 ('base_net_sha256', '0'*64), ('evidence_sha256', {})]:
                write(root / 'nomination.json', dict(n, **{field:value}))
                try:
                    plan(root, campaign, admission)
                except AssertionError:
                    rejected += 1
                else:
                    raise AssertionError('unreviewed or mismatched nomination admitted')
            write(root / 'nomination.json', n)
            # Rebind the altered receipt hash so these test semantic guards,
            # independently of the simpler hash-mismatch rejections above.
            for path, field, value in [(evidence['admission'], 'admitted', False),
                                       (evidence['admission'], 'games', 10),
                                       (evidence['cuda'], 'passed', False),
                                       (evidence['parity'], 'parity_cases', 19),
                                       (evidence['parity'], 'reexport_identical', False)]:
                original = path.read_bytes()
                changed = load(path)
                changed[field] = value
                write(path, changed)
                changed_nomination = dict(n, evidence_sha256={k:digest(p) for k,p in evidence.items()})
                write(root / 'nomination.json', changed_nomination)
                try:
                    plan(root, campaign, admission)
                except AssertionError:
                    rejected += 1
                else:
                    raise AssertionError('failed semantic dependency admitted')
                path.write_bytes(original)
            write(root / 'nomination.json', n)
            (root / 'nomination.json').unlink()
            try:
                plan(root, campaign, admission)
            except FileNotFoundError:
                rejected += 1
            else:
                raise AssertionError('missing nomination admitted')
            write(root / 'nomination.json', n)
            (campaign / 'session8b/cuda-controller.exit').write_text('1\n')
            try:
                plan(root, campaign, admission)
            except AssertionError:
                rejected += 1
            else:
                raise AssertionError('failed CUDA dependency admitted')
            (campaign / 'session8b/cuda-controller.exit').write_text('0\n')
            (root / 'frozen/ngn').write_bytes(b'changed synthetic binary')
            try:
                plan(root, campaign, admission)
            except AssertionError:
                rejected += 1
            else:
                raise AssertionError('changed engine binary admitted')
    return rejected


def main():
    cli = argparse.ArgumentParser(description=__doc__)
    cli.add_argument('mode', choices=('plan', 'parity', 'gate', 'selftest', 'train'))
    cli.add_argument('root', type=Path)
    cli.add_argument('--gate-mode', choices=('aa', 'gate'))
    args = cli.parse_args()
    if args.mode == 'selftest':
        result = selftest()
    elif args.mode == 'gate':
        result = gate(args.root.read_text(), args.gate_mode)
    elif args.mode == 'parity':
        result = parity(args.root)
    else:
        result = plan(args.root)
        if args.mode == 'train':
            assert not (args.root / 'candidate').exists(), 'never restart or overwrite a trial'
            write(args.root / 'launch-plan.json', result)
            subprocess.run(result['trainer_argv'], cwd=args.root / 'source', check=True)
    print(json.dumps(result, allow_nan=False))


if __name__ == '__main__':
    main()
