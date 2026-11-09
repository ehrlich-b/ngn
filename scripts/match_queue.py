#!/usr/bin/env python3
"""Standing, serial cmd/sprt queue and append-only strength ledger (stdlib only)."""
import argparse
from contextlib import contextmanager
import datetime
import fcntl
import hashlib
import json
import math
import os
from pathlib import Path
import re
import signal
import subprocess
import time

REPO = Path(__file__).resolve().parents[1]
CPU_LOCK = Path('/home/ehrli/ngn-data/night-20261007/session138/cpu0-11.lock')


def sha(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(chunk)
    return h.hexdigest()


def identity(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, allow_nan=False).encode()).hexdigest()


def read(path):
    return json.loads(Path(path).read_text())


def clean_env():
    return {k: v for k, v in os.environ.items() if not re.search(
        r'credential|token|secret|password|api.?key|access.?key|authorization|cookie|private.?key', k, re.I)}


def validate(value, schema, where='$'):
    """Validate the JSON Schema keywords used by the two published schemas."""
    kinds = {'object': dict, 'array': list, 'string': str, 'integer': int,
             'number': (int, float)}
    kind = schema.get('type')
    if kind and (not isinstance(value, kinds[kind]) or
                 (kind in ('integer', 'number') and isinstance(value, bool))):
        raise ValueError(f'{where}: expected {kind}')
    if 'const' in schema and value != schema['const']:
        raise ValueError(f'{where}: wrong constant')
    if 'enum' in schema and value not in schema['enum']:
        raise ValueError(f'{where}: invalid choice')
    if isinstance(value, dict):
        missing = set(schema.get('required', [])) - value.keys()
        if missing:
            raise ValueError(f'{where}: missing {sorted(missing)}')
        properties = schema.get('properties', {})
        if schema.get('additionalProperties') is False and value.keys() - properties.keys():
            raise ValueError(f'{where}: unknown fields')
        for key, item in value.items():
            if key in properties:
                validate(item, properties[key], where + '.' + key)
    if isinstance(value, list):
        if len(value) < schema.get('minItems', 0) or len(value) > schema.get('maxItems', math.inf):
            raise ValueError(f'{where}: invalid array length')
        for i, item in enumerate(value):
            validate(item, schema.get('items', {}), f'{where}[{i}]')
    if isinstance(value, str):
        if len(value) < schema.get('minLength', 0) or ('pattern' in schema and
                                                     not re.search(schema['pattern'], value)):
            raise ValueError(f'{where}: invalid string')
    if type(value) in (int, float):
        if (not math.isfinite(value) or value < schema.get('minimum', -math.inf) or
                value > schema.get('maximum', math.inf) or
                value <= schema.get('exclusiveMinimum', -math.inf) or
                value >= schema.get('exclusiveMaximum', math.inf)):
            raise ValueError(f'{where}: invalid number')


def validate_row(row):
    validate(row, read(REPO / 'experiments/ledger.schema.json'))
    if (sum(row['pair_counts']) != row['pairs'] or sum(row['wdl']) != row['games'] or
            row['games'] != 2 * row['pairs'] + row['orphan_games']):
        raise ValueError('ledger pair/game accounting mismatch')
    if row['verdict'] != 'ERROR' and (row['faults'] or row['games'] == 0):
        raise ValueError('scored row has faults or no games')


def rows(path):
    data = Path(path).read_bytes() if Path(path).exists() else b''
    if data and not data.endswith(b'\n'):
        raise ValueError('incomplete ledger line; inspect before continuing')
    result = [json.loads(line) for line in data.splitlines()]
    ids = set()
    for row in result:
        validate_row(row)
        if row['job_id'] in ids:
            raise ValueError('duplicate ledger job ID')
        ids.add(row['job_id'])
    return result


def strength(ledger, output):
    """Render only ledger fields; never read jobs, clocks, binaries or Git."""
    entries = rows(ledger)
    lines = ['# Strength', '', 'Generated from the strength ledger. Relative match results only;',
             'no absolute rating or automatic adoption is implied.', '',
             '| Job | Finished (UTC) | Games | Pairs | Verdict |',
             '| --- | --- | ---: | ---: | --- |']
    for row in entries:
        lines.append(f"| {row['job_id']} | {row['finished_utc']} | {row['games']} | {row['pairs']} | {row['verdict']} |")
    if not entries:
        lines.extend(['', 'No finished matches.'])
    for row in entries:
        lines.extend(['', f"## {row['job_id']}", '', f"Pair counts (LL, LD, LW/DD, WD, WW): {row['pair_counts']}.",
                      f"Orphan games: {row['orphan_games']}. Faults: {json.dumps(row['faults'])}."])
        for key in ('source_hash', 'binary_hash', 'net_hash'):
            for role in ('base', 'candidate'):
                lines.append(f"- {key} {role}: `{row[key][role]}`")
        lines.extend([f"- book_hash: `{row['book_hash']}`", f"- harness_hash: `{row['harness_hash']}`",
                      f"- job_hash: `{row['job_hash']}`"])
    Path(output).write_text('\n'.join(lines) + '\n')


