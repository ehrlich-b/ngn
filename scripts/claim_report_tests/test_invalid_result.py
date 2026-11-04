import json
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import claim_report


class InvalidResultTest(unittest.TestCase):
    def test_invalid_results_and_colors_stop_before_output(self):
        game = {'Schema': 'NGN-CLAIM-4', 'ID': 'game', 'Anchor': 'Berserk 12',
                'Error': '', 'Result': '1-0', 'CandidateColor': 'white'}
        for field, value in [('Result', 'invalid-result'), ('Result', ''),
                             ('CandidateColor', 'invalid')]:
            with self.subTest(field=field, value=value):
                altered = dict(game, **{field: value})
                def read_text(path):
                    return json.dumps({'Schema': 'NGN-CLAIM-4'} if path.name == 'reference.json' else altered)
                with patch.object(sys, 'argv', ['claim_report.py', '--reference', 'reference.json', '--output', 'out.json']), \
                     patch.object(Path, 'glob', return_value=[Path('game.json')]), \
                     patch.object(Path, 'read_text', read_text), \
                     patch.object(Path, 'write_text') as write:
                    with self.assertRaisesRegex(ValueError, 'invalid result or candidate color'):
                        claim_report.main()
                    write.assert_not_called()


if __name__ == '__main__':
    unittest.main()
