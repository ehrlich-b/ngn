"""Synthetic captured controls; no hardware, model or search is exercised."""
import builtins
import importlib.util
import io
from pathlib import Path
import sys
import unittest
from unittest import mock

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import resources
import run as profile_run


def captured(version, values):
    files = {}
    for index, value in enumerate(values):
        folder = '/cg' + '/child' * index
        if version == 2:
            files[folder + '/cpu.max'] = value
        else:
            quota, period = value
            files[folder + '/cpu.cfs_quota_us'] = quota
            files[folder + '/cpu.cfs_period_us'] = period
    return {'gomaxprocs': 2, 'affinity': [0, 1], 'files': files}


class QuotaStateTests(unittest.TestCase):
    def check_summary(self, evidence, state, bound, complete):
        result = resources.summarize_resources(evidence, 8)
        self.assertEqual(result['cgroup_cpu_quota_state'], state)
        self.assertEqual(result['cgroup_cpu_quota_cores'], bound)
        self.assertIs(result['cgroup_cpu_quota_evidence_complete'], complete)
        self.assertEqual('cgroup_cpu_quota_cores' in result['unknown_capacity_fields'], not complete)
        self.assertEqual(result['raw_evidence']['files'], evidence['files'])
        self.assertFalse(result['core_scaling_qualified'])
        self.assertFalse(result['capacity_is_exclusive_physical_cores'])
        return result

    def test_unlimited_versus_unavailable_controls(self):
        for version, unlimited, unavailable in (
            (2, 'max 100000', {'unavailable': 'permission denied'}),
            (1, ('-1', '100000'), ({'unavailable': 'permission denied'}, '100000'))):
            with self.subTest(version=version):
                self.check_summary(captured(version, [unlimited]), 'unlimited', None, True)
                self.check_summary(captured(version, [unavailable]), 'unknown', None, False)

    def test_hierarchy_bounds(self):
        for version in (1, 2):
            def encode(q):
                return ('max' if q == '-1' else q) + ' 100000' if version == 2 else (q, '100000')
            for values, bound in ((['-1', '150000'], 1.5), (['150000', '-1'], 1.5),
                                  (['300000', '50000', '100000'], .5), (['-1', '-1'], None)):
                with self.subTest(version=version, values=values):
                    self.check_summary(captured(version, list(map(encode, values))),
                                       'unlimited' if bound is None else 'limited', bound, True)

    def test_invalid_periods_and_tokens(self):
        for period in ('0', '-1', 'bad', '', None, '1.5', '+100000'):
            for quota in ('-1', '200000'):
                self.check_summary(captured(1, [(quota, period)]), 'unknown', None, False)
            for quota in ('max', '200000'):
                self.check_summary(captured(2, [quota + ' ' + str(period)]), 'unknown', None, False)
        for token in ('0', '-2', 'wat', '1.5', '+1000', 'max extra'):
            self.check_summary(captured(2, [token + ' 100000']), 'unknown', None, False)
            self.check_summary(captured(1, [(token, '100000')]), 'unknown', None, False)

    def test_absent_and_orphan_period(self):
        self.check_summary({'files': {}}, 'unknown', None, False)
        self.check_summary({'files': {'/cg/cpu.cfs_period_us': '100000'}}, 'unknown', None, False)
        self.check_summary({'files': {'/cg/cpu.cfs_quota_us': '-1'}}, 'unknown', None, False)

    def test_inaccessible_ancestor_keeps_observed_bound(self):
        for version, finite, unlimited, denied in (
            (2, '50000 100000', 'max 100000', {'unavailable': 'denied'}),
            (1, ('50000', '100000'), ('-1', '100000'), ({'unavailable': 'denied'}, '100000'))):
            for child, state, bound in ((finite, 'limited', .5), (unlimited, 'unknown', None)):
                result = self.check_summary(captured(version, [denied, child]), state, bound, False)
                self.assertEqual(len(result['cgroup_cpu_quota_unknown_levels']), 1)

    def test_missing_ancestor_from_mount_metadata(self):
        for version in (1, 2):
            evidence = captured(version, [])
            evidence['files'] = {'/cg/child/cpu.max': 'max 100000'} if version == 2 else {
                '/cg/child/cpu.cfs_quota_us': '-1', '/cg/child/cpu.cfs_period_us': '100000'}
            evidence['cgroup'] = '0::/parent/child' if version == 2 else '2:cpu:/parent/child'
            evidence['mountinfo'] = ('1 0 0:1 /parent /cg rw - cgroup2 cgroup rw' if version == 2 else
                                     '1 0 0:1 /parent /cg rw - cgroup cgroup rw,cpu')
            result = self.check_summary(evidence, 'unknown', None, False)
            self.assertEqual(len(result['cgroup_cpu_quota_unknown_levels']), 1)

    def test_native_v2_root_has_no_quota_file(self):
        evidence = captured(2, [])
        evidence.update(cgroup='0::/user.slice/job.scope\n',
                        mountinfo='36 25 0:32 / /sys/fs/cgroup rw - cgroup2 cgroup rw\n')
        root = '/sys/fs/cgroup/cpu.max'
        parent = '/sys/fs/cgroup/user.slice/cpu.max'
        child = '/sys/fs/cgroup/user.slice/job.scope/cpu.max'
        for root_value in (None, {'unavailable': 'FileNotFoundError: [Errno 2] No such file or directory'}):
            for child_value, state, bound in (('max 100000', 'unlimited', None),
                                               ('50000 100000', 'limited', .5)):
                with self.subTest(root=root_value, child=child_value):
                    evidence['files'] = {parent: 'max 100000', child: child_value}
                    if root_value is not None:
                        evidence['files'][root] = root_value
                    result = self.check_summary(evidence, state, bound, True)
                    self.assertEqual(result['cgroup_cpu_quota_unknown_levels'], [])
                    if root_value is not None:
                        self.assertIn(root, result['cgroup_quota_evidence'])
                    self.check_wiring(evidence, result)
        # An unreadable root cannot be treated as a confirmed missing file.
        evidence['files'] = {root: {'unavailable': 'PermissionError: denied'},
                             parent: 'max 100000', child: 'max 100000'}
        result = self.check_summary(evidence, 'unknown', None, False)
        self.assertEqual(result['cgroup_cpu_quota_unknown_levels'], [root])
        self.check_wiring(evidence, result)
        # Missing or inaccessible non-root ancestors remain incomplete.
        for parent_value in (None, {'unavailable': 'FileNotFoundError: missing'},
                             {'unavailable': 'PermissionError: denied'}):
            for child_value, state, bound in (('max 100000', 'unknown', None),
                                               ('50000 100000', 'limited', .5)):
                evidence['files'] = {child: child_value}
                if parent_value is not None:
                    evidence['files'][parent] = parent_value
                result = self.check_summary(evidence, state, bound, False)
                self.assertEqual(result['cgroup_cpu_quota_unknown_levels'], [parent])
        # A mount of a non-root cgroup still has a relevant quota file.
        evidence['mountinfo'] = '36 25 0:32 /user.slice /sys/fs/cgroup rw - cgroup2 cgroup rw\n'
        evidence['files'] = {'/sys/fs/cgroup/job.scope/cpu.max': 'max 100000',
                             root: {'unavailable': 'FileNotFoundError: missing'}}
        result = self.check_summary(evidence, 'unknown', None, False)
        self.assertEqual(result['cgroup_cpu_quota_unknown_levels'], [root])

    def test_hybrid_cpu_controller_uses_v1_only(self):
        evidence = captured(1, [])
        evidence.update(cgroup='2:cpu,cpuacct:/user.slice/job.scope\n0::/user.slice/job.scope\n',
                        mountinfo=('31 20 0:29 / /sys/fs/cgroup/cpu rw - cgroup cgroup rw,cpu,cpuacct\n'
                                   '36 25 0:32 / /sys/fs/cgroup/unified rw - cgroup2 cgroup rw\n'))
        root = '/sys/fs/cgroup/cpu'
        child = root + '/user.slice/job.scope'
        for include_v2_files in (False, True):
            for quota, state, bound, complete in (('-1', 'unlimited', None, True),
                    ('50000', 'limited', .5, True),
                    ({'unavailable': 'PermissionError: denied'}, 'unknown', None, False)):
                with self.subTest(v2_files=include_v2_files, quota=quota):
                    evidence['files'] = {}
                    for folder in (root, root + '/user.slice', child):
                        evidence['files'][folder + '/cpu.cfs_quota_us'] = quota if folder == child else '-1'
                        evidence['files'][folder + '/cpu.cfs_period_us'] = '100000'
                    if include_v2_files:
                        for suffix in ('', '/user.slice', '/user.slice/job.scope'):
                            evidence['files']['/sys/fs/cgroup/unified' + suffix + '/cpu.max'] = {
                                'unavailable': 'FileNotFoundError: missing'}
                    result = self.check_summary(evidence, state, bound, complete)
                    self.assertEqual(result['cgroup_cpu_quota_unknown_levels'],
                                     [] if complete else [child + '/cpu.cfs_quota_us'])
                    self.check_wiring(evidence, result)
        # A v1 cpuset mount alone does not make v2 CPU evidence irrelevant.
        evidence.update(cgroup='3:cpuset:/job\n0::/job\n',
                        mountinfo=('31 20 0:29 / /sys/fs/cgroup/cpuset rw - cgroup cgroup rw,cpuset\n'
                                   '36 25 0:32 / /sys/fs/cgroup/unified rw - cgroup2 cgroup rw\n'))
        evidence['files'] = {'/sys/fs/cgroup/unified/job/cpu.max': 'max 100000'}
        self.check_summary(evidence, 'unlimited', None, True)
        evidence['files']['/sys/fs/cgroup/unified/job/cpu.max'] = {'unavailable': 'PermissionError: denied'}
        self.check_summary(evidence, 'unknown', None, False)

    def check_wiring(self, evidence, expected):
        for status in ('MEASURED_CLOSE_NO_POLICY', 'BLOCKED', 'HARNESS_COMPILED_ONLY'):
            report = profile_run.wire_report_resources({'status': status}, evidence)
            self.assertEqual(report['status'], status)
            for key in ('cgroup_cpu_quota_state', 'cgroup_cpu_quota_evidence_complete',
                        'cgroup_cpu_quota_unknown_levels', 'unknown_capacity_fields', 'raw_evidence'):
                self.assertEqual(report['resources'][key], expected[key])
        for width in (4, 8):
            row = profile_run.wire_cell_resources({'gomaxprocs': 2, 'effective_threads': width,
                'logical_worker_ratio': .25}, {'resource_evidence': evidence}, width, '/receipt')
            for key in ('cgroup_cpu_quota_state', 'cgroup_cpu_quota_evidence_complete',
                        'cgroup_cpu_quota_unknown_levels', 'unknown_capacity_fields', 'raw_evidence'):
                self.assertEqual(row['resources'][key], expected[key])
            self.assertEqual(row['resources']['subprocess_gomaxprocs'], 2)
            self.assertEqual(row['logical_worker_ratio'], .25)
            self.assertEqual(row['measurement_classification'], 'oversubscribed worker measurement')
            self.assertFalse(row['resources']['core_scaling_qualified'])
            self.assertFalse(row['resources']['capacity_is_exclusive_physical_cores'])

    def test_report_and_cell_wiring(self):
        for values, state, bound, complete in ((['max 100000'], 'unlimited', None, True),
                (['max 0'], 'unknown', None, False),
                (['bad', '50000 100000'], 'limited', .5, False)):
            evidence = captured(2, values)
            expected = self.check_summary(evidence, state, bound, complete)
            self.check_wiring(evidence, expected)

    def test_resource_test_import_writes_nothing(self):
        spec = importlib.util.spec_from_file_location('resource_import_control', HERE / 'test_resources.py')
        module = importlib.util.module_from_spec(spec)
        original_open = builtins.open
        def read_only_open(file, mode='r', *args, **kwargs):
            if any(flag in mode for flag in 'wax+'):
                raise AssertionError('import must not write')
            return original_open(file, mode, *args, **kwargs)
        with mock.patch('os.mkdir', side_effect=AssertionError('import must not mkdir')), \
             mock.patch('builtins.open', side_effect=read_only_open), \
             mock.patch('io.open', side_effect=read_only_open):
            spec.loader.exec_module(module)


if __name__ == '__main__':
    unittest.main()
