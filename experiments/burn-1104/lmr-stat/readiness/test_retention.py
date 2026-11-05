"""Check replay helpers against the files that survive evidence retention."""
import json
from pathlib import Path
import shlex
import shutil
import subprocess
import tempfile
import unittest


REPO = Path.cwd().resolve()
ROOT = REPO.parent
BACKUPS = Path(__file__).resolve().parent
HELPERS = ('proof.py', 'acceptance.py', 'identity.sh', 'smoke.sh')


def require_retained(path, keep):
    relative = path.resolve().relative_to(ROOT)
    if not any(relative == entry or entry in relative.parents for entry in keep):
        raise AssertionError(f'unretained command dependency: {relative}')
    if not path.is_file():
        raise AssertionError(f'missing command dependency: {relative}')


class RetentionRegression(unittest.TestCase):
    def test_plan_dependencies_survive_retention(self):
        keep = [Path(line) for line in (ROOT / 'KEEP').read_text().splitlines() if line]
        for entry in keep:
            self.assertFalse(entry.is_absolute() or '..' in entry.parts)
        for name in HELPERS:
            helper = BACKUPS / name
            self.assertTrue(helper.is_file(), helper)
            subprocess.run(['git', 'ls-files', '--error-unmatch', str(helper.relative_to(REPO))],
                           check=True, stdout=subprocess.DEVNULL)
        plan = json.loads((ROOT / 'readiness-plan.json').read_text())
        require_retained(ROOT / 'readiness-plan.json', keep)
        for row in plan['acceptance'] + plan['gates']:
            for word in shlex.split(row['command']):
                if word.endswith(('.py', '.sh')):
                    path = (REPO / word).resolve()
                    if REPO in path.parents:
                        # Repository files are pinned in the retained Git bundle.
                        self.assertTrue(path.is_file(), path)
                        self.assertTrue((ROOT / 'evidence/candidate.bundle').is_file())
                    else:
                        require_retained(path, keep)

    def test_registered_helpers_run_without_lane_root_copies(self):
        plan = json.loads((ROOT / 'readiness-plan.json').read_text())
        for row in plan['acceptance'] + plan['gates']:
            for word in shlex.split(row['command']):
                if word.endswith(('.py', '.sh')):
                    self.assertTrue((REPO / word).resolve().is_relative_to(REPO), word)
        # Reproduce retention with only the committed helpers and required evidence.
        with tempfile.TemporaryDirectory(dir=ROOT / 'tmp') as directory:
            lane = Path(directory)
            repo = lane / 'repo'
            support = repo / BACKUPS.relative_to(REPO)
            support.mkdir(parents=True)
            for name in HELPERS:
                shutil.copyfile(BACKUPS / name, support / name)
                self.assertFalse((lane / name).exists())
            (lane / 'evidence').mkdir()
            shutil.copyfile(ROOT / 'evidence/decisions.json', lane / 'evidence/decisions.json')
            command = next(row['command'] for row in plan['acceptance'] if row['id'] == 'A9')
            result = subprocess.run(shlex.split(command), cwd=repo, text=True,
                                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            self.assertEqual(result.returncode, 0, result.stdout)
            self.assertIn('Headless checks', result.stdout)
            self.assertFalse((support / 'tmp').exists())


if __name__ == '__main__':
    unittest.main()
