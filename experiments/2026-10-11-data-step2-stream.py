"""Measure old-corpus bottom-k samples from bounded data-loading byte streams."""
import datetime
from http.server import BaseHTTPRequestHandler, HTTPServer
import json
from pathlib import Path
import time

import numpy as np
from trainer.archive import ARCHIVE_DTYPE, board_hash

s = Path('/work/lanes/ngn-data-step2-20261011')
token = (s / 'token').read_text().strip()
expected = {'t80': 75977281056, 't76': 55627628064}
K = 65536


def update(sample, hashes):
    if len(sample) == K:
        hashes = hashes[hashes < sample[-1]]
    values = np.unique(np.concatenate((sample, hashes)))
    if len(values) > K:
        values = np.partition(values, K - 1)[:K]
        values.sort()
    return values


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_args):
        pass

    def do_POST(self):
        name = self.path.removeprefix('/' + token + '/')
        size = int(self.headers.get('Content-Length', '-1'))
        if name not in expected or size != expected[name] or not self.path.startswith('/' + token + '/'):
            self.send_error(400)
            return
        if (s / (name + '-stream.json')).exists():
            self.send_error(409)
            return
        started = time.monotonic()
        remaining = size
        sample = np.empty(0, dtype=np.uint64)
        while remaining:
            block = self.rfile.read(min(262144 * 32, remaining))
            if not block or len(block) % 32:
                raise ConnectionError('Incomplete or unaligned data-loading stream')
            sample = update(sample, board_hash(np.frombuffer(block, dtype=ARCHIVE_DTYPE), 20261006))
            remaining -= len(block)
        np.save(s / (name + '-bottom65536.npy'), sample)
        receipt = dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                       source=name, bytes=size, seconds=time.monotonic() - started,
                       method='bottom65536 occupied-board hashes; ngn-board-mix64-v1/seed20261006',
                       machine='forge; streamed bytes from existing hash-pinned rig corpus',
                       source_hash_reaudit=False)
        (s / (name + '-stream.json')).write_text(json.dumps(receipt, indent=2) + '\n')
        self.send_response(200)
        self.send_header('Content-Length', '0')
        self.end_headers()


if __name__ == '__main__':
    server = HTTPServer(('192.168.4.65', 8770), Handler)
    while not all((s / (name + '-stream.json')).exists() for name in expected):
        server.handle_request()
    server.server_close()
    (s / 'stream.done').write_text(datetime.datetime.now(datetime.timezone.utc).isoformat() + '\n')