def load_job(path, repo):
    job = read(path)
    validate(job, read(REPO / 'experiments/match-queue/job.schema.json'))
    # cmd/sprt truncates seconds to integer milliseconds; a zero base
    # selects fixed depth instead of the requested clock.
    if float(job['time_control'].split('+')[0]) * 1000 < 1:
        raise ValueError('time control base must be at least 1 millisecond')
    if job['bounds']['elo0'] >= job['bounds']['elo1']:
        raise ValueError('elo0 must be below elo1')
    if (job['min_games'] > job['max_games'] or job['max_games'] % 2 or job['min_games'] % 2):
        raise ValueError('game bounds must be even and ordered')
    for role in ('base', 'candidate'):
        for option in job[role]['options']:
            name, sep, _ = option.partition('=')
            if not sep or not name.strip() or re.search(r'[\r\n]', option):
                raise ValueError('option must be one-line NAME=VALUE')
            if name.strip().lower() in ('evalfile', 'usennue'):
                raise ValueError('EvalFile and UseNNUE are supplied from the pinned net')
        artifact = job[role]['net']
        if sha(repo / artifact['path']) != artifact['sha256']:
            raise ValueError(f'{role} net hash mismatch')
    if sha(repo / job['book']['path']) != job['book']['sha256']:
        raise ValueError('book hash mismatch')
    return job


def command(job, repo, sprt, cpus):
    args = ['taskset', '-c', cpus, str(sprt), '-new', str(repo / job['candidate']['binary']),
            '-base', str(repo / job['base']['binary']), '-tc', job['time_control'],
            '-openings', str(repo / job['book']['path']), '-maxgames', str(job['max_games']),
            '-mingames', str(job['min_games']), '-concurrency', str(job['concurrency']),
            '-lowpower=false', '-noretries', '-maxmoves', '400']
    for key, value in job['bounds'].items():
        args.extend(['-' + key, str(value)])
    for role, flag in (('base', '-baseoption'), ('candidate', '-newoption')):
        for option in job[role]['options'] + [f"EvalFile={repo / job[role]['net']['path']}", 'UseNNUE=true']:
            args.extend([flag, option])
    return args


@contextmanager
def cpu_lock(path):
    # Share the existing inode with SPSA. Never create, truncate or replace it.
    with Path(path).open('r') as stream:
        fcntl.flock(stream, fcntl.LOCK_EX)
        try:
            yield stream
        finally:
            fcntl.flock(stream, fcntl.LOCK_UN)


def summarize(text, returncode):
    result = dict(games=0, pairs=0, orphan_games=0, pair_counts=[0] * 5,
                  wdl=[0] * 3, verdict='ERROR', faults=[])
    if returncode:
        result['faults'].append(f'harness exit {returncode}')
    if re.search(r'WATCHDOG|panic:|fatal error:|search-error|engine-error|no-move|illegal-move|failed UCI|info string error|WARNING: skipping', text):
        result['faults'].append('harness/engine fault')
    flags = re.search(r'Flag-outs \(lost on time\): new (\d+), base (\d+)', text)
    if flags and any(map(int, flags.groups())):
        result['faults'].append('flag-out')
    try:
        # Parse the terminal section only, never a progress line.
        terminal = text.rsplit('=== RESULT (', 1)[1]
        games, *wdl = map(int, re.search(r'Games: (\d+)\s+W-D-L: (\d+)-(\d+)-(\d+)', terminal).groups())
        counts = re.search(r'Pentanomial \[LL (\d+)\s+LD (\d+)\s+\{LW,DD\} (\d+)\s+WD (\d+)\s+WW (\d+)\] over (\d+) pairs', terminal)
        *penta, pairs = map(int, counts.groups())
        if sum(wdl) != games or sum(penta) != pairs or 2 * pairs > games:
            raise ValueError('inconsistent counts')
        result.update(games=games, wdl=wdl, pair_counts=penta, pairs=pairs, orphan_games=games - 2 * pairs)
        native = re.search(r'^Verdict: (.+)$', terminal, re.M).group(1)
        verdict = next(v for prefix, v in [('H1 ACCEPTED:', 'H1'), ('H0 ACCEPTED:', 'H0'),
                                          ('INCONCLUSIVE (', 'INCONCLUSIVE')] if native.startswith(prefix))
        if not games or not pairs:
            result['faults'].append('no completed pairs')
        if not result['faults']:
            result['verdict'] = verdict
    except (AttributeError, IndexError, ValueError, StopIteration):
        result['faults'].append('missing or invalid terminal summary')
    return result


