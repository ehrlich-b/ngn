"""Independent malformed file-map controls for offline resource summaries."""
import copy
import unittest

import resources


class ResourceFilesShapeTests(unittest.TestCase):
    def test_nonmapping_files_are_unknown_and_preserved(self):
        for files in (None, False, True, 0, 17, 1.5, '', 'cpu.max', [], ['cpu.max'], (), ('cpu.max',)):
            with self.subTest(files=files):
                evidence = {'files': files, 'affinity': [0, 1], 'gomaxprocs': '2'}
                before = copy.deepcopy(evidence)
                result = resources.summarize_resources(evidence, 4)
                self.assertEqual(evidence, before)
                self.assertEqual(result['raw_evidence']['files'], files)
                self.assertEqual(result['cgroup_cpu_quota_state'], 'unknown')
                self.assertFalse(result['cgroup_cpu_quota_evidence_complete'])
                self.assertIsNone(result['cpuset_cpu_count'])
                self.assertIn('cgroup_cpu_quota_cores', result['unknown_capacity_fields'])
                self.assertIn('cpuset_cpu_count', result['unknown_capacity_fields'])
                self.assertEqual((result['affinity_cpu_count'], result['subprocess_gomaxprocs']), (2, 2))

    def test_nonstring_keys_and_mixed_maps(self):
        malformed = {None: 'max 100000', 7: '0-3', (1, 2): 'anything', False: 'value'}
        valid = {'/cg/cpu.max': '150000 100000', '/cg/cpuset.cpus': '0-3'}
        for files in (malformed, {**malformed, **valid}):
            with self.subTest(files=files):
                evidence = {'files': files, 'affinity': [0, 1], 'gomaxprocs': 2}
                before = copy.deepcopy(evidence)
                result = resources.summarize_resources(evidence, 4)
                self.assertEqual(evidence, before)
                self.assertEqual(result['raw_evidence']['files'], files)
                self.assertEqual(result['cgroup_cpu_quota_state'], 'limited' if files != malformed else 'unknown')
                self.assertEqual(result['cgroup_cpu_quota_cores'], 1.5 if files != malformed else None)
                self.assertEqual(result['cpuset_cpu_count'], 4 if files != malformed else None)
                if files == malformed:
                    self.assertIn('cgroup_cpu_quota_cores', result['unknown_capacity_fields'])
                    self.assertIn('cpuset_cpu_count', result['unknown_capacity_fields'])

    def test_missing_required_quota_with_mixed_keys(self):
        evidence = {'cgroup': '0::/child',
                    'mountinfo': '1 0 0:1 /parent /cg rw - cgroup2 cgroup rw',
                    'files': {None: 'max 100000', '/cg/cpuset.cpus': '0-1'}}
        # A subtree mount has a required quota even if no usable quota file exists.
        evidence['cgroup'] = '0::/parent/child'
        result = resources.summarize_resources(evidence, 4)
        self.assertEqual(result['cgroup_cpu_quota_state'], 'unknown')
        self.assertEqual(result['cgroup_cpu_quota_unknown_levels'], ['/cg/child/cpu.max', '/cg/cpu.max'])
        self.assertEqual(result['cpuset_cpu_count'], 2)

    def test_valid_v1_v2_hybrid_maps_unchanged_by_bad_keys(self):
        fixtures = [
            {'cgroup': '2:cpu:/job', 'mountinfo': '1 0 0:1 / /cg rw - cgroup cgroup rw,cpu',
             'files': {'/cg/cpu.cfs_quota_us': '-1', '/cg/cpu.cfs_period_us': '100000',
                       '/cg/job/cpu.cfs_quota_us': '150000', '/cg/job/cpu.cfs_period_us': '100000',
                       '/cg/job/cpuset.cpus': '0-3'}},
            {'cgroup': '0::/job', 'mountinfo': '1 0 0:1 / /cg rw - cgroup2 cgroup rw',
             'files': {'/cg/job/cpu.max': '150000 100000', '/cg/job/cpuset.cpus.effective': '0-3'}},
            {'cgroup': '2:cpu:/job\n0::/job',
             'mountinfo': '1 0 0:1 / /cpu rw - cgroup cgroup rw,cpu\n2 0 0:2 / /unified rw - cgroup2 cgroup rw',
             'files': {'/cpu/cpu.cfs_quota_us': '-1', '/cpu/cpu.cfs_period_us': '100000',
                       '/cpu/job/cpu.cfs_quota_us': '150000', '/cpu/job/cpu.cfs_period_us': '100000',
                       '/unified/job/cpu.max': 'max 100000', '/unified/job/cpuset.cpus': '0-3'}}]
        for evidence in fixtures:
            with self.subTest(cgroup=evidence['cgroup']):
                evidence.update(affinity=[0, 1], gomaxprocs=2)
                before = copy.deepcopy(evidence)
                expected = resources.summarize_resources(evidence, 4)
                mixed = copy.deepcopy(evidence)
                mixed['files'][None] = {'unavailable': 'bad key'}
                mixed_before = copy.deepcopy(mixed)
                actual = resources.summarize_resources(mixed, 4)
                self.assertEqual(evidence, before)
                self.assertEqual(mixed, mixed_before)
                self.assertEqual(actual['raw_evidence']['files'], mixed['files'])
                actual['raw_evidence'] = expected['raw_evidence']
                self.assertEqual(actual, expected)
                self.assertEqual((actual['cgroup_cpu_quota_cores'], actual['cpuset_cpu_count']), (1.5, 4))


if __name__ == '__main__':
    unittest.main()
