import unittest

from scripts.campaign_queue import paired, summarize


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



def fixture(penta, flags=0, kind='net-transfer', cap=400):
    outcomes = ((0, 0), (0, .5), (.5, .5), (1, .5), (1, 1))
    wdl, pgn = [0, 0, 0], []
    pair = 0
    for count, scores in zip(penta, outcomes):
        for _ in range(count):
            pair += 1
            for color, score in enumerate(scores, 1):
                wdl[0 if score == 1 else 1 if score == .5 else 2] += 1
                result = '1-0' if score == 1 else '0-1' if score == 0 else '1/2-1/2'
                pgn.append(f'[Event "fixture"]\n[Round "{pair}.{color}"]\n'
                           f'[White "candidate"]\n[Black "base"]\n'
                           f'[Result "{result}"]\n[Termination "draw-rule"]\n\n{result}\n')
    elo, low, high = paired(penta)
    log = (f'Games: {pair * 2} W-D-L: {wdl[0]}-{wdl[1]}-{wdl[2]}\n'
           f'Pentanomial [LL {penta[0]} LD {penta[1]} {{LW,DD}} {penta[2]} WD {penta[3]} WW {penta[4]}]\n'
           f'Penta Elo: {elo:.1f} 95% CI [{low:.0f}, {high:.0f}] pLLR 0.00\n'
           f'Flag-outs (lost on time): new {flags}, base 0\n'
           'Verdict: INCONCLUSIVE (ran out of games)\n')
    return summarize(log, '\n'.join(pgn), {'class': kind, 'max_games': cap}, 0)


class TransferRuleTests(unittest.TestCase):
    def test_positive_point_with_overlapping_interval_passes_transfer_only(self):
        penta = [1, 25, 140, 29, 5]
        result = fixture(penta)
        self.assertEqual(result['verdict'], 'TRANSFER_PASS')
        self.assertGreater(result['exact_elo_ci95'][0], 0)
        self.assertLess(result['exact_elo_ci95'][1], 0)
        self.assertEqual(fixture(penta, kind='net')['verdict'], 'SHELVED')

    def test_zero_and_negative_points_fail_transfer(self):
        for penta in ([0, 20, 160, 20, 0], [5, 29, 140, 25, 1]):
            self.assertEqual(fixture(penta)['verdict'], 'TRANSFER_FAIL')

    def test_faults_and_incomplete_samples_are_unscored(self):
        self.assertEqual(fixture([1, 25, 140, 29, 5], flags=1)['verdict'], 'UNSCORED')
        self.assertEqual(fixture([0, 2, 14, 4, 0])['verdict'], 'UNSCORED')


if __name__ == '__main__':
    unittest.main()
