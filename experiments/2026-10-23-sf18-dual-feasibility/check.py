#!/usr/bin/env python3
"""Bounded, offline acceptance: replay legal moves and check arithmetic/costs."""
import ctypes
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
HERE = Path(__file__).resolve().parent


def trunc(n, d):
    return (-1 if n < 0 else 1) * (abs(n) // d)


def read(name):
    return json.loads((HERE / name).read_text())


def main():
    changed = subprocess.check_output([
        'git', 'diff', '--name-only', '8ba878cb03a7eb81ca465b3d562953d0f45e81c7',
    ], cwd=ROOT, text=True).splitlines()
    assert all(name == 'TODO.md' or name.startswith('experiments/2026-10-23-sf18-dual-feasibility/')
               for name in changed), 'production scope changed'
    p1 = (ROOT / 'experiments/2026-09-19-p1-staged-picker-profile/README.md').read_text()
    assert 'P1 is admitted' in p1
    p = read('provenance.json')
    assert p['study'] == 'stockfish-18-big-dual'
    assert p['source_commit'] == 'cb3d4ee9b47d0c5aae855b12379378ea1439675c'
    assert p['network_archive_commit'] == 'dd5c7f74c073a7d0bcbb52648d26c537a3d04cf4'
    for m in p['models']:
        text = (ROOT / m['local_identity_source']).read_text()
        assert m['sha256'] in text
        assert str(m['bytes']) in text.replace(',', '')
    assert 'incorporation blocked' in p['provenance_status']
    a = read('arithmetic.json')
    for v in a['wrapped_pairs']:
        wrapped = ctypes.c_int16(v['base'] + v['threat']).value
        assert wrapped == v['wrapped']
        assert min(255, max(0, wrapped)) * v['other'] // 512 == v['output']
    for v in a['psqt']:
        assert trunc(v['difference'], 2) == v['halved']
        assert trunc(v['halved'], 16) == v['component']
    v = a['affine']
    assert ctypes.c_int32(v['bias'] + v['input'] * v['weight']).value == v['wrapped']
    v = a['activation']
    assert min(127, (v['input'] * v['input']) >> 19) == v['squared']
    assert min(127, max(0, v['input'] >> 6)) == v['clipped']
    v = a['skip']
    assert trunc(ctypes.c_int32(v['fc0_15'] * 9600).value, 8128) == v['forward']
    v = a['blend']
    assert trunc(125*v['psqt'] + 131*v['positional'], 128) == v['expected']
    for v in a['selector']:
        small = abs(v['simple']) > 962
        assert ('SMALL' if small else 'BIG') == v['first']
        assert (small and abs(v['small_blend']) < 277) == v['fallback']

    run = subprocess.run([
        'go', 'run', './experiments/2026-10-23-sf18-dual-feasibility',
        'nnue/sf18small/testdata/upstream_incremental_oracle/sequences.json',
        'experiments/2026-10-23-sf18-dual-feasibility/sequences.json',
    ], cwd=ROOT, capture_output=True, text=True)
    if run.returncode:
        print(run.stderr, file=sys.stderr)
        run.check_returncode()
    data = json.loads(run.stdout)
    rows = data['rows']
    assert data['cases'] == 151 and len(rows) == 757
    assert {r['stack'] for r in rows} == set(range(8))
    assert sum(r['action'] not in ('root', 'pop') for r in rows) == 303
    assert sum(r['action'] == 'null' for r in rows) > 0
    assert {r['first_network'] for r in rows} == {'BIG', 'SMALL'}
    for r in rows:
        # Independent operation formulas, not a neural evaluation.
        assert r['dual_base_lane_adds'] == r['base_columns'] * (1024+128)
        assert r['big_threat_lane_adds_upper'] == r['threat_columns_upper'] * 1024
        assert r['dual_psqt_adds_upper'] == (2*r['base_columns'] + r['threat_columns_upper']) * 8
        if r['action'] in ('null', 'pop'):
            assert r['base_columns'] == r['threat_columns_upper'] == 0
        small = abs(r['simple_eval_stm']) > 962
        big_macs = 16*1024 + 32*32 + 32
        small_macs = 16*128 + 32*32 + 32
        assert r['selected_dense_macs_lower'] == (small_macs if small else big_macs)
        assert r['selected_dense_macs_upper'] == big_macs + (small_macs if small else 0)

    # Read-only tensors, numeric frame payloads and optional inherited cache estimate.
    big_tensors = 1024*2 + 22528*1024*2 + 79856*1024 + (22528+79856)*8*4 + 8*(16*4+16*1024+32*4+32*32+4+32)
    small_tensors = 128*2 + 22528*128*2 + 22528*8*4 + 8*(16*4+16*128+32*4+32*32+4+32)
    frame = 2*(128*2+8*4) + 2*(1024*2+8*4)*2
    assert big_tensors == 131329312 and small_tensors == 6514720 and frame == 8896
    memory = {
        'shared_tensor_bytes': big_tensors+small_tensors,
        'numeric_frame_bytes': frame,
        'numeric_stack_128_bytes_per_worker': frame*128,
        'numeric_stack_247_bytes_per_worker': frame*247,
        'eight_workers_247_plus_inherited_cache_estimate_bytes': 8*(frame*247+327680),
        'peak_model_replacement_tensor_bytes': 2*(big_tensors+small_tensors),
    }
    pushes = [r for r in rows if r['action'] not in ('root', 'pop')]
    summary = {
        'cases': data['cases'], 'checkpoints': len(rows), 'pushes': len(pushes),
        'material_stacks': sorted({r['stack'] for r in rows}),
        'first_network_counts': {n: sum(r['first_network']==n for r in rows) for n in ('BIG', 'SMALL')},
        'memory': memory,
        'push_totals': {k: sum(r[k] for r in pushes) for k in (
            'base_columns', 'threat_columns_upper', 'dual_base_lane_adds',
            'big_threat_lane_adds_upper', 'dual_psqt_adds_upper', 'eager_numeric_copy_bytes')},
        'push_maxima': {k: max(r[k] for r in pushes) for k in (
            'base_columns', 'threat_columns_upper', 'dual_base_lane_adds',
            'big_threat_lane_adds_upper', 'dual_psqt_adds_upper')},
        'selected_dense_macs_all_checkpoints_lower': sum(r['selected_dense_macs_lower'] for r in rows),
        'selected_dense_macs_all_checkpoints_upper': sum(r['selected_dense_macs_upper'] for r in rows),
        'arithmetic_witnesses': 'PASS; synthetic only; no donor oracle executed',
        'provenance': p['provenance_status'],
    }
    print(json.dumps({'summary': summary, 'trace': data}, indent=2))


if __name__ == '__main__':
    main()
