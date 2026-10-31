"""Rig-only exact identity of incremental SEE x-ray discovery."""
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session117-speed/s6'
spec = importlib.util.spec_from_file_location('identity', R/'session100-speed/identity.py')
old = importlib.util.module_from_spec(spec)
spec.loader.exec_module(old)


def main():
    binaries = dict(base=R/'session117/accepted-ngn', candidate=D/'s6-ngn')
    rows = []
    for hash_mib in (64, 256):
        engines = [old.UCI(str(path)[:-4], hash_mib) for path in binaries.values()]
        try:
            for frame in old.FRAMES:
                cells = [engine.search(frame) for engine in engines]
                assert cells[0] == cells[1], (hash_mib, frame['id'], cells)
                rows.append(dict(hash_mib=hash_mib, id=frame['id'], identity=cells[0]))
        finally:
            for engine in engines:
                engine.close()
    for name, net, depth in [('ngnn3', old.NET, 14),
            ('ngnn2', R/'session100-speed/baseline/engine/nets/ngn-h512-c6c12796.nnue', 14),
            ('hce', '', 11)]:
        cells = []
        for kind, binary in binaries.items():
            probe = binary.with_name(binary.name[:-4]+'-probe')
            raw = subprocess.check_output(['taskset', '-c', '15', 'nice', '-n', '19', str(probe),
                '-net', str(net), '-mode', 'identity', '-depth', str(depth)],
                env=dict(os.environ, GOMAXPROCS='1'), text=True, timeout=900)
            (D/f'identity-{name}-{kind}.jsonl').write_text(raw)
            data = [json.loads(line) for line in raw.splitlines()]
            assert len(data) == 12 and all(row['nodes'] > 0 for row in data)
            cells.append([{k:v for k,v in row.items() if k not in ('seconds','nps')} for row in data])
        assert cells[0] == cells[1], name
    sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
    result = dict(verdict='PASS', candidate='s6', immediate_base=(R/'session117/accepted-source.txt').read_text().strip()+'/81215229/compiled warm2000',
        uci_cases=24, probe_cases=36, identity_rows=rows, tests_pass=True,
        binary_sha256={name:sha(path) for name,path in binaries.items()},
        patch_sha256=sha(D/'candidate.patch'), net_sha256=sha(old.NET),
        goamd64='v3', check_cpus=[13,15], utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        automatic_adoption=False, next='Timing unarmed behind S5; retarget if accepted source changes')
    (D/'identity.json').write_text(json.dumps(result, indent=2)+'\n')


if __name__ == '__main__':
    main()
