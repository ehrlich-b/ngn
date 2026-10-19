"""Rig-only proof that compiled defaults reproduce the winning option vector."""
import datetime
import hashlib
import importlib.util
import json
from pathlib import Path
import re

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session115'
spec = importlib.util.spec_from_file_location('identity', R/'session100-speed/identity.py')
identity = importlib.util.module_from_spec(spec)
spec.loader.exec_module(identity)


def main():
    endpoint = json.loads((D/'endpoint.json').read_text())
    vector = endpoint['rounded']
    rows = []
    for hash_mib in (64, 256):
        engines = [identity.UCI(str(path)[:-4], hash_mib) for path in
                   (R/'session104-speed/s1-ngn', D/'accepted-ngn')]
        try:
            engines[1].send('uci')
            advertised = engines[1].until('uciok')
            defaults = {name:int(value) for line in advertised
                        for name,value in re.findall(r'^option name (\w+) type spin default (-?\d+)', line)}
            assert all(defaults[name] == value for name,value in vector.items()), defaults
            for name,value in vector.items():
                engines[0].send(f'setoption name {name} value {value}')
            engines[0].send('isready')
            engines[0].until('readyok')
            for frame in identity.FRAMES:
                cells = [engine.search(frame) for engine in engines]
                assert cells[0] == cells[1], (hash_mib, frame['id'], cells)
                rows.append(dict(hash_mib=hash_mib, id=frame['id'], identity=cells[0]))
        finally:
            for engine in engines:
                engine.close()
    binary = D/'accepted-ngn'
    result = dict(verdict='PASS', cases=len(rows), rows=rows, vector=vector,
                  binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),
                  treatment='Compiled warm final2000 defaults versus winning explicit options on accepted fe92ced',
                  finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    (D/'default-parity.json').write_text(json.dumps(result, indent=2)+'\n')


if __name__ == '__main__':
    main()
