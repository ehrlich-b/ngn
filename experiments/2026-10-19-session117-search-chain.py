"""After a clean failed claim, run the checked continuation/pawn/ProbCut queue."""
import datetime,importlib.util,json,subprocess
from pathlib import Path
R=Path('/home/ehrli/ngn-data/night-20261007');D=R/'session117-search'
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
def report(state,**detail):
 result=dict(state=state,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),automatic_adoption=False,**detail)
 (D/'result.json').write_text(json.dumps(result,indent=2)+'\n')
 with (R/'status.jsonl').open('a') as f:f.write(json.dumps(dict(utc=result['utc'],stage='session117-search',event=state,detail=detail))+'\n')
def main():
 runner=module('campaign_queue',R/'session93/campaign_queue.py')
 for name in ['continuation','pawn','probcut']:
  job=json.loads((D/(name+'-nomination.json')).read_text())
  report('RUNNING',candidate=name,output=job['output'])
  result=runner.run(job,R/'session80/controls.json',R/'status.jsonl')
  if result['verdict']!='SHELVED' or result['faults']:
   report('REVIEW_SEARCH_GATE',candidate=name,verdict=result['verdict'],next='Adopt only after review; retarget all unstarted work after a keep')
   return
 speed=R/'session117-speed/s5'
 assert json.loads((speed/'checks-result.json').read_text())['state']=='S5_CHECKS_PASS'
 report('S5_TIMING',rule='Unchanged fixed40/both-strata/all-topology lower95>1')
 subprocess.run(['taskset','-c','0-15','python3',str(R/'session117-speed/speed-gate.py'),
  '--candidate','s5','--profile-admission',str(speed/'admission.json'),'--out',str(speed/'timing'),
  '--base-binary',str(R/'session117/accepted-ngn'),'--candidate-binary',str(speed/'s5-ngn'),
  '--identity',str(speed/'identity.json')],check=True)
 report('REVIEW_S5',result=json.loads((speed/'timing/result.json').read_text()),next='Review before adoption or S6 admission; S6 timing remains unarmed')
if __name__=='__main__':
 try:main()
 except BaseException as error:
  report('HELD_REVIEW',error=str(error));raise
