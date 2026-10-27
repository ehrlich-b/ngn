"""Regression cases for the shared-host busy CPU estimate."""
import unittest

from resources import external_busy_cpu_estimate


class BusyCounterTypesTest(unittest.TestCase):
    def test_boolean_operands_are_unknown(self):
        # Each case isolates one boolean operand and had a numeric baseline.
        for host, owned, baseline in ((True, 0.25, 0.75), (False, 0.25, 0.0),
                                      (2.0, True, 1.0), (2.0, False, 2.0)):
            with self.subTest(host=host, owned=owned, baseline=baseline):
                original = (host, owned)
                self.assertIsNone(external_busy_cpu_estimate(host, owned))
                self.assertEqual((host, owned), original)

    def test_numeric_controls_and_invalid_inputs(self):
        cases = ((8, 3, 5.0), (8.25, 3.0, 5.25), ("8.25", "3.0", 5.25),
                 (0, 0, 0.0), (2, 4, 0.0), (-1, 0, None), (1, -1, None),
                 (float('nan'), 1, None), (1, float('nan'), None),
                 (float('inf'), 1, None), (1, float('inf'), None),
                 (float('-inf'), 1, None), (1, float('-inf'), None),
                 ("malformed", 1, None), (1, "malformed", None),
                 (10**400, 1, None), (1, 10**400, None),
                 ("1e400", 1, None), (1, "1e400", None))
        for host, owned, expected in cases:
            with self.subTest(host=host, owned=owned):
                original = (host, owned)
                self.assertEqual(external_busy_cpu_estimate(host, owned), expected)
                self.assertEqual((host, owned), original)


if __name__ == '__main__':
    unittest.main()
