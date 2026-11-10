#!/usr/bin/env python3
"""Generate opt-in engine defaults from accepted, recipe-bound ledger rows."""
import argparse
from pathlib import Path
import re

from scripts import match_queue as mq

JOBS = mq.REPO / 'experiments/option-verdict/jobs'
PREREQUISITES = {'W2NullMargin': 'W2NullVerify'}


def options():
    # Keep the UCI registry authoritative; fail if its declaration stops matching.
    source = (mq.REPO / 'engine/search.go').read_text()
    registry = source.split('var TunableSearchParams = []TunableParam{', 1)[1].split('\n}', 1)[0]
    names = re.findall(r'\{"((?:T1104|W2)[^"]+)", &\w+, [01], 0, 1\}', registry)
    if not names or len(names) != len(set(names)):
        raise ValueError('invalid experimental option registry')
    return names + ['W2ScoreTime']


def recipes(directory=JOBS):
    names = options()
    jobs = {}
    covered = []
    for path in sorted(Path(directory).glob('*.json')):
        job = mq.load_job(path, mq.REPO)
        base, candidate = job['base'], job['candidate']
        off = ['Threads=1', 'Hash=64', 'OwnBook=false'] + [n + '=0' for n in names]
        changed = []
        for name in names:
            prerequisite = PREREQUISITES.get(name)
            control = [prerequisite + '=1' if o == prerequisite + '=0' else o
                       for o in off] if prerequisite else off
            treatment = [name + '=1' if o == name + '=0' else o for o in control]
            if base['options'] == control and candidate['options'] == treatment:
                changed.append(name)
        if len(changed) != 1 or {k: v for k, v in base.items() if k != 'options'} != {
                k: v for k, v in candidate.items() if k != 'options'}:
            raise ValueError('recipe must change exactly one experimental option')
        if job['bounds'] != dict(elo0=0, elo1=5, alpha=0.05, beta=0.05) or job['time_control'] != '10+0.1':
            raise ValueError('expected SPRT [0,5] at 10+0.1')
        if job['id'] in jobs:
            raise ValueError('duplicate job ID')
        jobs[job['id']] = (job, changed[0])
        covered.extend(changed)
    if covered != names:
        raise ValueError('expected one job per option in registry order, ContPlies first')
    return jobs


def artifact_hashes(job, artifacts):
    """Use local artifacts or an independently trusted manifest, never the row."""
    if artifacts is None:
        return ({r: mq.sha(mq.REPO / job[r]['binary']) for r in ('base', 'candidate')},
                mq.sha(mq.REPO / 'build/sprt'))
    manifest = mq.read(artifacts)
    digest = {'type': 'string', 'pattern': '^[0-9a-f]{64}$'}
    binaries = {job[r]['binary']: digest for r in ('base', 'candidate')}
    mq.validate(manifest, {
        'type': 'object', 'additionalProperties': False,
        'required': ['version', 'binary_hash', 'harness_hash'],
        'properties': {
            'version': {'type': 'integer', 'const': 1},
            'binary_hash': {'type': 'object', 'required': list(binaries),
                            'properties': binaries, 'additionalProperties': False},
            'harness_hash': digest,
        },
    })
    return ({r: manifest['binary_hash'][job[r]['binary']] for r in ('base', 'candidate')},
            manifest['harness_hash'])


def defaults(ledger, directory=JOBS, artifacts=None):
    jobs = recipes(directory)
    enabled = {}
    for row in mq.rows(ledger):
        if row['job_id'] not in jobs:
            continue  # Other queue families never authorize experimental defaults.
        job, name = jobs[row['job_id']]
        if row['job_hash'] != mq.identity(job):
            raise ValueError('ledger job hash does not match recipe')
        # Both sides of these recipes name the same executable. Checking each
        # role against one independent identity also rejects equal forged hashes.
        binary_hash, harness_hash = artifact_hashes(job, artifacts)
        for key, expected in (
                ('source_hash', {r: job[r]['source_hash'] for r in ('base', 'candidate')}),
                ('binary_hash', binary_hash),
                ('harness_hash', harness_hash),
                ('net_hash', {r: job[r]['net']['sha256'] for r in ('base', 'candidate')}),
                ('book_hash', job['book']['sha256'])):
            if row[key] != expected:
                raise ValueError('ledger provenance mismatch: ' + key)
        validate_command(job, row['argv'])
        if row['verdict'] == 'H1':
            if row['pairs'] < 1 or row['faults'] or row['games'] < job['min_games']:
                raise ValueError('acceptance requires fault-free pairs and the recipe minimum games')
            enabled[name] = 1
    return enabled


def validate_command(job, argv):
    # The execution checkout can differ from this checkout. Infer just that
    # prefix, then reconstruct the ENTIRE command; do not strip arbitrary paths
    # or normalize options, executable names, bounds, or artifact identities.
    suffix = '/' + job['candidate']['binary']
    if len(argv) < 6 or not argv[5].endswith(suffix):
        raise ValueError('ledger command does not match recipe')
    prefix = argv[5][:-len(suffix)] or '/'
    recorded_repo = Path(prefix)
    if (not recorded_repo.is_absolute() or '..' in recorded_repo.parts or
            str(recorded_repo) != prefix):
        raise ValueError('invalid recorded checkout path')
    # These option recipes use the queue's default harness. Accepting an
    # arbitrary argv[3] here would also accept an altered executable identity.
    expected = mq.command(job, recorded_repo, recorded_repo / 'build/sprt', argv[2])
    if argv != expected:
        raise ValueError('ledger command does not match recipe')


def render(enabled):
    lines = ['// Code generated by scripts/option_verdict.py; DO NOT EDIT.',
             'package engine', '']
    if not enabled:
        return '\n'.join(lines + ['var acceptedOptionDefaults = map[string]int{}', ''])
    lines.append('var acceptedOptionDefaults = map[string]int{')
    lines += [f'\t"{name}": 1,' for name in sorted(enabled)]
    return '\n'.join(lines + ['}', ''])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--ledger', type=Path, required=True)
    parser.add_argument('--jobs', type=Path, default=JOBS)
    parser.add_argument('--artifacts', type=Path,
                        help='independently trusted binary/harness digest manifest; '
                             'otherwise hash local recipe binaries and build/sprt')
    parser.add_argument('--output', type=Path, required=True,
                        help='Go source output; adoption requires explicit source update and rebuild')
    args = parser.parse_args()
    # Validate everything before touching the output. No games or services run.
    result = render(defaults(args.ledger, args.jobs, args.artifacts))
    args.output.write_text(result)


if __name__ == '__main__':
    main()
