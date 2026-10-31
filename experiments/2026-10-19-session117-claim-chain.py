"""One prospective changed-package screen, automatic confirmation, then search.

Uses the existing v4 launcher/auditor/statistics with the approved 3555 rule.
No running gate is stopped, no training resumed, and no package auto-adopted.
"""
import importlib.util,json,os,select,subprocess
from pathlib import Path
R=Path('/home/ehrli/ngn-data/night-20261007');D=R/'session117'
A=Path('/home/ehrli/ngn-data/claim-v4-20261016/attempt-002')
T=A.parent/'tooling-s117'
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
old=module('claim_chain',R/'session94/claim-chain.py')
old.D=D;old.V=T
read=lambda p:json.loads(p.read_text())
def report(state,**detail):
 result=dict(state=state,utc=old.now(),automatic_adoption=False,**detail)
 (D/'result.json').write_text(json.dumps(result,indent=2)+'\n')
 with (R/'status.jsonl').open('a') as f:f.write(json.dumps(dict(utc=result['utc'],stage='session117-claim',event=state,detail=detail))+'\n')
def wait_checks():
 pid=int((D/'checks-launcher.pid').read_text())
 report('WAITING_CHECKS',pid=pid)
 try:fd=os.pidfd_open(pid)
 except OSError as error:
  if error.errno not in (3,22):raise
 else:
  try:select.select([fd],[],[])
  finally:os.close(fd)
 assert (D/'checks.exit').read_text().strip()=='0'
 assert read(D/'checks-result.json')['state']=='ALL_RETARGET_CHECKS_PASS'
def main():
 (D/'controller.pid').write_text(str(os.getpid())+'\n')
 wait_checks()
 package=read(D/'package.json');candidate=Path(package['identity'])
 assert read(T/'build-receipt.json')['state']=='READY_TOOLING'
 assert package['source_commit']==read(candidate)['SourceCommit']
 speed=6+.6*(21.568671346554353+8.326727710505265)
 old.write(A/'nomination.json',dict(authority='Bryan Oct17 09:45 screen point>=3555; fresh attempt after plausible +20 pool accepted gains',
  candidate=str(candidate),accepted_source=package['source_commit'],accepted_net='81215229',vector='compiled warm2000',
  prior_attempt='attempt-001 remains failed; no reanalysis or pooling',
  expected_pool_gain=[speed,speed+1],expected_screen_point=[3535.365648330272+speed,3535.365648330272+speed+1],
  expectation='Planning hypothesis: speed +6 to +7, 60% transfer of warm SPSA +21.5687 and qsearch +8.3267; no width credit',
  screen='Fixed320 at10+0.1; pooled point>=3555 and raw scores bracket50%;zero faults/exclusions',
  confirmation='On eligible screen only: fixed640 at120+1,strict pooled lower95>3600,raw scores bracket50%;zero faults/exclusions',
  resources='Exclusive rig0-15,nice19,no CPU quota; engines physical8,GOMAXPROCS8;GPU idle;hopper untouched',
  occupancy='Existing real-launcher occupancy probe before games; same scope, cpu.max recorded',
  monitor='Existing descriptive resources.jsonl every600seconds;no score stop or verdict effect',
  successors='On clean failed screen/confirmation: checked continuation,pawn,ProbCut serially; clean shelves advance;KEEP/PROVISIONAL/fault hold;then S5;no S6 auto-timing',
  created_utc=old.now()))
 sidecar=subprocess.Popen(['python3',str(R/'session94/monitor.py'),str(os.getpid()),str(A/'resources.jsonl')])
 try:
  report('SCREEN_RUNNING',attempt=str(A))
  screen=old.claim_stage('screen',candidate,A)
  if screen['ScreenEligible']:
   report('CONFIRMATION_RUNNING',screen_primary=screen['Primary'])
   confirmation=old.claim_stage('confirmation',candidate,A)
   if confirmation['ClaimEligible']:
    report('GOAL_MET',analysis=str(A/'confirmation/analysis.json'),primary=confirmation['Primary'])
    return
   report('CONFIRMATION_FAILED',primary=confirmation['Primary'])
  else:report('SCREEN_FAILED',primary=screen['Primary'],anchors=screen['Anchors'])
 finally:
  sidecar.terminate();sidecar.wait()
 report('SEARCH_QUEUE_RUNNING',next='continuation,pawn,ProbCut;then S5 on three clean shelves')
 subprocess.run(['python3',str(D/'search-chain.py')],check=True)
 report('REVIEW_SEARCH_OR_SPEED',result=read(R/'session117-search/result.json'))
if __name__=='__main__':
 try:main()
 except BaseException as error:
  report('HELD_REVIEW',error=str(error));raise
