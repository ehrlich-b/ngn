"""Review newly completed compatibility and key evidence without engine runs."""
import hashlib
import importlib.util
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parent
RESULTS = ROOT / '2026-10-09-session35-results'
spec = importlib.util.spec_from_file_location('factorizer', ROOT / '2026-10-08-factorizer-trial.py')
parser = importlib.util.module_from_spec(spec)
spec.loader.exec_module(parser)
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()


def terminal(run):
    assert (run / 'controller.exit').read_text().strip() == '0'
    assert (run / 'controller.done').stat().st_size
    for mode in ['full-short', 'full-race']:
        text = (run / f'{mode}.log').read_text()
        assert 'github.com/ehrlich-b/ngn/engine' in text and 'FAIL' not in text
    assert not (run / 'vet.log').stat().st_size


run = RESULTS / 'session32-key'
terminal(run)
aa = parser.gate((run / 'aa.log').read_text(), 'aa')
gate = parser.gate((run / 'gate.log').read_text(), 'gate')
reported = json.loads((run / 'gate-summary.json').read_text())
assert gate['exact_elo_ci95'] == reported['exact_elo_ci95']
assert gate['exact_elo_ci95'][2] < 0 and reported['verdict'] == 'shelved'
assert not re.search(r'PANIC|RECOVERED|FATAL', (run / 'ngn_crashes.log').read_text())
manifest = json.loads((run / 'game-manifest.json').read_text())
for file, expected in manifest['candidate_source_sha256'].items():
    assert sha(ROOT.parent / 'output/campaign-key-after-see' / file) == expected
assert sha(run / 'candidate.patch') == manifest['files_sha256']['candidate.patch']
for name in ['input', 'frozen']:
    for when in ['before', 'after']:
        lines = (run / f'{name}-check-{when}.txt').read_text().splitlines()
        assert lines and all(line.endswith(': OK') for line in lines)
review = dict(aa=aa, gate=gate, source_matches=True, input_pins=True,
              full_short_race_vet=True, terminal=(run / 'controller.done').read_text().strip(),
              verdict='shelved', rule='unrounded upper95 > 0; zero operational faults',
              retry_or_extension=False, playing_source='ac658e2')
(RESULTS / 'key-review.json').write_text(json.dumps(review, indent=2) + '\n')
print(json.dumps(review))

run = RESULTS / 'session34-architecture'
terminal(run)
receipt = json.loads((run / 'receipt.json').read_text())
assert receipt['base_source'] == 'ac658e2' and receipt['cuda_no_skips']
topic = ROOT.parent / 'output/campaign-ngnn4-current'
for file, expected in receipt['source_sha256'].items():
    assert sha(topic / file) == expected, file
assert sha(topic / 'testdata/ngnn4/random_h16_l16_k8_b8.nnue') == 'c91adc11128dd1698e12976f967f0e81892533710acc632510cdfc6ac5e9d99c'
pins = dict(line.split('  ', 1)[::-1] for line in (run / 'inputs.sha256').read_text().splitlines())
assert sha(run / 'candidate.patch') == pins['candidate.patch']
for when in ['before', 'after']:
    lines = (run / f'input-check-{when}.txt').read_text().splitlines()
    assert lines and all(line.endswith(': OK') for line in lines)
for mode in ['ngnn3', 'ngnn2', 'hce']:
    rows = [[{k: v for k, v in json.loads(line).items() if k not in ['seconds', 'nps']}
             for line in (run / f'identity-{mode}-{kind}.jsonl').read_text().splitlines()]
            for kind in ['baseline', 'candidate']]
    assert len(rows[0]) == 12 and rows[0] == rows[1]
cuda = (run / 'cuda.log').read_text()
assert 'Ran 2 tests' in cuda and 'OK' in cuda and 'skipped' not in cuda
assert 'Ran 65 tests' in (run / 'trainer-full.log').read_text()
assert 'OK (skipped=6)' in (run / 'trainer-full.log').read_text()
assert 'VNNI512 hardware and OS state admitted' in (run / 'differential-avx2.log').read_text()
review = dict(**receipt, source_matches=True, input_pins=True, exact_identities=36,
              cpu_trainer_tests=65, expected_cpu_cuda_skips=6, actual_cuda_tests=2,
              verdict='retain isolated architecture topic', speed_repeated=False,
              training_launched=False, terminal=(run / 'controller.done').read_text().strip())
(RESULTS / 'architecture-review.json').write_text(json.dumps(review, indent=2) + '\n')
print(json.dumps({k: v for k, v in review.items() if k != 'source_sha256'}))
