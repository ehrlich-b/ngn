"""Temporary raw storage, exact mapped T74 loading, and raw restoration."""
import datetime
import hashlib
import http.client
import json
import os
from pathlib import Path
from urllib.request import Request, urlopen

S = Path('/home/ehrli/ngn-data/night-20261007/session57-data-step2')


def receive(endpoint, stage, target):
    with urlopen(endpoint + '/' + stage + '/descriptor', timeout=21600) as response:
        pin = json.loads(response.read())
    disk = os.statvfs('/mnt/c')
    assert disk.f_bavail * disk.f_frsize - pin['bytes'] >= 25.5 * 1024**3
    temporary = target.with_suffix(target.suffix + '.copying')
    assert not temporary.exists() and not target.exists()
    sha, size = hashlib.sha256(), 0
    with urlopen(pin['url'], timeout=21600) as source, temporary.open('xb') as output:
        while block := source.read(1024**2):
            output.write(block)
            sha.update(block)
            size += len(block)
        output.flush()
        os.fsync(output.fileno())
    assert size == pin['bytes'] and sha.hexdigest() == pin['sha256']
    temporary.replace(target)
    fd = os.open(S, os.O_DIRECTORY)
    os.fsync(fd)
    os.close(fd)
    if stage == 't74':
        files = {'t74full.bf': dict(bytes=size, sha256=pin['sha256'], mtime_ns=target.stat().st_mtime_ns)}
        (S / 'files.json.tmp').write_text(json.dumps(files, indent=2) + '\n')
        (S / 'files.json.tmp').replace(S / 'files.json')
    receipt = dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                   bytes=size, sha256=pin['sha256'], path=str(target),
                   method='streaming SHA256 during copy; fsync file and directory')
    (S / ('t74full-copy-receipt.json' if stage == 't74' else 't75-copy-receipt.json')).write_text(json.dumps(receipt, indent=2) + '\n')
    with urlopen(Request(endpoint + '/' + stage + '/copied', data=json.dumps(receipt).encode(),
                         headers={'Content-Type': 'application/json'}, method='POST'), timeout=60) as response:
        assert response.status == 200
    return pin


def main():
    token = (S / 'token').read_text().strip()
    endpoint = 'http://192.168.4.65:8769/' + token
    raw = S / 'farseerT75.binpack'
    pin = receive(endpoint, 'raw', raw)
    receive(endpoint, 't74', S / 't74full.bf')
    connection = http.client.HTTPConnection('192.168.4.65', 8769, timeout=21600)
    connection.putrequest('POST', '/' + token + '/raw/restore')
    connection.putheader('Content-Length', str(pin['bytes']))
    connection.endheaders()
    with raw.open('rb') as source:
        while block := source.read(1024**2):
            connection.send(block)
    response = connection.getresponse()
    assert response.status == 200
    response.read()
    connection.close()
    raw.unlink()  # Forge acknowledged exact durable restoration.


if __name__ == '__main__':
    main()
