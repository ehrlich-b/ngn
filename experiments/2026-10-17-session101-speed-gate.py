"""Rig-only fixed balanced timing for ONE profile-admitted speed candidate.

Use after reviewing the exclusive panel. Each candidate must be rebuilt and nominated against
the immediate accepted base after an earlier speed candidate is accepted. This script never adopts code or starts a second gate.
"""
import argparse
import datetime
import hashlib
import json
import math
from pathlib import Path
import statistics

import sys
sys.path.insert(0, '/home/ehrli/ngn-data/night-20261007/session100-speed')
from panel import NET, R, S, UCI, write


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('--candidate',choices=['s1','s2','s3'],required=True)
    parser.add_argument('--profile-admission',type=Path,required=True,
                        help='Short nomination stating the measured hotspot and immediate accepted base')
    parser.add_argument('--out',type=Path,required=True)
    parser.add_argument('--candidate-binary',type=Path,required=True)
    parser.add_argument('--base-binary',type=Path,required=True)
    parser.add_argument('--identity',type=Path,required=True)
    parser.add_argument('--smoke',action='store_true')
    args=parser.parse_args()
    admission=json.loads(args.profile_admission.read_text())
    if not args.smoke:
        assert admission['candidate']==args.candidate and admission['profile_admitted']
        assert json.loads((S/'panel/result.json').read_text())['state']=='MEASURED'
        identity=json.loads(args.identity.read_text())
        assert identity['verdict']=='PASS'
        if args.candidate=='s1':
            assert json.loads((S/'s1-prefault-identity.json').read_text())['verdict']=='PASS'
            assert json.loads((S/'s1-prefault-hugepage-witness.json').read_text())['anon_huge_kib']>=256*1024
    args.out.mkdir(exist_ok=False)
    frames=json.loads((R/'session77-smp-panel/panel.json').read_text())
    binaries=dict(base=args.base_binary,candidate=args.candidate_binary)
    hashes={key:hashlib.sha256(path.read_bytes()).hexdigest() for key,path in binaries.items()}
    if not args.smoke:
        assert hashes==identity['binary_sha256'], (hashes,identity['binary_sha256'])
    topology=[(1,64)] if args.smoke else [(1,64),(1,256),(8,256)]
    count,ms=(2,500) if args.smoke else (40,20000)
    manifest=dict(candidate=args.candidate,admission=admission,binary_sha256=hashes,
        identity_receipt=str(args.identity),identity_sha256=hashlib.sha256(args.identity.read_bytes()).hexdigest(),
        net=str(NET),net_sha256=hashlib.sha256(NET.read_bytes()).hexdigest(),topologies=topology,
        paired_blocks=count,milliseconds_per_engine=ms,frame='(block//2)%12; one frame per block',
        order='AB/BA alternating; both engines receive identical frame per pair',
        rule='Every topology and both order strata: paired log-NPS Student-t lower95>0,df19; otherwise SHELVED; no extensions',
        smoke=args.smoke,utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    write(args.out/'manifest.json',manifest)
    records=[]
    results=[]
    with (args.out/'blocks.jsonl').open('x') as journal:
        for width,hash_mib in topology:
            engines={key:UCI(path,width,hash_mib,args.out,smoke=args.smoke) for key,path in binaries.items()}
            try:
                for e in engines.values():
                    e.search(frames[0],500 if args.smoke else 2000)
                for block in range(count):
                    order=['base','candidate'] if block%2==0 else ['candidate','base']
                    cells={key:engines[key].search(frames[(block//2)%12],ms) for key in order}
                    row=dict(width=width,hash_mib=hash_mib,block=block,order=order,cells=cells,
                        ratio=cells['candidate']['nps']/cells['base']['nps'])
                    records.append(row)
                    journal.write(json.dumps(row)+'\n')
                    journal.flush()
            finally:
                for e in engines.values(): e.close()
            if not args.smoke:
                strata=[]
                for parity in (0,1):
                    logs=[math.log(r['ratio']) for r in records if r['width']==width
                          and r['hash_mib']==hash_mib and r['block']%2==parity]
                    assert len(logs)==20
                    mean=statistics.mean(logs)
                    margin=2.093024054408263*statistics.stdev(logs)/math.sqrt(len(logs))
                    strata.append(dict(ratio=math.exp(mean),ci95=[math.exp(mean-margin),math.exp(mean+margin)]))
                results.append(dict(width=width,hash_mib=hash_mib,strata=strata,
                    pass_rule=all(s['ci95'][0]>1 for s in strata)))
    verdict='SMOKE_PASS' if args.smoke else ('KEEP_ELIGIBLE' if all(r['pass_rule'] for r in results) else 'SHELVED')
    write(args.out/'result.json',dict(verdict=verdict,topologies=results,
        finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),automatic_adoption=False,
        next='Review identity and immediate-base match before integration; no Elo gain inferred'))


if __name__=='__main__': main()
