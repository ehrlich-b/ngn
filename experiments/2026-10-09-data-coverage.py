"""Estimate distinct occupied boards with a fixed bottom-k hash sample.

Every 32-byte archive row is scanned, but only 65,536 smallest distinct
board hashes are retained per source. Labels/padding do not affect identity.
This is an estimate of occupied-board diversity, not game/symmetry diversity.
"""
import json,math,os,select,sys,time
from pathlib import Path
import numpy as np
from trainer.archive import ARCHIVE_DTYPE,board_hash

D=Path('/home/ehrli/ngn-data/night-20261007/session19-data')
K=65536

def update(sample, hashes):
    if len(sample)==K: hashes=hashes[hashes<sample[-1]]
    if not len(hashes):return sample
    values=np.unique(np.concatenate((sample,hashes)))
    if len(values)>K:
        values=np.partition(values,K-1)[:K];values.sort()
    return values

def estimate(sample):
    if len(sample)<K:return {'distinct':len(sample),'exact_below_k':True}
    n=(K-1)*2**64/(int(sample[-1])+1)
    relative=1.96/math.sqrt(K-2)
    return {'distinct':n,'approx_ci95':[n*(1-relative),n*(1+relative)],'relative_standard_error':1/math.sqrt(K-2),'exact_below_k':False}

def main():
    # Native wait; no repeated status/log scans while the corpus is prepared.
    if not (D/'controller.done').exists():
        fd=os.pidfd_open(int((D/'controller.pid').read_text()));p=select.poll();p.register(fd,select.POLLIN)
        assert p.poll(12*3600*1000),'corpus preparation wait timed out';os.close(fd)
    assert (D/'controller.exit').read_text().strip()=='0'
    sources=[('t80',Path('/home/ehrli/nnue-owned-k4-20260920/archive-a1/corpus-a1/train-archive.bf')),('t76',D/'t76.bf'),('t74',D/'t74.bf')]
    results={};union=np.empty(0,dtype=np.uint64);samples={};started=time.monotonic()
    for name,path in sources:
        rows=np.memmap(path,dtype=ARCHIVE_DTYPE,mode='r');sample=np.empty(0,dtype=np.uint64)
        for start in range(0,len(rows),262144):sample=update(sample,board_hash(rows[start:start+262144],20261006))
        samples[name]=sample;union=update(union,sample)
        results[name]={'path':str(path),'bytes':path.stat().st_size,'records':len(rows),**estimate(sample)}
    before=results['t80']['distinct'];after=estimate(union)
    x={'method':'bottom-65536 distinct 64-bit board hashes, ngn-board-mix64-v1 seed20261006','sources':results,'combined':after,'estimated_unique_ratio':after['distinct']/before,'record_ratio':sum(v['records'] for v in results.values())/results['t80']['records'],'seconds':time.monotonic()-started,'limits':'occupied boards, not original games or mirrored model-input classes; interval is approximate and source/union errors are correlated','training_launched':False}
    (D/'coverage.json').write_text(json.dumps(x,indent=2)+'\n');print(json.dumps(x))
if __name__=='__main__':main()
