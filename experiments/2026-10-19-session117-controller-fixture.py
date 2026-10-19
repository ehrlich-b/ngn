"""Rig-only mocked branch checks for the new orchestration; zero engine launches."""
import importlib.util,json,tempfile,types
from pathlib import Path
R=Path('/home/ehrli/ngn-data/night-20261007');D=R/'session117'
spec=importlib.util.spec_from_file_location('under_test',D/'claim-chain.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
real_read=m.read
for screen_ok,confirm_ok in [(False,False),(True,False),(True,True)]:
 with tempfile.TemporaryDirectory(prefix='ngn-s117-fixture-') as temp:
  m.D=Path(temp);events=[];commands=[];writes=[]
  m.wait_checks=lambda:None
  m.read=lambda p: {'source_commit':'fixture','identity':str(m.D/'candidate.json')} if p==m.D/'package.json' else ({'SourceCommit':'fixture'} if p==m.D/'candidate.json' else ({'state':'READY_TOOLING'} if p.name=='build-receipt.json' else {'state':'REVIEW_SEARCH_GATE'}))
  m.report=lambda state,**detail:events.append(state)
  m.old.write=lambda p,v:writes.append((p,v))
  def stage(mode,*args):
   events.append('stage-'+mode)
   return dict(ScreenEligible=screen_ok,ClaimEligible=confirm_ok,Primary={'Estimate':3560},Anchors=[])
  m.old.claim_stage=stage
  m.subprocess=types.SimpleNamespace(Popen=lambda *a,**k:types.SimpleNamespace(terminate=lambda:None,wait=lambda:0),run=lambda argv,**kw:commands.append(argv))
  m.main()
  assert ('stage-confirmation' in events)==screen_ok,events
  assert ('GOAL_MET' in events)==(screen_ok and confirm_ok),events
  assert bool(commands)==(not (screen_ok and confirm_ok)),events
  assert writes[0][1]['expected_pool_gain'][0]>=20
  print(json.dumps(dict(screen=screen_ok,confirmation=confirm_ok,events=events,search_fallback=bool(commands))))
