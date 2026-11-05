#!/usr/bin/env python3
"""Reproduce two CPU training updates and check the committed nonzero fixture."""
import argparse
import hashlib
import io
import json
from pathlib import Path
import sys

import numpy as np

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))
from trainer.export import export_bytes
from trainer.ngnp import RECORD_DTYPE, decode_record, to_fen
from trainer.residual_refeval import Network
from trainer.tests.residual_fixture import PACKET, SOURCE, train_fixture


def packet():
    model, training = train_fixture()
    raw = export_bytes(model.state_dict(), 6)
    oracle = Network(raw)
    cases = []
    records = np.frombuffer((PACKET/'records.ngnp').read_bytes(), dtype=RECORD_DTYPE)
    assert len(records) == 16
    for record in records:
        p = decode_record(record)
        linear, residual = oracle.components(p)
        cases.append(dict(fen=to_fen(p), bucket=oracle.bucket(p), stm=p.stm,
                          cp=oracle.evaluate(p), linear_cp=linear, residual_cp=residual,
                          acc=oracle.accumulators(p).reshape(-1).tolist()))
    weights = io.BytesIO()
    np.savez(weights, **{k: v.detach().numpy() for k, v in model.state_dict().items()})
    expected = dict(format='NGNN6', sha256=hashlib.sha256(raw).hexdigest(), file_bytes=len(raw),
                    source_sha256=hashlib.sha256(SOURCE.read_bytes()).hexdigest(),
                    records_sha256=hashlib.sha256((PACKET/'records.ngnp').read_bytes()).hexdigest(),
                    training=training, cases=cases,
                    claim='Integer parity and staged gradient proof only; no speed or Elo claim.')
    return {'trained-h32.ngn6': raw, 'weights.npz': weights.getvalue(),
            'expected.json': (json.dumps(expected, indent=2)+'\n').encode()}


if __name__ == '__main__':
    cli = argparse.ArgumentParser(description=__doc__)
    cli.add_argument('--write', action='store_true', help='Explicitly regenerate the committed fixture')
    args = cli.parse_args()
    for name, data in packet().items():
        if args.write:
            (PACKET/name).write_bytes(data)
        else:
            assert (PACKET/name).read_bytes() == data, 'stale fixture: '+name
        print(f'{name}: {len(data)} bytes sha256={hashlib.sha256(data).hexdigest()}', flush=True)
