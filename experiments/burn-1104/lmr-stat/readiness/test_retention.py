"""Check replay helpers against the files that survive evidence retention."""
import json
from pathlib import Path
import shlex
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
            helper = ROOT / name
            require_retained(helper, keep)
            self.assertEqual(helper.read_bytes(), (BACKUPS / name).read_bytes())
            # Simulate the reported omission without changing or deleting any file.
            with self.assertRaisesRegex(AssertionError, 'unretained command dependency'):
                require_retained(helper, [p for p in keep if p != Path(name)])
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


if __name__ == '__main__':
    unittest.main()
