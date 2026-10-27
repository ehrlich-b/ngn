"""Regression cases for the shared-host busy estimate."""
import unittest

from resources import external_busy_cpu_estimate


class ExternalBusyFiniteTests(unittest.TestCase):
    def test_invalid_operands_remain_unknown(self):
        invalid = (None, float('nan'), float('inf'), float('-inf'), -1,
                   'nan', 'inf', '-inf', '-0.1', 10 ** 400, '1e400',
                   '', 'not a number', [], {})
        for value in invalid:
            for host, owned in ((value, 1), (1, value)):
                with self.subTest(host=host, owned=owned):
                    self.assertIsNone(external_busy_cpu_estimate(host, owned))

    def test_valid_operands_are_bounded(self):
        cases = ((8.25, 3.0, 5.25), ('8.25', '3', 5.25),
                 (2, 4, 0.0), (0, 0, 0.0), ('0', 0, 0.0),
                 (1.5, 0.25, 1.25))
        for host, owned, expected in cases:
            with self.subTest(host=host, owned=owned):
                self.assertEqual(external_busy_cpu_estimate(host, owned), expected)


if __name__ == '__main__':
    unittest.main()
