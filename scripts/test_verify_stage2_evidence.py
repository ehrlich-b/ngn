#!/usr/bin/env python3
"""Exercise the complete evidence verifier with signed Elo results."""
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


VERIFIER = Path(__file__).with_name('verify_stage2_evidence.py').resolve()


class SignedEloTests(unittest.TestCase):
    def test_all_result_consumers_accept_signed_estimates_and_bounds(self):
        for estimate, low, high in [('-20.0', '-30', '-10'),
                                    ('+10.0', '-10', '+30'),
                                    ('-10.0', '-30', '+10')]:
            with self.subTest(estimate=estimate, high=high), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                repo, evidence = root / 'repo', root / 'evidence'
                repo.mkdir(); evidence.mkdir()
                (evidence / 'net-vs-hce.log').write_text(
                    f'mode: nodes 1000/move\nGames: 1000 W-D-L: 300-400-300\n'
                    f'Penta Elo: {estimate} 95% CI [{low}, {high}]\n')
                for name in ('loss.csv', 'stage1-loss.csv'):
                    (evidence / name).write_text('epoch,train,validation\n1,0.1,0.1\n')
                (evidence / 'stage3.md').write_text('Stage 3: data, label, size, learning rate.')
                (evidence / 'training.log').write_text('positions train/valid: 15000000 300000\npositions_seen=15000000\n')
                (evidence / 'loss.png').write_bytes(b'fixture')
                (root / 'SHOWCASE.json').write_text(json.dumps({
                    'hero': 'evidence/loss.png', 'metrics': [
                        {'name': 'Elo of ngn-v1 net vs HCE', 'value': float(estimate), 'direction': 'higher'},
                        {'name': 'Training positions', 'value': 15000000, 'direction': 'higher'},
                        {'name': 'HCE Elo vs Counter 5.5', 'value': -63, 'direction': 'higher'},
                    ]}))
                for item in ('A4', 'A5', 'A6'):
                    result = subprocess.run([sys.executable, str(VERIFIER), item], cwd=repo,
                                            capture_output=True, text=True)
                    self.assertEqual(result.returncode, 0, f'{item}: {result.stderr}')


if __name__ == '__main__':
    unittest.main()
