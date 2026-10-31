"""Bounded source/output-phase diagnostics on the existing content holdout.

Prepare fixed uniform byte windows before looking at either network. Conversion,
partitioning and integer evaluation run on forge; these diagnostics never select
weights or change the fixed game verdict.
"""
import argparse
import hashlib
import json
import math
from pathlib import Path

import numpy as np
from trainer.archive import ARCHIVE_DTYPE, board_hash, canonical_records
from trainer.ngnp import RECORD_DTYPE, decode_record
from trainer.refeval import Network

SEED=20261006
MODULUS=1024
WINDOWS=64
WINDOW_RECORDS=8192


def prepare(root):
    root=Path(root)
    manifest=json.loads((root/'sample-manifest.json').read_text())
    receipts=[]
    for source in manifest['sources']:
        path=root/(source['name']+'.raw')
        raw=np.fromfile(path,dtype=ARCHIVE_DTYPE)
        assert len(raw)==WINDOWS*WINDOW_RECORDS
        chosen=raw[(board_hash(raw,SEED)&np.uint64(MODULUS-1))==0]
        rows=canonical_records(chosen)
        output=root/(source['name']+'.ngnp')
        rows.tofile(output)
        receipts.append(dict(source=source['name'],sample_records=len(raw),holdout_records=len(rows),sample_sha256=hashlib.sha256(path.read_bytes()).hexdigest(),ngnp_sha256=hashlib.sha256(output.read_bytes()).hexdigest()))
    (root/'sample-receipt.json').write_text(json.dumps(dict(seed=SEED,modulus=MODULUS,windows=WINDOWS,window_records=WINDOW_RECORDS,sources=receipts),indent=2)+'\n')


def sigmoid(cp):
    return 1/(1+math.exp(-cp/400))


def evaluate(root,base,candidate,receipt):
    root=Path(root);training=json.loads(Path(receipt).read_text())
    assert training['epochs']==58 and training['reexport_identical'] and training['parity_cases']==20
    digest=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
    assert digest(base)=='be34532cf3d053d3901cc28eceb1fb03f1dc3bcafede23c4bc17128f48f5cd7a'
    assert digest(candidate)==training['net_sha256']
    networks=[Network(base),Network(candidate)]
    assert all((n.version,n.hidden,n.king_buckets,n.buckets)==(3,512,8,8) for n in networks)
    rows=[]
    samples=json.loads((root/'sample-receipt.json').read_text())
    for source in samples['sources']:
        path=root/(source['source']+'.ngnp')
        assert digest(path)==source['ngnp_sha256']
        records=np.fromfile(path,dtype=RECORD_DTYPE)
        sums={}
        for record in records:
            position=decode_record(record);bucket=networks[0].bucket(position)
            assert bucket==networks[1].bucket(position)
            teacher=sigmoid(int(record['score']));result=int(record['result'])/2
            target=.75*teacher+.25*result
            group=sums.setdefault(bucket,[])
            probabilities=[sigmoid(n.evaluate(position)) for n in networks]
            group.append((teacher,result,target,*probabilities))
        for bucket,group in sorted(sums.items()):
            a=np.asarray(group,dtype=np.float64)
            metrics=[]
            for index,name in [(3,'base'),(4,'candidate')]:
                delta=a[:,index]-a[:,2]
                metrics.append(dict(net=name,mean_probability=float(a[:,index].mean()),mean_teacher_residual=float((a[:,index]-a[:,0]).mean()),mean_blended_residual=float(delta.mean()),blended_mse=float(np.square(delta).mean())))
            rows.append(dict(source=source['source'],output_phase_bucket=bucket,records=len(a),small_cell=len(a)<32,mean_teacher_probability=float(a[:,0].mean()),mean_result=float(a[:,1].mean()),mean_blended_target=float(a[:,2].mean()),metrics=metrics))
    report=dict(base_net_sha256=digest(base),candidate_net_sha256=digest(candidate),training_receipt_sha256=digest(receipt),sample_receipt=samples,rows=rows,decision='diagnostic only; no net selection, rescue games, source weighting or playing adoption',limitations=['Fixed clustered windows are a bounded selection-holdout sample, not an independent or population confidence interval.','Counts are records; repeated boards and tiny output-phase cells remain explicit.','Integer reference evaluation; original full-holdout selection and fixed400-game rule stay unchanged.'])
    (root/'residuals.json').write_text(json.dumps(report,indent=2)+'\n')
    print(json.dumps(report,indent=2))


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('mode',choices=['prepare','evaluate']);p.add_argument('root');p.add_argument('--base');p.add_argument('--candidate');p.add_argument('--training-receipt');a=p.parse_args()
    if a.mode=='prepare':prepare(a.root)
    else:evaluate(a.root,a.base,a.candidate,a.training_receipt)
