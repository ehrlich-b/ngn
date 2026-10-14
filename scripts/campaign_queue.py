#!/usr/bin/env python3
"""Run rig nominations serially: one manifest, result JSON and PGN per job.

The native cmd/sprt owns first crossing and natural stop. This runner never
times out or administratively interrupts a gate, and never adopts engine code.
Passing A/A results are reusable by machine/topology/clock/harness key.
"""
import argparse
import datetime
import hashlib
import json
import math
import os
from pathlib import Path
import re
import subprocess


def write(path, value):
    temporary = path.with_suffix('.tmp')
    temporary.write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')
    temporary.replace(path)


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def paired(penta):
    pairs = sum(penta)
    mean = sum(k * count / 4 for k, count in enumerate(penta)) / pairs
    variance = max(sum((k / 4) ** 2 * count for k, count in enumerate(penta)) / pairs - mean ** 2, .001)
    error = 1.96 * math.sqrt(variance / pairs)

    def elo(score):
        if score <= 1e-4:
            return -800
        if score >= 1 - 1e-4:
            return 800
        return 400 * math.log10(score / (1 - score))

    return [elo(mean), elo(mean - error), elo(mean + error)]


def exact_llr(penta, elo0, elo1):
    pairs = sum(penta)
    total = sum(k * count / 4 for k, count in enumerate(penta))
    mean = total / pairs
    variance = max(sum((k / 4) ** 2 * count for k, count in enumerate(penta)) / pairs - mean ** 2, .001)
    score0, score1 = (1 / (1 + 10 ** (-value / 400)) for value in (elo0, elo1))
    return (score1 - score0) / variance * (total - pairs * (score0 + score1) / 2)


def summarize(log, pgn, job, returncode):
    games, wins, draws, losses = map(int, re.search(r'Games: (\d+)\s+W-D-L: (\d+)-(\d+)-(\d+)', log).groups())
    penta = list(map(int, re.search(r'Pentanomial \[LL (\d+)\s+LD (\d+)\s+\{LW,DD\} (\d+)\s+WD (\d+)\s+WW (\d+)\]', log).groups()))
    elo, low, high, llr = map(float, re.search(r'Penta Elo: ([-+\d.]+)\s+95% CI \[([-+\d.]+), ([-+\d.]+)\]\s+pLLR ([-+\d.]+)', log).groups())
    flags = list(map(int, re.search(r'Flag-outs \(lost on time\): new (\d+), base (\d+)', log).groups()))
    native = re.search(r'^Verdict: (.+)$', log, re.M).group(1)
    faults = []
    if returncode:
        faults.append(f'harness exit{returncode}')
    if any(flags):
        faults.append('flag-out')
    if re.search(r'WATCHDOG|panic:|fatal error:|no-move|illegal-move|PGN capture:|failed UCI', log):
        faults.append('harness/engine operational fault')
    tags = [dict(re.findall(r'^\[(\w+) "(.*)"\]$', block, re.M))
            for block in re.split(r'(?=\[Event )', pgn) if block.strip()]
    reasons, scores, wdl = {}, {}, [0, 0, 0]
    for tag in tags:
        reason = tag['Termination']
        reasons[reason] = reasons.get(reason, 0) + 1
        if reason not in ('checkmate', 'stalemate', 'draw-rule', 'max-moves', 'insufficient-material'):
            faults.append(reason)
        score = .5
        if tag['Result'] != '1/2-1/2':
            score = float((tag['Result'] == '1-0') == (tag['White'] == 'candidate'))
        wdl[0 if score == 1 else 1 if score == .5 else 2] += 1
        scores.setdefault(tag['Round'].split('.')[0], []).append(score)
    recounted = [0] * 5
    for outcomes in scores.values():
        if len(outcomes) == 2:
            recounted[round(2 * sum(outcomes))] += 1
        elif len(outcomes) > 2:
            faults.append('duplicate pair')
    if len(tags) != games or wdl != [wins, draws, losses] or recounted != penta:
        faults.append('PGN/result pair accounting mismatch')
    exact = paired(penta)
    unrounded_llr = exact_llr(penta, job.get('elo0', 0), job.get('elo1', 5))
    kind = job['class']
    full = games == job['max_games'] and sum(penta) * 2 == games
    if kind in ('aa', 'measurement', 'net', 'net-transfer', 'correctness') and not full:
        faults.append('incomplete fixed sample')
    verdict = 'UNSCORED' if faults else 'SHELVED'
    if not faults:
        if kind == 'aa':
            verdict = 'AA_PASS' if low <= 0 <= high and exact[1] <= 0 <= exact[2] else 'AA_FAIL'
        elif kind == 'measurement':
            verdict = 'MEASURED'
        elif kind == 'correctness':
            verdict = 'REGRESSION' if exact[2] < 0 else 'CORRECTNESS_KEEP'
        elif kind == 'net-transfer':
            verdict = 'TRANSFER_PASS' if exact[0] > 0 else 'TRANSFER_FAIL'
        elif kind == 'net':
            verdict = 'KEEP' if low > 0 and exact[1] > 0 else 'SHELVED'
        elif kind in ('search', 'provenance'):
            if native.startswith('H1 ACCEPTED'):
                verdict = 'CERTIFIED' if kind == 'provenance' else 'KEEP'
            elif native.startswith('H0 ACCEPTED'):
                verdict = 'BISECT' if kind == 'provenance' else 'SHELVED'
            elif kind == 'provenance':
                verdict = 'UNCERTIFIED_CAP'
            elif full and exact[0] >= 1 and unrounded_llr > 0 and exact[2] >= 0:
                verdict = 'PROVISIONAL'
    return dict(games=games, pairs=sum(penta), orphan_games=games - sum(penta) * 2,
                wdl=[wins, draws, losses], penta=penta, printed_elo_ci95=[elo, low, high],
                exact_elo_ci95=exact, final_pllr=llr, exact_pllr=unrounded_llr, native_verdict=native,
                flags=flags, reasons=reasons, faults=sorted(set(faults)), verdict=verdict)


