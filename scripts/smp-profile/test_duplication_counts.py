"""Synthetic checks of sampling overlap and invalid count handling."""
import copy
import unittest

from run import duplication


def old_duplication(workers):
    """The previous calculation, retained to demonstrate its accepted inputs."""
    union = set()
    memberships = visits = 0
    for worker in workers:
        samples = worker.get("samples", {})
        union.update(samples)
        memberships += len(samples)
        visits += sum(samples.values())
    return {"sampled_visits": visits, "worker_unique_sum": memberships,
            "union_unique": len(union), "cross_worker_excess": memberships - len(union),
            "cross_worker_excess_fraction": (memberships - len(union)) / memberships if memberships else 0,
            "within_worker_revisits": visits - memberships}


class DuplicationCountsTest(unittest.TestCase):
    def test_exact_metrics(self):
        cases = [
            ([], (0, 0, 0, 0, 0, 0)),
            ([{}, {"samples": {}}], (0, 0, 0, 0, 0, 0)),
            ([{"samples": {"zero": 0}}], (0, 1, 1, 0, 0, -1)),
            ([{"samples": {"a": 1}}], (1, 1, 1, 0, 0, 0)),
            ([{"samples": {"a": 2}}, {"samples": {"b": 1}}], (3, 2, 2, 0, 0, 1)),
            ([{"samples": {"a": 2, "b": 1}}, {"samples": {"b": 3, "c": 1}},
              {"samples": {"b": 2}}], (9, 5, 3, 2, 2 / 5, 4)),
        ]
        keys = ("sampled_visits", "worker_unique_sum", "union_unique", "cross_worker_excess",
                "cross_worker_excess_fraction", "within_worker_revisits")
        for workers, values in cases:
            with self.subTest(workers=workers):
                before = copy.deepcopy(workers)
                self.assertEqual(duplication(workers), dict(zip(keys, values)))
                self.assertEqual(workers, before)

    def test_invalid_counts_and_old_behavior(self):
        for invalid in (-1, 1.5, float('nan'), float('inf'), float('-inf'), True, False, '2'):
            for index in (0, 1):
                with self.subTest(invalid=invalid, worker=index):
                    workers = [{"samples": {"good": 2}}, {"samples": {"also_good": 1}}]
                    workers[index]["samples"]["bad"] = invalid
                    before = copy.deepcopy(workers)
                    with self.assertRaisesRegex(ValueError, f"worker {index}.*bad.*nonnegative integer"):
                        duplication(workers)
                    self.assertEqual(workers, before)
                    if isinstance(invalid, str):
                        with self.assertRaises(TypeError):
                            old_duplication(workers)
                    else:
                        self.assertIsInstance(old_duplication(workers), dict)


if __name__ == '__main__':
    unittest.main()
