"""Exercise the observed manual checkpoint and converter-directory failure on forge."""
import importlib.util
import json
from pathlib import Path
import sys
import tempfile

spec = importlib.util.spec_from_file_location('recovery', 'recovery-session59.py')
recovery = importlib.util.module_from_spec(spec)
spec.loader.exec_module(recovery)

class Server:
    def __init__(self, *_):
        pass
    def serve_forever(self):
        pass

recovery.ThreadingHTTPServer = Server
sys.argv = ['recovery-session59.py', '--resume-after-raw']
with tempfile.TemporaryDirectory(prefix='ngn-s59-check-') as directory:
    recovery.S = Path(directory)
    s = recovery.S
    (s / 'controller.exit').write_text('1\n')
    (s / 't75-rig-copy.json').write_text(json.dumps(recovery.RAW_PIN))
    (s / 'scatter-t74full').mkdir()
    (s / 'prepare.py').write_text('''from pathlib import Path
old = Path('/original-frozen-input')
def convert(name, input_path, keep, bucket_seed, shuffle_seed):
    assert (name, str(input_path), keep, bucket_seed, shuffle_seed) == (
        't74full', '/original-frozen-input/farseerT74.binpack', 1000, 2026100903, 2026100912)
    target = Path(__file__).parent / 'scatter-t74full'
    target.mkdir()  # Same create-new requirement as the frozen converter.
    raise RuntimeError('fixture-converter-entered')
''')
    try:
        recovery.main()
    except RuntimeError as error:
        assert str(error) == 'fixture-converter-entered'
    else:
        raise AssertionError('converter was not reached')
    assert recovery.COPIED['raw'].is_set()
    assert not recovery.READY['raw'].is_set(), 'completed raw bytes must not be served again'
    assert 'raw-storage-reused' in (s / 'status.jsonl').read_text()
    print('PASS: completed raw copy reused; absent directory delivered to frozen converter; fixed arguments intact')
    (s / 't75-rig-copy.json').write_text(json.dumps(dict(recovery.RAW_PIN, sha256='bad')))
    try:
        recovery.main()
    except AssertionError:
        pass
    else:
        raise AssertionError('mismatched raw receipt was accepted')
    print('PASS: mismatched raw receipt cannot resume conversion')
