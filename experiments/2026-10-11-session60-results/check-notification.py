"""Check the actual reader's waiting connection and descriptor delivery on forge."""
import concurrent.futures
import importlib.util
import json
from pathlib import Path
import socket
import sys
import tempfile
import threading
from urllib.request import urlopen

spec = importlib.util.spec_from_file_location('reader', sys.argv[1])
reader = importlib.util.module_from_spec(spec)
spec.loader.exec_module(reader)
with tempfile.TemporaryDirectory(prefix='ngn-s60-notification-') as temporary:
    root = Path(temporary)
    server = reader.make_server(root, 'fixture', ('127.0.0.1', 0))
    handler = server.RequestHandlerClass
    original = handler.do_GET
    accepted = threading.Event()
    options = []

    def receive(self):
        options.extend([
            self.connection.getsockopt(socket.SOL_SOCKET, socket.SO_KEEPALIVE),
            self.connection.getsockopt(socket.IPPROTO_TCP, socket.TCP_KEEPIDLE),
            self.connection.getsockopt(socket.IPPROTO_TCP, socket.TCP_KEEPINTVL),
            self.connection.getsockopt(socket.IPPROTO_TCP, socket.TCP_KEEPCNT),
        ])
        accepted.set()
        original(self)

    handler.do_GET = receive
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()

    def fetch():
        with urlopen(f'http://127.0.0.1:{server.server_port}/fixture/descriptor.json', timeout=10) as response:
            return json.loads(response.read())

    with concurrent.futures.ThreadPoolExecutor(1) as pool:
        response = pool.submit(fetch)
        assert accepted.wait(5)
        assert options == [1, 60, 30, 6], options
        assert not response.done(), 'descriptor returned before publication'
        expected = {'fixture': 'same bytes; native notification'}
        (root / 't74.archive.json.tmp').write_text(json.dumps(expected))
        (root / 't74.archive.json.tmp').replace(root / 't74.archive.json')
        assert response.result(timeout=5) == expected
    server.shutdown()
    server.server_close()
    thread.join(timeout=2)
    assert not thread.is_alive()
print('PASS: accepted TCP keepalive 60/30/6; native wait; exact atomic descriptor delivery')
