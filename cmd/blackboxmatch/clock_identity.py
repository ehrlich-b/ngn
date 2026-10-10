"""One bounded actual-UCI node-identity check for the diagnostic builds."""
import json
import queue
import subprocess
import sys
import threading
from pathlib import Path

positions=[s for s in Path(sys.argv[4]).read_text().splitlines() if s and not s.startswith('#')][:12]
net=Path(sys.argv[5]).resolve()
results={}
for name,binary in zip(['accepted','normal','clock'],sys.argv[1:4]):
    error_path=Path(name+'-identity.stderr')
    with error_path.open('w') as err:
        p=subprocess.Popen([str(Path(binary).resolve())],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=err,text=True,bufsize=1)
        lines=queue.Queue()
        def reader():
            for line in p.stdout:lines.put(line.strip())
            lines.put(None)
        thread=threading.Thread(target=reader,daemon=True);thread.start()
        def send(command):
            p.stdin.write(command+'\n');p.stdin.flush()
        def until(prefix):
            rows=[]
            while True:
                line=lines.get(timeout=20)
                assert line is not None,'engine exited early'
                assert not line.startswith('info string error'),line
                rows.append(line)
                if line.startswith(prefix):return rows
        try:
            send('uci');until('uciok')
            for key,value in [('EvalFile',str(net)),('UseNNUE','true'),('Hash','64'),('Threads','1'),('OwnBook','false')]:send(f'setoption name {key} value {value}')
            send('isready');until('readyok')
            cases=[]
            for fen in positions:
                send('ucinewgame');send('isready');until('readyok');send('position fen '+fen);send('go nodes 50000')
                rows=until('bestmove ')
                infos=[];observations=[]
                for row in rows:
                    if row.startswith('info string ngn_clock '):observations.append(json.loads(row.removeprefix('info string ngn_clock ')))
                    elif row.startswith('info depth '):
                        fields=row.split();clean=[];i=0
                        while i<len(fields):
                            if fields[i] in ['time','nps']:i+=2
                            else:clean.append(fields[i]);i+=1
                        infos.append(' '.join(clean))
                if name=='clock':
                    assert len(observations)==1 and observations[0]['actual_nodes']==50000 and observations[0]['first_reason']=='node-cap',observations
                else:assert not observations
                cases.append(dict(fen=fen,info=infos,bestmove=rows[-1]))
            results[name]=cases
        finally:
            if p.poll() is None:
                send('quit')
                try:p.wait(timeout=2)
                except subprocess.TimeoutExpired:p.kill();p.wait();raise
            thread.join(timeout=2)
        assert p.returncode==0
        assert not error_path.stat().st_size
assert results['accepted']==results['normal']==results['clock']
Path('clock-identity.json').write_text(json.dumps(dict(cases=12,node_limit=50000,normal_matches_accepted=True,clock_matches_normal=True,observer_reports_actual_nodes=True,identity=results['clock']),indent=2)+'\n')
print('12 actual-UCI fixed-node identities: accepted = normal = clock; all observer node counts exact')
