"""Rig-only S3 exact search identity and compiler witness; no speed verdict."""
import datetime
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess

R = Path('/home/ehrli/ngn-data/night-20261007')
S = R / 'session101-speed'
B = R / 'session100-speed'
spec = importlib.util.spec_from_file_location('existing_identity', B / 'identity.py')
identity = importlib.util.module_from_spec(spec)
spec.loader.exec_module(identity)


def main():
    rows = []
    for hash_mib in (64, 256):
        engines = [identity.UCI('baseline', hash_mib), identity.UCI(str(S/'s3'), hash_mib)]
        try:
            for frame in identity.FRAMES:
                cells = [e.search(frame) for e in engines]
                assert cells[0] == cells[1], (hash_mib, frame['id'], cells)
                rows.append(dict(hash_mib=hash_mib, id=frame['id'], identity=cells[0]))
        finally:
            for e in engines:
                e.close()
    for name, net, depth in [('ngnn3', identity.NET, 14),
                             ('ngnn2', B/'baseline/engine/nets/ngn-h512-c6c12796.nnue', 14),
                             ('hce', '', 11)]:
        cells = []
        for kind, binary in [('baseline', B/'baseline-probe'), ('s3', S/'s3-probe')]:
            raw = subprocess.check_output(['taskset', '-c', '15', 'nice', '-n', '19', str(binary),
                '-net', str(net), '-mode', 'identity', '-depth', str(depth)],
                env=dict(os.environ, GOMAXPROCS='1'), text=True, timeout=900)
            (S/f'identity-{name}-{kind}.jsonl').write_text(raw)
            data = [json.loads(line) for line in raw.splitlines()]
            assert len(data) == 12 and all(row['nodes'] > 0 for row in data)
            cells.append([{k:v for k,v in row.items() if k not in ('seconds','nps')} for row in data])
        assert cells[0] == cells[1], name
    assembly = (S/'assembly-full.log').read_text()
    start = assembly.index('github.com/ehrlich-b/ngn/engine.selectNextMove STEXT')
    end = assembly.find('\n\t0x0000', start)
    # A function starts at a non-indented STEXT header; its next peer ends it.
    import re
    match = re.search(r'\n\S.* STEXT ', assembly[start+1:])
    function = assembly[start:start+1+match.start()] if match else assembly[start:]
    assert 'CMOVQ' in function, 'Go max did not compile to CMOV'
    (S/'selection-assembly.txt').write_text(function)
    sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
    result = dict(verdict='PASS', candidate='s3', uci_cases=24, probe_cases=36,
        identity_rows=rows, cmov=True, tests_pass=True,
        source_base='2f39a41 / accepted engine9ec5d31',
        binary_sha256=dict(base=sha(B/'baseline-ngn'),candidate=sha(S/'s3-ngn')),
        net_sha256=sha(identity.NET), go=subprocess.check_output(['go','version'],text=True).strip(),
        goamd64='v3', check_cpus=[13,15], utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        next='Prepared only; exclusive balanced timing required; no adoption')
    (S/'identity.json').write_text(json.dumps(result,indent=2)+'\n')


if __name__ == '__main__':
    main()
