import datetime,hashlib,json,math,os,select,statistics,subprocess
from pathlib import Path
R=Path('/home/ehrli/ngn-data/night-20261007');D=R/'session19-arch-cost';T=R/'session19-time'
if not (T/'controller.done').exists():
    fd=os.pidfd_open(int((T/'controller.pid').read_text()));p=select.poll();p.register(fd,select.POLLIN)
    assert p.poll(3*3600*1000),'time-test dependency timeout';os.close(fd)
assert (T/'controller.done').exists()
S=R/'session17f';binary=S/'candidate-probe';nets={'ngnn3':R/'kb8h512/best.nnue','ngnn4':S/'random-h512-l16.nnue'}
env=dict(os.environ,GOMAXPROCS='1');commands=[]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
pins={'binary':sha(binary),**{k:sha(v) for k,v in nets.items()}}
def run(kind,name):
    argv=['taskset','-c','13','nice','-n','19',str(binary),'-net',str(nets[kind]),'-mode','timed','-duration','3s'];commands.append(argv)
    with (D/f'{name}.jsonl').open('w') as out,(D/f'{name}.err').open('w') as err:subprocess.run(argv,env=env,stdout=out,stderr=err,check=True,timeout=180)
    rows=[json.loads(line) for line in (D/f'{name}.jsonl').read_text().splitlines()]
    assert len(rows)==6 and all(r['nodes']>0 and 2.94<=r['seconds']<=5 for r in rows)
    assert not (D/f'{name}.err').stat().st_size
    return sum(r['nodes'] for r in rows)/sum(r['seconds'] for r in rows)
for kind in nets:run(kind,'warm-'+kind)
rounds=[]
with (D/'rounds.jsonl').open('w') as f:
    for i in range(12):
        order=['ngnn3','ngnn4'] if i%2==0 else ['ngnn4','ngnn3']
        nps={k:run(k,f'{i:02d}-{k}') for k in order}
        row={'index':i,'order':order,'nps':nps,'ratio':nps['ngnn4']/nps['ngnn3']};rounds.append(row);f.write(json.dumps(row)+'\n');f.flush()
logs=[math.log(r['ratio']) for r in rounds];mean=statistics.mean(logs);margin=2.200985160082949*statistics.stdev(logs)/math.sqrt(12)
ratio=math.exp(mean);ci=[math.exp(mean-margin),math.exp(mean+margin)];strata=[math.exp(statistics.mean(logs[i::2])) for i in range(2)]
x={'completed_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'source':'14a203a','pins':pins,'width':512,'ngnn4_hidden2':16,'ratio':ratio,'ci95':ci,'order_strata_ratios':strata,'budget_rule':'geometric ratio >=0.85','gpu_architecture_admitted':ratio>=.85,'elo_verdict':None,'limits':'seeded random NGNN4 fixture versus accepted NGNN3; tree mix differs; no trained-net strength evidence','commands':commands,'rounds':rounds}
(D/'receipt.json').write_text(json.dumps(x,indent=2)+'\n')
with (R/'status.jsonl').open('a') as f:f.write(json.dumps({'utc':x['completed_utc'],'stage':'ngnn4-cost','event':'done','detail':f'NGNN4/NGNN3 H512 NPS {ratio:.6f} [{ci[0]:.6f},{ci[1]:.6f}]; architecture budget only, no Elo verdict'})+'\n')
print(json.dumps({k:v for k,v in x.items() if k not in ('commands','rounds')}))
