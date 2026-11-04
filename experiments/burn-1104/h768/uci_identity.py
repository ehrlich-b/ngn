#!/usr/bin/env python3
"""Zero-game UCI depth-10 identity of the embedded and explicit EvalFile net."""
import json, os, re, selectors, subprocess, sys, time
from pathlib import Path

def run(binary, net, fen):
    p=subprocess.Popen([binary],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,bufsize=1,env=dict(os.environ,GOMAXPROCS='1'))
    transcript=[]
    def send(s):
        transcript.append('> '+s); p.stdin.write(s+'\n'); p.stdin.flush()
    def until(prefix):
        rows=[]
        # Each proof subprocess has an outer timeout in the runner.
        while True:
            line=p.stdout.readline()
            if not line: raise RuntimeError('UCI process ended: '+repr(transcript))
            line=line.strip(); transcript.append(line); rows.append(line)
            if line.startswith(prefix): return rows
    try:
        send('uci'); hello=until('uciok')
        assert 'option name EvalFile type string default <embedded>' in hello
        assert 'option name UseNNUE type check default true' in hello
        send('setoption name Hash value 16'); send('setoption name Threads value 1')
        if net: send('setoption name EvalFile value '+net)
        send('isready'); ready=until('readyok')
        assert not any('error' in line.lower() for line in ready), ready
        send('eval'); diagnostic=until('info string eval backend')
        assert any('eval backend ngnn3 ' in line for line in diagnostic), diagnostic
        send('position fen '+fen); send('go depth 10'); rows=until('bestmove ')
        infos=[line for line in rows if line.startswith('info depth 10 ')]
        assert infos, rows
        info=infos[-1]; fields=info.split()
        score_index=fields.index('score'); pv_index=fields.index('pv')
        result=dict(depth=int(fields[fields.index('depth')+1]),nodes=int(fields[fields.index('nodes')+1]),score=fields[score_index+1:score_index+3],pv=fields[pv_index+1:],best=rows[-1].split()[1])
        assert result['nodes']>0
        return result,transcript
    finally:
        if p.poll() is None:
            send('quit')
            try: p.wait(timeout=10)
            except subprocess.TimeoutExpired: p.kill(); p.wait()
        p.stdin.close(); p.stdout.close()

def main():
    binary,net,out=sys.argv[1:]
    text=Path('cmd/speedprobe/main.go').read_text().split('var positions = []string{',1)[1].split('}',1)[0]
    fens=re.findall(r'"([^"]+)"',text); assert len(fens)==6
    rows=[]
    for fen in fens:
        a,ta=run(binary,'',fen); b,tb=run(binary,net,fen)
        print(json.dumps(dict(fen=fen,embedded=a,evalfile=b)),flush=True)
        assert a==b,(fen,a,b,ta,tb)
        rows.append(dict(fen=fen,identity=a))
    Path(out).write_text(json.dumps(dict(equal=True,depth=10,evalfile=net,cases=rows),indent=2)+'\n')
    print('PASS: six exact UCI embedded / EvalFile identities; no games')

if __name__=='__main__': main()
