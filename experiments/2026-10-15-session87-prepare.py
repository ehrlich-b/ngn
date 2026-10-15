"""Retarget approved width and Final174 nominations to the accepted package."""
import hashlib
import json
from pathlib import Path

R=Path('/home/ehrli/ngn-data/night-20261007')
D=R/'session87'


def main():
    source=json.loads((R/'session84/search-queue.json').read_text())
    base=source['jobs'][0]
    engine=base['base']
    pins=base['artifacts'].copy()
    pins.pop(base['candidate'])
    # UCI alternates turns with pondering off. A/A roles share four logical
    # CPUs; no opponent runs on an SMT sibling and training keeps12,14.
    roles={'ngn4-a':('0,2,4,6',4),'ngn4-b':('0,2,4,6',4),'ngn1-c':('8',1)}
    for name,(cpus,width) in roles.items():
        path=D/name
        path.write_text(f'#!/bin/bash\nexec taskset -c {cpus} env GOMAXPROCS={width} {engine} "$@"\n')
        path.chmod(0o755)
        pins[str(path)]=hashlib.sha256(path.read_bytes()).hexdigest()
    common=dict(base)
    for key in ['source_checks','candidate_source','treatment','review_after']:
        common.pop(key,None)
    common.update(engine=engine,candidate_source='9ec5d31',candidate=str(D/'ngn4-a'),base=str(D/'ngn4-b'),
                  candidate_options=[('Hash=256' if o.startswith('Hash=') else 'Threads=4' if o.startswith('Threads=') else o) for o in base['base_options']],
                  max_games=40,concurrency=1,artifacts=pins,
                  control_key='rig/same-team-T4/Hash256/c1/10+0.1/GOMAX4/'+pins[base['harness']],
                  approval='Bryan October15: proceed with using cores; merged13:05 plan; fresh probe and A/A before fixed400')
    common['base_options']=common['candidate_options'].copy()
    aa=dict(common,name='s87-aa-true-width',output=str(D/'aa-true-width'),cpus='0-11',
            **{'class':'aa'},affinity='BothT4 roles0,2,4,6; alternating turns/ponderoff; four physical cores per engine,no SMT-sibling contention; trainer12,14',
            decision='Fixed40/20, printed and exact interval contain0,zero faults; no score stopping')
    measurement=dict(common,name='s87-true-width',output=str(D/'true-width'),cpus='0-11',
            **{'class':'measurement'},base=str(D/'ngn1-c'),max_games=400,
            base_options=[('Threads=1' if o=='Threads=4' else o) for o in common['base_options']],
            affinity='T4a0,2,4,6/GOMAX4 vsT1c8/GOMAX1; five distinct physical cores; loader12,14',
            decision='Fixed400/200; >=+60 closes SMP policy work pending integrated long-clock; <+50 opens5-20s T4 profile;50-60 unresolved; no source adoption')
    ready=json.loads((R/'session84-training174/result.json').read_text())
    assert ready['epochs']==174 and ready['reexport_identical'] and ready['parity_cases']==20
    netjob=dict(base,name='s87-net-final174',output=str(D/'net-final174'),**{'class':'net'},
                candidate=engine,base=engine,max_games=1000,review_after=True,
                candidate_options=[('EvalFile='+str(R/'session84-training174/trained/latest.nnue') if o.startswith('EvalFile=') else o) for o in base['base_options']],
                candidate_source='9ec5d31',treatment='Final174 only vs acceptedFinal115; identical engine/final14/Threads1/Hash64',
                net_sha256=ready['net_sha256'],base_net_sha256=base['net_sha256'],
                decision='Fixed1000/500 at10+0.1/c6; printed AND exact lower95>0,zero faults; no extension/checkpoint shopping; winning net transfer400/60+0.6 point>0 before foreign games')
    netjob['artifacts']=base['artifacts'].copy()
    netjob['artifacts'].pop(base['candidate'])
    netjob['artifacts'][str(R/'session84-training174/trained/latest.nnue')]=ready['net_sha256']
    netjob['finalization']=ready
    queue=dict(controls=source['controls'],status=source['status'],
               after=dict(pid=39843,result=str(R/'session83/provenance/result.json'),verdicts=['CERTIFIED','BISECT','UNCERTIFIED_CAP']),
               preflight_argv=['python3',str(D/'width-preflight.py')],jobs=[measurement,netjob],
               next='Review Final174; KEEP transfers then adoption/retarget; SHELVED warm24kSPSA on81215229; provenance H0/cap is separate debt,never restart/pool.')
    for name,value in [('width-probe.json',dict(aa=aa,measurement=measurement)),('queue.json',queue)]:
        (D/name).write_text(json.dumps(value,indent=2)+'\n')
    print('Prepared true-width A/A40, measurement400 and Final174 gate1000 on9ec5d31/81215229/final14')


if __name__=='__main__':
    main()
