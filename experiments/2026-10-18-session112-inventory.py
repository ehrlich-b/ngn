"""Read campaign file metadata and small manifests; never read corpus payloads.

Run only on rig CPUs 13,15. Output stays in session112. No deletion, download,
hash refresh, engine, training, or match is performed.
"""
import collections
import csv
import datetime
import gzip
import json
import os
from pathlib import Path
import re
import stat
import time

ROOT = Path('/home/ehrli/ngn-data')
CAMPAIGN = ROOT / 'night-20261007'
OUT = CAMPAIGN / 'session112'
NOW = time.time()
GIB = 1024**3
DATA_SUFFIXES = {'.bf', '.binpack', '.ngnp', '.pt', '.nnue', '.npy', '.npz',
                 '.idx', '.fen', '.epd', '.pgn', '.h5', '.parquet'}
SKIP_REF_DIRS = {'.git', '__pycache__', 'candidate', 'base', 'source',
                 'training-source', 'profile-source', 'accepted-source',
                 'bin-src', 'src', 'repo', 'vendor', 'node_modules'}
ACCEPTED_INPUTS = {
    str(CAMPAIGN / 'session19-data/t76.bf'),
    str(CAMPAIGN / 'session57-data-step2/t74full.bf'),
    '/home/ehrli/nnue-owned-k4-20260920/archive-a1/corpus-a1/train-archive.bf',
}
ACCEPTED_NET = str(CAMPAIGN / 'session82/net-final115/candidate.nnue')
ACCEPTED_FINAL = {str(CAMPAIGN / 'session80-training/trained' / name)
                  for name in ('latest.pt', 'latest.nnue', 'final-reexport.nnue')}
LIVE_MANIFESTS = [
    CAMPAIGN / 'session109/nomination.json',
    CAMPAIGN / 'session109/spsa-planned-pause.json',
    CAMPAIGN / 'session88/spsa-warm/nomination.json',
    CAMPAIGN / 'session108-speed/s4-profile-admission.json',
    CAMPAIGN / 'session108-speed/s4-timing/manifest.json',
    CAMPAIGN / 'session108-speed/profile-manifest.json',
]


def absolute_paths(value):
    if isinstance(value, str):
        yield from re.findall(r'/home/ehrli/[^\s\"\x27,;\]\[{}()]+', value)
    elif isinstance(value, dict):
        for k, v in value.items():
            yield from absolute_paths(k)
            yield from absolute_paths(v)
    elif isinstance(value, list):
        for v in value:
            yield from absolute_paths(v)


def manifest_refs(paths):
    result = collections.defaultdict(set)
    scanned = []
    for p in paths:
        try:
            if p.stat().st_size > 4 * 1024**2:
                continue
            value = json.loads(p.read_text())
        except (OSError, ValueError, UnicodeError):
            continue
        scanned.append(str(p))
        for target in absolute_paths(value):
            result[target].add(str(p))
    return result, scanned


def creator(relative):
    if relative.parts[0] != 'night-20261007':
        return 'predates-3600-campaign-or-unverified'
    match = re.match(r'session(\d+)', relative.parts[1])
    return 'session' + match[1] + ' (path inference)' if match else 'campaign (session unverified)'


def data_class(path):
    if path.suffix in {'.bf', '.binpack', '.ngnp', '.h5', '.parquet'}:
        return 'corpus'
    if path.suffix in {'.nnue', '.pt'}:
        return 'model'
    if path.suffix in {'.pgn'}:
        return 'game-evidence'
    if path.suffix in {'.fen', '.epd'}:
        return 'positions'
    return 'derived-array'


