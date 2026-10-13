"""Rig-only, fixed-depth Threads1 move/score/node/PV identity, not speed."""
import json,os,pathlib,queue,re,subprocess,threading
p=pathlib.Path('/home/ehrli/ngn-data/night-20261007/session77-consensus')
base=p.parent/'session77-see-rig/baseline-ngn';net=p.parent/'session77-see-rig/base.nnue'
frames=json.loads((p.parent/'session77-smp-panel/panel.json').read_text())
class UCI:
 def __init__(self,binary):
  self.q=queue.Queue();self.child=subprocess.Popen(['taskset','-c','13,15',str(binary)],cwd=p,env=dict(os.environ,GOMAXPROCS='1'),stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True,bufsize=1)
  self.errors=[]
  def read():
   for line in self.child.stdout:self.q.put(line.strip())
   self.q.put(None)
  def errors():
   for line in self.child.stderr:self.errors.append(line.strip())
  self.read=threading.Thread(target=read);self.err=threading.Thread(target=errors);self.read.start();self.err.start()
  self.send('uci');self.until('uciok')
  for option in [f'EvalFile value {net}','UseNNUE value true','Hash value 256','Threads value 1','OwnBook value false']:self.send('setoption name '+option)
  self.send('isready');self.until('readyok')
 def send(self,text):self.child.stdin.write(text+'\n');self.child.stdin.flush()
 def until(self,prefix):
  lines=[]
  while True:
   line=self.q.get(timeout=120);assert line is not None;lines.append(line)
   if line.startswith(prefix):return lines
 def search(self,row):
  self.send('ucinewgame');self.send('position startpos moves '+' '.join(row['moves']));self.send('isready');self.until('readyok');self.send('go depth 14');lines=self.until('bestmove ')
  info=[line for line in lines if line.startswith('info depth ')][-1]
  return {'depth':int(re.search(r'\bdepth (\d+)',info)[1]),'nodes':int(re.search(r'\bnodes (\d+)',info)[1]),'score':re.search(r'\bscore (cp|mate) (-?\d+)',info).group(0),'pv':info.split(' pv ',1)[1] if ' pv ' in info else '', 'bestmove':lines[-1].split()[1]}
 def close(self):
  self.send('quit');assert self.child.wait(timeout=15)==0;self.read.join();self.err.join();assert all('Crash Handler Initialized' in line or 'Crash log:' in line for line in self.errors),self.errors
engines=[UCI(base),UCI(p/'candidate-ngn')];records=[]
try:
 for repeat in range(3):
  for row in frames:
   results=[e.search(row) for e in engines];assert results[0]==results[1],(row['id'],repeat,results)
   records.append({'id':row['id'],'repeat':repeat,'identity':results[0]})
finally:
 for e in engines:e.close()
(p/'identity.json').write_text(json.dumps({'pairs':len(records),'source':'only SMP coordinator/selector changed','width':1,'hash_mib':256,'depth_limit':14,'verdict':'PASS','records':records},indent=2)+'\n')