def run(job, registry_path, status_path):
    directory = Path(job['output'])
    directory.mkdir(parents=True, exist_ok=True)
    result_path = directory / 'result.json'
    if result_path.exists():
        return json.loads(result_path.read_text())
    if (directory / 'manifest.json').exists():
        raise RuntimeError(f'{directory}: existing unfinished run; inspect its process, never restart it')
    registry = json.loads(registry_path.read_text()) if registry_path.exists() else {}
    if job['class'] != 'aa' and job['control_key'] not in registry:
        raise RuntimeError(f"missing passing A/A: {job['control_key']}")
    # Pin each artifact once where it is used. No post-game custody refresh.
    for path, expected in job['artifacts'].items():
        if digest(path) != expected:
            raise RuntimeError(f'input mismatch: {path}')
    minimum = job['min_games'] if job['class'] in ('search', 'provenance') else 1000000
    argv = ['taskset', '-c', job['cpus'], job['harness'], '-new', job['candidate'], '-base', job['base'],
            '-tc', job['tc'], '-concurrency', str(job['concurrency']), '-mingames', str(minimum),
            '-maxgames', str(job['max_games']), '-elo0', str(job.get('elo0', 0)), '-elo1', str(job.get('elo1', 5)),
            '-alpha', '.05', '-beta', '.05', '-maxmoves', '400', '-gametimeout', str(job.get('game_timeout', 180)),
            '-openings', job['book'], '-lowpower=false', '-pgn', str(directory / 'games.pgn')]
    for role, options in (('new', job['candidate_options']), ('base', job['base_options'])):
        for option in options:
            argv += ['-' + role + 'option', option]
    env = os.environ.copy()
    env['GOMAXPROCS'] = '1'
    manifest = dict(job, argv=argv, environment={'GOMAXPROCS': '1'}, pid=os.getpid(),
                    started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                    reused_control=registry.get(job.get('control_key')))
    write(directory / 'manifest.json', manifest)

    def status(event, detail):
        with status_path.open('a') as stream:
            stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                         stage='campaign-queue', event=event, detail=detail)) + '\n')

    status('start', {'job': job['name'], 'manifest': str(directory / 'manifest.json')})
    with (directory / 'match.log').open('w') as log:
        child = subprocess.Popen(argv, env=env, cwd=job.get('cwd', str(directory)),
                                 stdout=log, stderr=subprocess.STDOUT)
        (directory / 'match.pid').write_text(str(child.pid) + '\n')
        rc = child.wait()
    try:
        result = summarize((directory / 'match.log').read_text(),
                           (directory / 'games.pgn').read_text(), job, rc)
    except Exception as error:
        result = dict(verdict='UNSCORED', faults=[str(error)], harness_exit=rc)
    result.update(name=job['name'], finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                  manifest=str(directory / 'manifest.json'), automatic_adoption=False)
    write(result_path, result)
    status('result', result)
    if job['class'] == 'aa' and result['verdict'] == 'AA_PASS':
        registry[job['control_key']] = dict(result=str(result_path), harness_sha256=job['artifacts'][job['harness']])
        write(registry_path, registry)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('queue', type=Path)
    args = parser.parse_args()
    queue = json.loads(args.queue.read_text())
    for job in queue['jobs']:
        result = run(job, Path(queue['controls']), Path(queue['status']))
        if result['verdict'] in ('UNSCORED', 'AA_FAIL', 'REGRESSION'):
            raise SystemExit(f"queue stopped at {job['name']}: {result['verdict']}")
        if job.get('review_after'):
            break


if __name__ == '__main__':
    main()