def utc():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def execute(job, args, lock_stream):
    directory = args.output / job['id']
    # An interrupted job requires inspection; never replay potentially live games.
    directory.mkdir()
    argv = command(job, args.repo, args.sprt, args.cpus)
    row = dict(version=1, job_id=job['id'], job_hash=identity(job), started_utc=utc(),
               source_hash={r: job[r]['source_hash'] for r in ('base', 'candidate')},
               binary_hash={r: sha(args.repo / job[r]['binary']) for r in ('base', 'candidate')},
               net_hash={r: job[r]['net']['sha256'] for r in ('base', 'candidate')},
               book_hash=job['book']['sha256'], harness_hash=sha(args.sprt), argv=argv,
               log=str(directory / 'sprt.log'))
    (directory / 'started.json').write_text(json.dumps(row, indent=2) + '\n')
    with (directory / 'sprt.log').open('x') as log:
        # The harness inherits the flock FD: it retains CPU ownership if the
        # controller is killed before systemd finishes stopping the process group.
        child = subprocess.Popen(argv, cwd=args.repo, env=clean_env(), stdout=log,
                                 stderr=subprocess.STDOUT, pass_fds=(lock_stream.fileno(),),
                                 start_new_session=True)
        try:
            rc = child.wait()
        except BaseException:
            # Stop the harness and its engines together, also outside systemd.
            try:
                os.killpg(child.pid, signal.SIGTERM)
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(child.pid, signal.SIGKILL)
                child.wait()
            except ProcessLookupError:
                child.wait()
            raise
    row.update(summarize((directory / 'sprt.log').read_text(), rc), finished_utc=utc())
    validate_row(row)
    with args.ledger.open('a') as ledger:
        ledger.write(json.dumps(row, sort_keys=True, allow_nan=False) + '\n')
        ledger.flush()
        os.fsync(ledger.fileno())
    print(json.dumps(row), flush=True)
    return row


def scan(args):
    # Lock before reading completion state so concurrent services cannot duplicate
    # a job. Keep the lock through ledger append and report regeneration.
    with cpu_lock(args.lock) as lock_stream:
        completed = {r['job_id']: r for r in rows(args.ledger)}
        for path in sorted(args.jobs.glob('*.json')):
            job = load_job(path, args.repo)
            if job['id'] in completed:
                if completed[job['id']]['job_hash'] != identity(job):
                    raise ValueError('completed job changed; use a new ID')
                continue
            row = execute(job, args, lock_stream)
            completed[job['id']] = row
            strength(args.ledger, args.strength)
            if row['verdict'] == 'ERROR':
                raise ValueError(f"job {job['id']} failed; see {row['log']}")
        strength(args.ledger, args.strength)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='action', required=True)
    for name in ('validate', 'strength'):
        p = sub.add_parser(name)
        p.add_argument('--ledger', type=Path, default=REPO / 'experiments/ledger.jsonl')
        if name == 'strength':
            p.add_argument('--output', type=Path, default=REPO / 'docs/STRENGTH.md')
    p = sub.add_parser('run')
    p.add_argument('--repo', type=Path, default=REPO)
    p.add_argument('--jobs', type=Path, default=REPO / 'experiments/match-queue/jobs')
    p.add_argument('--ledger', type=Path, default=REPO / 'experiments/ledger.jsonl')
    p.add_argument('--strength', type=Path, default=REPO / 'docs/STRENGTH.md')
    p.add_argument('--sprt', type=Path, default=REPO / 'build/sprt')
    p.add_argument('--output', type=Path, default=Path.home() / '.local/state/ngn-match-queue/runs')
    p.add_argument('--lock', type=Path, default=CPU_LOCK)
    p.add_argument('--cpus', default='0-11')
    p.add_argument('--once', action='store_true')
    p.add_argument('--dry-run', action='store_true')
    args = parser.parse_args()
    if args.action == 'validate':
        print(f'{len(rows(args.ledger))} valid ledger rows')
    elif args.action == 'strength':
        strength(args.ledger, args.output)
    elif args.dry_run:
        paths = sorted(args.jobs.glob('*.json'))
        if not paths:
            raise ValueError('no job files')
        for path in paths:
            job = load_job(path, args.repo)
            print(json.dumps(dict(job=job['id'], argv=command(job, args.repo, args.sprt, args.cpus))))
    else:
        args.output.mkdir(parents=True, exist_ok=True)
        # User units send TERM to the entire control group. Raise so the child
        # is reaped and the shared CPU lock is released during shutdown.
        def stop(signum, frame):
            raise KeyboardInterrupt
        signal.signal(signal.SIGTERM, stop)
        while True:
            scan(args)
            if args.once:
                break
            time.sleep(1)


if __name__ == '__main__':
    main()
