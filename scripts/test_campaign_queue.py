import unittest

from campaign_queue import summarize


LOG = '''Games: 4   W-D-L: 1-2-1   score: 50.0%
Pentanomial [LL 0  LD 0  {LW,DD} 2  WD 0  WW 0] over 2 pairs
Penta Elo: +0.0   95% CI [-15, +15]   pLLR -0.05
Flag-outs (lost on time): new 0, base 0
Verdict: INCONCLUSIVE
'''


def game(pair, color, result):
    return f'''[Event "test"]
[Round "{pair}"]
[White "{color}"]
[Result "{result}"]
[Termination "draw-rule"]

*

'''


PGN = game('1.1', 'candidate', '1-0') + game('1.2', 'base', '1-0') + game('2.1', 'candidate', '1/2-1/2') + game('2.2', 'base', '1/2-1/2')


class QueueResultTest(unittest.TestCase):
    def test_pgn_pair_and_color_accounting(self):
        result = summarize(LOG, PGN, {'class': 'aa', 'max_games': 4}, 0)
        self.assertEqual(result['verdict'], 'AA_PASS')
        self.assertEqual(result['penta'], [0, 0, 2, 0, 0])

    def test_capture_fault_or_flag_cannot_become_a_verdict(self):
        for log, pgn in ((LOG, PGN.rsplit('[Event', 1)[0]), (LOG.replace('new 0', 'new 1'), PGN)):
            result = summarize(log, pgn, {'class': 'correctness', 'max_games': 4}, 0)
            self.assertEqual(result['verdict'], 'UNSCORED')

    def test_native_first_crossing_survives_drain(self):
        log = LOG.replace('INCONCLUSIVE', 'H1 ACCEPTED: new is stronger (>= 5 ELO)')
        result = summarize(log, PGN, {'class': 'search', 'max_games': 8000}, 0)
        self.assertEqual(result['verdict'], 'KEEP')
        self.assertLess(result['exact_pllr'], 0)


if __name__ == '__main__':
    unittest.main()
