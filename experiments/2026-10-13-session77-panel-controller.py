"""Native SEE completion -> bounded SMP panel; no engine adoption or games."""
import ctypes,datetime,hashlib,json,os,pathlib,re,select,subprocess,time
p=pathlib.Path('/home/ehrli/ngn-data/night-20261007/session77-smp-panel')
see=p.parent/'session77-see-rig'
def write(name,value): (p/name).write_text(json.dumps(value,indent=2,allow_nan=False)+'\n')
def status(event,detail):
 with (p/'status.jsonl').open('a') as f:f.write(json.dumps({'utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'event':event,'detail':detail})+'\n')
def wait_done(directory,name):
 if (directory/name).exists():return
 libc=ctypes.CDLL(None,use_errno=True);fd=libc.inotify_init1(os.O_CLOEXEC|os.O_NONBLOCK);assert fd>=0
 try:
  assert libc.inotify_add_watch(fd,os.fsencode(directory),0x8|0x80)>=0
  deadline=time.monotonic()+24*3600
  while not (directory/name).exists():
   remaining=deadline-time.monotonic();assert remaining>0
   readable,_,_=select.select([fd],[],[],remaining);assert readable;os.read(fd,65536)
 finally:os.close(fd)
def rows(name):return [json.loads(line.split('S77PANEL ',1)[1]) for line in (p/name).read_text().splitlines() if line.startswith('S77PANEL ')]
def opportunity(row):
 ws=row['workers'];primary=ws[0]['iterations'][-1];groups={};reps={};classes={}
 mate=lambda score:1 if score>=29900 else -1 if score<=-29900 else 0
 if mate(primary['score'])>0:return False
 for i,w in enumerate(ws):
  if not w['iterations']:continue
  v=w['iterations'][-1]
  if v['depth']<primary['depth']:continue
  groups[v['move']]=groups.get(v['move'],0)+1
  classes.setdefault(v['move'],set()).add(mate(v['score']))
  if v['move'] not in reps or v['depth']>reps[v['move']][1]['depth']:reps[v['move']]=(i,v)
 best=max(groups.values());winners=[m for m,n in groups.items() if n==best]
 if len(winners)!=1:return False
 move=winners[0]
 return (move!=primary['move'] and best>=2 and best>groups[primary['move']]
         and len(classes[move])==1 and not (min(classes[move])<0 and mate(primary['score'])>=0))
def run_mode(mode,sampling=False,solves=False):
 env=os.environ.copy();env.update(NGN_CAMPAIGN_NET=str(see/'base.nnue'),NGN_S77_PANEL=str(p/'panel.json'),NGN_S77_SAMPLING='1' if sampling else '0',NGN_S77_BLOCKS='1' if sampling else '6',NGN_S77_SOLVES='1' if solves else '',GOMAXPROCS='4')
 pattern='^TestS77HeldOutMateStability$' if solves else '^TestS77WorkerPanel$'
 argv=['taskset','-c','0,2,4,6',str(p/'panel.test'),'-test.run',pattern,'-test.v','-test.timeout','30m']
 status(mode+'-start',{'argv':argv,'sampling':sampling})
 with (p/(mode+'.log')).open('w') as f:result=subprocess.run(argv,cwd=p,env=env,stdout=f,stderr=subprocess.STDOUT)
 assert result.returncode==0,(mode,result.returncode)
 data=rows(mode+'.log');assert len(data)==(288 if not sampling else 48),(mode,len(data))
 status(mode+'-done',{'rows':len(data),'exit':0});return data
def run():
 status('see-wait','native terminal; clean shelf only, winner/fault needs manual actual-base review')
 wait_done(see,'controller.done')
 gate=json.loads((see/'gate-summary.json').read_text()) if (see/'gate-summary.json').exists() else {}
 if (see/'controller.exit').read_text().strip()!='0' or gate.get('verdict')!='shelved':
  status('actual-base-hold',{'see':gate,'action':'manual review and panel rebase; no automatic adoption'});return
 assert (p/'checks.exit').read_text().strip()=='0'
 busy=[line for line in subprocess.check_output(['ps','-eo','args'],text=True).splitlines() if re.search(r'trainer\.train|(?:^|/)sprt(?:\s|$)|(?:^|/)datagen(?:\s|$)|(?:^|/)pairmatch(?:\s|$)',line)]
 assert not busy,busy
 files=['panel.test','panel.json','manifest.md','diagnostic.patch','controller.py']
 write('execution-manifest.json',{'utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'source':'f1d0a03 plus diagnostic-only hooks','net':'a4d90b41','widths':[1,4],'hash_mib':256,'physical_cpus':[0,2,4,6],'background':'SEE terminal; no games/trainer','sha256':{f:hashlib.sha256((p/f).read_bytes()).hexdigest() for f in files}})
 main=run_mode('panel');sample=run_mode('sample',sampling=True);solves=run_mode('solves',solves=True)
 opportunities=[r for r in main if r['width']==4 and opportunity(r)]
 frames=sorted({r['id'] for r in opportunities})
 scaling={}
 for width in [1,4]:
  cells=[r for r in main if r['width']==width]
  wall=sum(r['elapsed_ns'] for r in cells);cpu=sum(r['cpu_ns'] for r in cells);nodes=sum(r['nodes'] for r in cells)
  scaling[width]={'wall_seconds':wall/1e9,'cpu_seconds':cpu/1e9,'nodes':nodes,'nps':nodes/(wall/1e9),'cpu_occupancy':cpu/wall,'setup_fraction':sum(r['setup_ns'] for r in cells)/wall,'join_fraction':sum(r['join_ns'] for r in cells)/wall}
 overlaps=[]
 for row in sample:
  if row['width']!=4:continue
  sets=[{(v['key'],v['depth']) for v in w['visits']} for w in row['workers']]
  pairs=[]
  for i in range(4):
   for j in range(i):
    union=sets[i]|sets[j];pairs.append(len(sets[i]&sets[j])/len(union) if union else 0)
  overlaps.append({'id':row['id'],'budget_ms':row['budget_ms'],'mean_pairwise_jaccard':sum(pairs)/len(pairs),'samples':[len(s) for s in sets]})
 summary={'rows':len(main),'sample_rows':len(sample),'solve_rows':len(solves),'scaling':scaling,'eligible_consensus_cells':len(opportunities),'eligible_frames':frames,'consensus_admission':'PASS' if len(opportunities)>=5 and len(frames)>=2 else 'HOLD','eligible_cells':[{'id':r['id'],'block':r['block'],'budget_ms':r['budget_ms']} for r in opportunities],'mate_misses':[r for r in solves if not r['correct']],'sample_overlap':overlaps,'sample_overflow':sum(w['sample_overflow'] for r in sample for w in r['workers']),'automatic_adoption':False,'elo_claim':False}
 write('panel-summary.json',summary);status('panel-terminal',summary)
if __name__=='__main__':run()
