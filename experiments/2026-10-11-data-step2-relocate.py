"""Manually recover interrupted fixed conversion using rig temporary storage."""
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import numpy as np
from trainer.archive import ARCHIVE_DTYPE, board_hash

S = Path('/work/lanes/ngn-data-step2-20261011')
TOKEN = (S / 'token').read_text().strip()
RAW_PIN = dict(bytes=46983203020,
               sha256='f1907be568d4a48297dd28c2060c53b07ee4170e4e9392faa525d6d697948441')
READY = {name: threading.Event() for name in ['raw', 't74']}
COPIED = {name: threading.Event() for name in ['raw', 't74']}
RESTORED = threading.Event()
PINS = {'raw': RAW_PIN.copy(), 't74': {}}


def atomic(name, value):
    p = S / name
    p.with_suffix(p.suffix + '.tmp').write_text(json.dumps(value, indent=2) + '\n')
    p.with_suffix(p.suffix + '.tmp').replace(p)


def status(event, detail):
    with (S / 'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                               stage='session58-data-recovery', event=event, detail=detail)) + '\n')


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_args):
        pass

    def do_GET(self):
        paths = {f'/{TOKEN}/{name}/{kind}': (name, kind)
                 for name in READY for kind in ['descriptor', 'data']}
        if self.path not in paths:
            self.send_error(404)
            return
        name, kind = paths[self.path]
        if not READY[name].wait(6 * 3600):
            self.send_error(504)
            return
        if kind == 'descriptor':
            raw = json.dumps(PINS[name]).encode()
            self.send_response(200)
            self.send_header('Content-Length', str(len(raw)))
            self.end_headers()
            self.wfile.write(raw)
            return
        path = S / ('farseerT75.binpack' if name == 'raw' else 't74full.bf')
        with path.open('rb') as source:
            self.send_response(200)
            self.send_header('Content-Length', str(PINS[name]['bytes']))
            self.end_headers()
            while block := source.read(1024**2):
                self.wfile.write(block)

    def do_POST(self):
        paths = {f'/{TOKEN}/{name}/copied': name for name in COPIED}
        size = int(self.headers.get('Content-Length', '0'))
        if self.path in paths:
            name = paths[self.path]
            if not READY[name].is_set() or not 0 < size < 4096:
                self.send_error(409)
                return
            receipt = json.loads(self.rfile.read(size))
            if any(receipt.get(key) != PINS[name][key] for key in ['bytes', 'sha256']):
                self.send_error(412)
                return
            atomic('t75-rig-copy.json' if name == 'raw' else 't74full-rig-copy.json', receipt)
            self.send_response(200)
            self.send_header('Content-Length', '0')
            self.end_headers()
            COPIED[name].set()
            return
        if self.path != f'/{TOKEN}/raw/restore' or size != RAW_PIN['bytes'] or not COPIED['t74'].is_set():
            self.send_error(409)
            return
        # T74 acknowledgment can arrive before its source has been removed.
        if not T74_REMOVED.wait(60):
            self.send_error(503)
            return
        path, temporary = S / 'farseerT75.binpack', S / 'farseerT75.binpack.restoring'
        if path.exists() or temporary.exists():
            self.send_error(409)
            return
        sha, left = hashlib.sha256(), size
        with temporary.open('xb') as output:
            while left:
                block = self.rfile.read(min(left, 1024**2))
                if not block:
                    raise ConnectionError('raw restore interrupted')
                output.write(block)
                sha.update(block)
                left -= len(block)
            output.flush()
            os.fsync(output.fileno())
        assert sha.hexdigest() == RAW_PIN['sha256']
        temporary.replace(path)
        fd = os.open(S, os.O_DIRECTORY)
        os.fsync(fd)
        os.close(fd)
        atomic('t75-restored.json', dict(**RAW_PIN, completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat()))
        self.send_response(200)
        self.send_header('Content-Length', '0')
        self.end_headers()
        RESTORED.set()


T74_REMOVED = threading.Event()


def main():
    assert (S / 'controller.exit').read_text().strip() == '1'
    assert not (S / 't74full.bf').exists() and not (S / 't75.bf').exists()
    assert (S / 'farseerT75.binpack').stat().st_size == RAW_PIN['bytes']
    server = ThreadingHTTPServer(('192.168.4.65', 8769), Handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    PINS['raw']['url'] = f'http://192.168.4.65:8769/{TOKEN}/raw/data'
    READY['raw'].set()
    status('raw-storage-start', 'preserve complete T75 on rig before rerunning interrupted T74 scatter')
    assert COPIED['raw'].wait(6 * 3600), 'no raw copy acknowledgment; retain source'
    (S / 'farseerT75.binpack').unlink()
    bucket_dir = S / 'scatter-t74full'
    status('discard-incomplete-buckets', f'new-lane partial files{len(list(bucket_dir.iterdir()))}; retained interruption logs')
    subprocess.run(['find', str(bucket_dir), '-mindepth', '1', '-delete'], check=True)
    spec = importlib.util.spec_from_file_location('frozen_prepare', S / 'prepare.py')
    prepare = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(prepare)
    pin = prepare.convert('t74full', prepare.old / 'farseerT74.binpack', 1000, 2026100903, 2026100912)
    path = S / 't74full.bf'
    rows = np.memmap(path, dtype=ARCHIVE_DTYPE, mode='r')
    sample = np.empty(0, dtype=np.uint64)
    for start in range(0, len(rows), 262144):
        hashes = board_hash(rows[start:start + 262144], 20261006)
        if len(sample) == 65536:
            hashes = hashes[hashes < sample[-1]]
        if len(hashes):
            sample = np.unique(np.concatenate((sample, hashes)))
            if len(sample) > 65536:
                sample = np.partition(sample, 65535)[:65536]
                sample.sort()
    np.save(S / 't74full-bottom65536.npy', sample)
    del rows
    atomic('t74full-sample.json', dict(bytes=pin['bytes'], sha256=pin['sha256'], seed=20261006,
                                      k=65536, sample_sha256=hashlib.sha256((S / 't74full-bottom65536.npy').read_bytes()).hexdigest()))
    PINS['t74'].update(bytes=pin['bytes'], sha256=pin['sha256'], url=f'http://192.168.4.65:8769/{TOKEN}/t74/data')
    READY['t74'].set()
    status('t74-copy-ready', f"completed fixed T74 bytes{pin['bytes']}; frozen sample retained")
    assert COPIED['t74'].wait(6 * 3600), 'T74 copy not confirmed; retain source'
    path.unlink()
    T74_REMOVED.set()
    status('t74-copy-confirmed', 'durable exact rig copy; removed only new forge duplicate; restore T75 next')
    assert RESTORED.wait(6 * 3600), 'raw restore incomplete; no T75 conversion'
    atomic('storage-resumed.json', dict(mode='manual recovery after interrupted scatter', full_t74_on_rig=True, t75_restored=True))
    status('conversion-resumed', 'T75 exact raw bytes restored; first fixed10% T75 conversion starts')
    files = {'t74full.bf': pin, 't75.bf': prepare.convert('t75', S / 'farseerT75.binpack', 100, 2026101101, 2026101111)}
    atomic('files.json', files)
    for name, file_pin in files.items():
        atomic(name[:-3] + '.archive.json', dict(bytes=file_pin['bytes'], sha256=file_pin['sha256'],
                                                url=f'http://192.168.4.65:8768/{TOKEN}/{name}'))
    status('corpus-done', 'unchanged fullT74/fixed10%T75; original coverage continuation next')
    server.shutdown()


if __name__ == '__main__':
    main()
