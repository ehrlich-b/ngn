import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPTS = Path(__file__).resolve().parent


class ShowcaseSelectionTest(unittest.TestCase):
    def test_each_network_can_win_and_checker_rejects_wrong_metrics(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            repo = root / 'repo'
            evidence = root / 'evidence'
            repo.mkdir()
            evidence.mkdir()
            (evidence / 'validation.log').write_text('exported net holdout blend MSE=0.04123456\n')
            (evidence / 'train.time').write_text('Maximum resident set size (kbytes): 1000000\n')

            for winner, expected_mse in ((2, 0.03093784), (3, 0.03108383), (4, 0.04123456)):
                with self.subTest(winner=winner):
                    for name in ('stage2-ngn', 'stage3-ngn', 'stage3-stage2', 'stage4-ngn', 'stage4-stage3'):
                        score = 20 if name == f'stage{winner}-ngn' else -60
                        (evidence / f'match-{name}.log').write_text(
                            'nodes 1000/move | openings: 500 (x2 colors)\n'
                            f'over 500 pairs\nPenta Elo: {score}  95% CI [{score-10}, {score+10}]\n')
                    subprocess.run([sys.executable, '-B', str(SCRIPTS / 'collect_stage4.py')], cwd=repo, check=True)
                    showcase_path = root / 'SHOWCASE.json'
                    showcase = json.loads(showcase_path.read_text())
                    self.assertEqual(showcase['metrics'][0]['number'], 20)
                    self.assertTrue(showcase['metrics'][0]['note'].startswith(f'Stage {winner}'))
                    self.assertEqual(showcase['metrics'][1]['number'], expected_mse)
                    checker = [sys.executable, '-B', str(SCRIPTS / 'check_stage4.py'), 'A9']
                    subprocess.run(checker, cwd=repo, check=True, capture_output=True)
                    showcase['metrics'][1]['number'] = 0.9
                    showcase_path.write_text(json.dumps(showcase))
                    self.assertNotEqual(subprocess.run(checker, cwd=repo, capture_output=True).returncode, 0)
                    showcase['metrics'][1]['number'] = expected_mse
                    showcase['metrics'][0]['number'] = -60
                    showcase_path.write_text(json.dumps(showcase))
                    self.assertNotEqual(subprocess.run(checker, cwd=repo, capture_output=True).returncode, 0)


if __name__ == '__main__':
    unittest.main()
