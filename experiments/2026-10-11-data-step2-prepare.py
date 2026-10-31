"""Convert the next fixed data volume on forge; publish exact read descriptors."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import sys

s = Path('/work/lanes/ngn-data-step2-20261011')
old = Path('/work/lanes/ngn-data-volume-20261010')
os.chdir(s)


def status(event, detail):
    with (s / 'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                               stage='session57-data-step2', event=event, detail=detail)) + '\n')


def run(argv, name):
    with (s / (name + '.log')).open('w') as log:
        child = subprocess.Popen(argv, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
        try:
            while True:
                disk = os.statvfs(s)
                if disk.f_bavail * disk.f_frsize < 20 * 1024**3:
                    raise RuntimeError('/work reserve crossed20GiB; preserve interrupted conversion')
                try:
                    rc = child.wait(timeout=60)
                    break
                except subprocess.TimeoutExpired:
                    pass
        except BaseException:
            if child.poll() is None:
                os.killpg(child.pid, signal.SIGTERM)
                child.wait(timeout=20)
            raise
    assert rc == 0, (name, rc)


def convert(name, input_path, keep, bucket_seed, shuffle_seed):
    status('scatter-start', f'{name}; fixed keep{keep}/1000; workers4; original pipeline')
    run([str(old / 'converter'), 'scatter', '--input', str(input_path),
         '--conflicts', str(old / 'conflicts.bin'), '--probe-keys', str(old / 'calibration-probe.keys'),
         '--output', str(s / ('scatter-' + name)), '--buckets', '256', '--workers', '4',
         '--seed', str(bucket_seed), '--keep-per-mille', str(keep)], 'scatter-' + name)
    run(['python3', str(old / 'fit-source-scale.py'), '--reference', str(old / 't80-statistics.json'),
         '--source', name + '=' + str(s / ('scatter-' + name) / 'scatter.json'),
         '--out', str(s / (name + '-scale.json'))], 'scale-' + name)
    scale = json.loads((s / (name + '-scale.json')).read_text())['sources'][name]['raw_scale']
    status('gather-start', f'{name}; raw scale{scale}; original map; consume own new buckets only')
    run([str(old / 'converter'), 'gather', '--scatter', str(s / ('scatter-' + name)) + '@' + str(scale),
         '--map', str(old / 'score-map.json'), '--output', str(s / (name + '.bf')),
         '--manifest', str(s / (name + '-manifest.json')), '--seed', str(shuffle_seed),
         '--consume-buckets', 'true'], 'gather-' + name)
    # Gather already computes the complete output digest; reuse it rather than rereading the corpus.
    pin = json.loads((s / (name + '-manifest.json')).read_text())['train']
    path = s / (name + '.bf')
    assert path.stat().st_size == pin['bytes']
    return dict(bytes=pin['bytes'], sha256=pin['sha256'], mtime_ns=path.stat().st_mtime_ns)


def main():
    assert hashlib.sha256((old / 'converter').read_bytes()).hexdigest() == '90200f4ed0e17c718d939ffdd3dd47cfd740d35c317029ffdd2eb259ddf32cc2'
    assert not (s / 't74full.bf').exists() and not (s / 't75.bf').exists()
    url = 'https://huggingface.co/datasets/official-stockfish/master-binpacks/resolve/1e095a758c630bc58d0b6dac4da44fcd38ac89c2/farseerT75.binpack'
    download_log = (s / 'download-t75.log').open('w')
    download = subprocess.Popen(['curl', '--fail', '--location', '--http1.1', '--retry', '3',
                                 '--retry-delay', '20', '--speed-limit', '100000', '--speed-time', '300',
                                 '--max-time', '14400', '--output', str(s / 'farseerT75.binpack'), url],
                                stdout=download_log, stderr=subprocess.STDOUT, start_new_session=True)
    (s / 'download.pid').write_text(str(download.pid) + '\n')
    try:
        files = {'t74full.bf': convert('t74full', old / 'farseerT74.binpack', 1000, 2026100903, 2026100912)}
        assert download.wait(timeout=14400) == 0
        assert (s / 'farseerT75.binpack').stat().st_size == 46983203020
        # The reused decoder verifies the pinned compressed input hash at its use.
        files['t75.bf'] = convert('t75', s / 'farseerT75.binpack', 100, 2026101101, 2026101111)
    finally:
        if download.poll() is None:
            os.killpg(download.pid, signal.SIGTERM)
            download.wait(timeout=20)
        download_log.close()
    (s / 'files.json').write_text(json.dumps(files, indent=2) + '\n')
    token = (s / 'token').read_text().strip()
    for name, pin in files.items():
        descriptor = dict(bytes=pin['bytes'], sha256=pin['sha256'],
                          url='http://192.168.4.65:8768/' + token + '/' + name)
        (s / (name[:-3] + '.archive.json')).write_text(json.dumps(descriptor, indent=2) + '\n')
    status('corpus-done', 'fullT74 + fixed10%T75 mapped/pinned; new union coverage next; no games/adoption')


if __name__ == '__main__':
    main()
