"""Calibrate approved T60/T70 data, then freeze its full-sample budget.

Reuse the external archive decoder/map unchanged. No engine/trainer changes,
downloads, old-data deletion, automatic training or games.
"""
import datetime
import json
import math
import os
from pathlib import Path
import signal
import subprocess

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session114-data'
OLD = R/'session19-data'
A = Path('/home/ehrli/nnue-owned-k4-20260920/archive-a1')
X = OLD/'tool/target/release/ngnk4archive'
SOURCE = Path('/home/ehrli/ngn-data/sources/T60T70wIsRightFarseer.binpack')
CONFLICTS = Path('/home/ehrli/nnue-owned-k4-20260920/runs/k4-production-pilot-20260921/selection/conflicts.bin')
GIB = 1024**3


def write(name, value):
    (D/name).write_text(json.dumps(value, indent=2, allow_nan=False)+'\n')


def status(event, detail):
    with (R/'status.jsonl').open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
            stage='session114-data', event=event, detail=detail))+'\n')


def host_free():
    disk = os.statvfs('/mnt/c')
    return disk.f_bavail * disk.f_frsize


def run(argv, name):
    status(name+'-start', dict(argv=list(map(str, argv))))
    with (D/(name+'.log')).open('w') as log:
        child = subprocess.Popen(list(map(str, argv)), stdout=log, stderr=subprocess.STDOUT,
                                 start_new_session=True)
        (D/'child.pid').write_text(str(child.pid)+'\n')
        try:
            while True:
                if host_free() < 27.5*GIB:
                    raise RuntimeError('Host C below27.5GiB safety margin; preserve partial calibration')
                try:
                    rc = child.wait(timeout=60)
                    break
                except subprocess.TimeoutExpired:
                    pass
        except BaseException:
            if child.poll() is None:
                os.killpg(child.pid, signal.SIGTERM)
                child.wait(timeout=30)
            raise
    if rc:
        raise RuntimeError(name+' exit '+str(rc))
    status(name+'-complete', dict(returncode=rc))


def scatter(output, keep):
    return [X, 'scatter', '--input', SOURCE, '--conflicts', CONFLICTS,
        '--probe-keys', A/'calibration-probe.keys', '--output', output,
        '--buckets', '256', '--workers', '2', '--seed', '2026101801',
        '--keep-per-mille', str(keep)]


def gather(scatter_dir, scale, name):
    return [X, 'gather', '--scatter', str(scatter_dir)+'@'+str(scale),
        '--map', A/'score-map.json', '--output', D/(name+'.bf'),
        '--manifest', D/(name+'-manifest.json'), '--seed', '2026101811',
        '--consume-buckets', 'true']


def main():
    (D/'controller.pid').write_text(str(os.getpid())+'\n')
    nomination = json.loads((D/'nomination.json').read_text())
    assert nomination['calibration_keep_per_mille'] == 1
    assert host_free() >= 30*GIB
    # A key-stable 1/1000 slice also supplies complete source statistics in
    # the same decode. The pinned converter checks the source at use.
    run(scatter(D/'scatter-calibration', 1), 'scatter-calibration')
    stats = json.loads((D/'scatter-calibration/scatter.json').read_text())
    counts = stats['counts']
    eligible = counts['eligible_train'] - counts['excluded_conflict_key']
    assert eligible > 0 and counts['emitted'] > 0
    run(['python3', OLD/'fit-source-scale.py', '--reference',
        OLD/'scatter-t80stats/scatter.json', '--source',
        't60t70='+str(D/'scatter-calibration/scatter.json'), '--out', D/'scale.json'], 'scale')
    scale = json.loads((D/'scale.json').read_text())['sources']['t60t70']['raw_scale']
    assert math.isfinite(scale) and scale > 0
    largest = max(stats['bucket_records'])
    raw_bytes = counts['emitted']*32
    run(gather(D/'scatter-calibration', scale, 'calibration'), 'gather-calibration')
    mapped = json.loads((D/'calibration-manifest.json').read_text())['train']
    assert (D/'calibration.bf').stat().st_size == mapped['bytes']
    # The existing sampler uses a uint16 modulo1000. For keep<=536, each
    # retained residue has66 preimages, rather than assuming exactly k/1000.
    # Target995M leaves sampling noise margin below the approved1B cap.
    keep = min(536, math.floor(995_000_000*65536/(66*eligible)))
    assert 1 <= keep < 536
    projected_records = eligible*keep*66/65536
    full_bytes = 1_000_000_000*32
    # Consuming gather never holds a second full copy. Allow twice the
    # observed largest bucket, scaled to1B, plus2GiB for bounded overhead.
    scratch = math.ceil(2*largest/counts['emitted']*full_bytes) + 2*GIB
    required = 25*GIB + full_bytes + scratch + 10*GIB
    plan = dict(state='CALIBRATED_SAMPLE_FROZEN', source=stats['source'],
        calibration=dict(raw_records=counts['emitted'], raw_bytes=raw_bytes,
            mapped=mapped, largest_bucket_records=largest,
            converter_seconds=stats['elapsed_seconds'], raw_scale=scale),
        eligible_nonconflict=eligible, keep_per_mille=keep,
        expected_retained_records=projected_records, maximum_records=1_000_000_000,
        sampling='u16le(K4 key[0:2])%1000<keep; fraction frozen before training; no score selection',
        sample_argv=list(map(str, scatter(D/'scatter-full', keep))),
        gather_argv=list(map(str, gather(D/'scatter-full', scale, 't60t70'))),
        peak_new_output_bytes=full_bytes, scratch_budget_bytes=scratch,
        training_headroom_bytes=10*GIB, host_c_required_bytes=required,
        host_c_free_bytes=host_free(), automatic_training=False,
        next='Review calibrated source/scales and capacity; run frozen sample, then board overlap and unchanged115-epoch training manifest')
    write('sample-plan.json', plan)
    # Bulk conversion is approved, but the disk budget must cover worst-case
    # VHDX growth: deleting ext4 files did not reliably free host space.
    if host_free() < required:
        plan['state'] = 'CALIBRATED_CAPACITY_HOLD'
        write('result.json', plan)
        status('capacity-hold', dict(required_gib=required/GIB, available_gib=host_free()/GIB,
            keep_per_mille=keep, eligible=eligible))
        return
    run(scatter(D/'scatter-full', keep), 'scatter-full')
    full = json.loads((D/'scatter-full/scatter.json').read_text())
    assert 0 < full['counts']['emitted'] <= 1_000_000_000
    run(gather(D/'scatter-full', scale, 't60t70'), 'gather-full')
    plan.update(state='CORPUS_READY_FOR_OVERLAP', train=json.loads((D/'t60t70-manifest.json').read_text())['train'])
    write('result.json', plan)
    status('corpus-ready', dict(result=str(D/'result.json'), automatic_training=False))


if __name__ == '__main__':
    try:
        main()
    except BaseException as error:
        write('result.json', dict(state='HELD_REVIEW', error=str(error), automatic_training=False))
        status('held-review', str(error))
        raise