def main():
    assert set(os.sched_getaffinity(0)) <= {13, 15}
    OUT.mkdir(exist_ok=True)
    live, live_scanned = manifest_refs(LIVE_MANIFESTS)
    historical_paths = []
    for directory, dirs, files in os.walk(ROOT):
        dirs[:] = [d for d in dirs if d not in SKIP_REF_DIRS and not d.endswith('-source')]
        for name in files:
            if name.endswith('.json') and any(t in name for t in ('manifest', 'nomination', 'config', 'receipt', 'descriptor', 'source')):
                historical_paths.append(Path(directory) / name)
    historical, scanned = manifest_refs(historical_paths)
    groups = collections.defaultdict(lambda: dict(files=0, bytes=0, allocated_bytes=0))
    inodes = set()
    unique_allocated = 0
    total = 0
    data = []
    full_path = OUT / 'all-files.tsv.gz'
    fields = ['path', 'bytes', 'allocated_bytes', 'mtime_utc', 'device', 'inode', 'links', 'kind']
    with gzip.open(full_path, 'wt', newline='') as stream:
        writer = csv.DictWriter(stream, fields, delimiter='\t')
        writer.writeheader()
        for directory, dirs, files in os.walk(ROOT):
            if Path(directory) == OUT:
                dirs[:] = []
                continue
            for name in files:
                p = Path(directory) / name
                try:
                    s = p.lstat()
                except FileNotFoundError:
                    continue
                rel = p.relative_to(ROOT)
                group = '/'.join(rel.parts[:2]) if rel.parts[0] == 'night-20261007' else rel.parts[0]
                kind = 'regular' if stat.S_ISREG(s.st_mode) else 'symlink-or-special'
                mtime = datetime.datetime.fromtimestamp(s.st_mtime, datetime.timezone.utc).isoformat()
                row = dict(path=str(p), bytes=s.st_size, allocated_bytes=s.st_blocks * 512,
                           mtime_utc=mtime, device=s.st_dev, inode=s.st_ino, links=s.st_nlink, kind=kind)
                writer.writerow(row)
                total += 1
                for key, amount in [('files', 1), ('bytes', s.st_size), ('allocated_bytes', s.st_blocks * 512)]:
                    groups[group][key] += amount
                if (s.st_dev, s.st_ino) not in inodes:
                    inodes.add((s.st_dev, s.st_ino))
                    unique_allocated += s.st_blocks * 512
                if p.suffix not in DATA_SUFFIXES or kind != 'regular':
                    continue
                protected = []
                if str(p) in ACCEPTED_INPUTS:
                    protected.append('accepted-net-training-input')
                if str(p) == ACCEPTED_NET:
                    protected.append('accepted-net')
                if str(p) in ACCEPTED_FINAL:
                    protected.append('accepted-net-final115-checkpoint-or-export')
                if str(p) in live:
                    protected.append('live-manifest')
                if NOW - s.st_mtime < 86400:
                    protected.append('modified-within-24h')
                if rel.parts[0] != 'night-20261007':
                    protected.append('outside-current-campaign-delete-scope')
                row.update(creating_session=creator(rel), data_class=data_class(p),
                           protected=','.join(protected) or 'none-established',
                           live_references=sorted(live.get(str(p), [])),
                           historical_references=sorted(historical.get(str(p), [])))
                data.append(row)
    data_fields = fields + ['creating_session', 'data_class', 'protected', 'live_references', 'historical_references']
    with (OUT / 'data-files.tsv').open('w', newline='') as stream:
        writer = csv.DictWriter(stream, data_fields, delimiter='\t')
        writer.writeheader()
        for row in data:
            writer.writerow({k: json.dumps(v) if isinstance(v, list) else v for k, v in row.items()})
    summary = dict(
        observed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        read_only=True, affinity=sorted(os.sched_getaffinity(0)),
        root=str(ROOT), files=total, data_files=len(data),
        bytes_by_path=sum(v['bytes'] for v in groups.values()),
        unique_inode_allocated_bytes=unique_allocated,
        all_files_inventory=str(full_path), data_inventory=str(OUT / 'data-files.tsv'),
        accepted_external_inputs=sorted(ACCEPTED_INPUTS - {p['path'] for p in data}),
        live_manifests_scanned=live_scanned, historical_manifests_scanned=scanned,
        groups=sorted([dict(path=k, **v) for k, v in groups.items()], key=lambda r: -r['bytes']),
        large_data=sorted([r for r in data if r['bytes'] >= 1024**2], key=lambda r: -r['bytes']),
        limitations=[
            'No training payloads read or hashed; manifest path references are static, not a proof that an unreferenced file is disposable.',
            'Creating session is inferred from the containing path, not a verified file birth record.',
            'Reference scan excludes source snapshots and JSON files over4MiB; full manifest list is recorded.',
            'All current-campaign data files are listed; pre-campaign files are inventoried but ineligible for deletion.',
            'Filesystem release does not imply immediate host C: recovery; VHDX may require an approved boundary/compaction.',
        ])
    (OUT / 'inventory-summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    print(json.dumps(dict(files=total, data_files=len(data), unique_allocated_gib=unique_allocated/GIB,
                          inventory_bytes=full_path.stat().st_size,
                          data_inventory_bytes=(OUT/'data-files.tsv').stat().st_size,
                          largest_data=[dict(path=r['path'], gib=r['bytes']/GIB, protected=r['protected']) for r in summary['large_data'][:12]])))


if __name__ == '__main__':
    main()
