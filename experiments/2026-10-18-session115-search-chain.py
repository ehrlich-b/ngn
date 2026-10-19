"""Run checked search candidates on warm2000, then its retargeted S5 timing."""
import datetime
import importlib.util
import json
import os
from pathlib import Path
import subprocess

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session115-search'


def module(name,path):
    spec=importlib.util.spec_from_file_location(name,path)
    m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
    return m


def report(state,**detail):
    result=dict(state=state,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),automatic_adoption=False,**detail)
    (D/'result.json').write_text(json.dumps(result,indent=2)+'\n')
    with (R/'status.jsonl').open('a') as f:
        f.write(json.dumps(dict(utc=result['utc'],stage='session115-search',event=state,detail=detail))+'\n')


def main():
    (D/'controller.pid').write_text(str(os.getpid())+'\n')
    runner=module('campaign_queue',R/'session93/campaign_queue.py')
    for name in ['qsearch','continuation','pawn','probcut']:
        job=json.loads((D/(name+'-nomination.json')).read_text())
        report('RUNNING',candidate=name,output=job['output'])
        result=runner.run(job,R/'session80/controls.json',R/'status.jsonl')
        if result['verdict']!='SHELVED' or result['faults']:
            report('REVIEW_SEARCH_GATE',candidate=name,verdict=result['verdict'],
                   next='Review before adoption; retarget successors after any keep; faults remain unscored')
            return
    config=json.loads((D/'successor.json').read_text())
    report('WAITING_S5_CHECKS',pid=config['checks_pid'])
    module('s101_wait',R/'session101-speed/boundary.py').wait_pid(config['checks_pid'])
    assert json.loads((R/'session115-speed/checks-result.json').read_text())['state']=='S5_CHECKS_PASS'
    report('S5_TIMING',rule='Fixed40 paired20s; both strata lower95>1 at all three topologies')
    subprocess.run(['bash',str(R/'session115-speed/timing.sh')],check=True)
    result=json.loads((R/'session115-speed/s5-timing/result.json').read_text())
    report('REVIEW_S5',result=result,next='KEEP_ELIGIBLE requires manual integration; shelf closes alpha-beta and moves to SEE')


if __name__=='__main__':
    try: main()
    except BaseException as error:
        report('HELD_REVIEW',error=str(error))
        raise
