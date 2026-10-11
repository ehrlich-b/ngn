"""Verify the frozen endpoint on the current accepted package; no games."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import queue
import re
import subprocess
import sys
import threading

p = Path(sys.argv[1]).resolve()
source = p / 'source'
vector = json.loads((p / 'vector.json').read_text())
text = (source / 'engine/search.go').read_text()
registry = re.findall(r'\{"(\w+)", &(\w+), (\d+), (\d+), (\d+)\}', text)
assert len(registry) == len(vector) == 14
params = []
for name, var, default, minimum, maximum in registry:
    assert int(minimum) <= vector[name] <= int(maximum)
    reads = [i for i, line in enumerate(text.splitlines(), 1)
             if re.search(r'\b' + var + r'\b', line.split('//')[0])
             and not re.search(r'\b' + var + r'\s*=', line.split('//')[0])
             and not line.lstrip().startswith('{')]
    assert reads
    params.append(dict(name=name, variable=var, default=int(default), value=vector[name], read_lines=reads))

positions = [
    'r1bq1rk1/pp2bppp/2n2n2/2pp4/3P4/2N1PN2/PPQ1BPPP/R1B2RK1 w - - 0 10',
    'r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1',
    'r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10',
    '8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1',
    '8/5k2/2p2p2/3p4/3P1P2/2P2K2/8/8 w - - 0 1',
    '6k1/5pp1/7p/3q4/8/3Q2P1/5P1P/6K1 w - - 0 1',
]


def probe(name, overrides):
    q = queue.Queue()
    lines, commands, results = [], [], []
    with (p / (name + '.stderr')).open('w') as err:
        process = subprocess.Popen(['taskset', '-c', '8', 'nice', '-n', '10', str(p/'ngn')],
            cwd=p, env=dict(os.environ, GOMAXPROCS='1'), stdin=subprocess.PIPE,
            stdout=subprocess.PIPE, stderr=err, text=True, bufsize=1)
        def drain():
            for line in process.stdout:
                q.put(line.rstrip('\n'))
            q.put(None)
        thread = threading.Thread(target=drain, daemon=True)
        thread.start()
        def send(line):
            commands.append(line)
            process.stdin.write(line+'\n'); process.stdin.flush()
        def until(prefix):
            while True:
                line = q.get(timeout=30)
                assert line is not None, 'premature exit'
                lines.append(line)
                assert not re.search(r'error|failed|unknown option|panic|fatal|illegal', line, re.I), line
                if line.startswith(prefix): return line
        try:
            send('uci'); until('uciok')
            for option, _, default, minimum, maximum in registry:
                assert f'option name {option} type spin default {default} min {minimum} max {maximum}' in lines
            for option, value in {'EvalFile':str(p/'base.nnue'),'UseNNUE':'true','Hash':'64','Threads':'1','OwnBook':'false',**overrides}.items():
                send(f'setoption name {option} value {value}')
            send('isready'); until('readyok')
            for fen in positions:
                send('ucinewgame'); send('isready'); until('readyok')
                send('position fen '+fen)
                start = len(lines)
                send('go depth 14 nodes 1000000')
                best = until('bestmove ').split()[1]
                final = [l for l in lines[start:] if l.startswith('info depth ')][-1].split()
                def value(field): return final[final.index(field)+1]
                result = dict(depth=int(value('depth')),nodes=int(value('nodes')),
                    score_kind=value('score'),score=int(final[final.index('score')+2]),
                    pv=final[final.index('pv')+1:],bestmove=best)
                assert result['nodes']>0 and result['pv'][0]==best and best not in ('0000','(none)')
                results.append(result)
            send('quit'); process.wait(timeout=10); thread.join(timeout=5)
            assert process.returncode==0 and not thread.is_alive()
        finally:
            if process.poll() is None:
                process.kill(); process.wait(timeout=10)
            (p/(name+'.json')).write_text(json.dumps(dict(commands=commands,stdout=lines,
                returncode=process.returncode,overrides=overrides,results=results),indent=2)+'\n')
    assert (p/(name+'.stderr')).stat().st_size==0
    return results

base = probe('defaults', {})
tuned = probe('final-vector', vector)
witnesses = [dict(position=i,fen=positions[i],base=b,tuned=t) for i,(b,t) in enumerate(zip(base,tuned)) if b!=t]
assert witnesses
sha = lambda f: hashlib.sha256((p/f).read_bytes()).hexdigest()
receipt = dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),source='ed34480',
    base_net_sha256=sha('base.nnue'),binary_sha256=sha('ngn'),endpoint=json.loads((p/'endpoint.json').read_text()),
    machine='forge Debian13/Ryzen6900HX',host_pinning='on',cpus='8-11',nice=10,memory_max_bytes=2*1024**3,
    parameters=params,handler_test='PASS all14 pointers match exact final values',searches=12,
    depth_cap=14,node_cap=1000000,witnesses=witnesses,normal_exits=True,
    verdict='live-vector prerequisite PASS; manual nomination and controlled games still required',
    files_sha256={n:sha(n) for n in ['vector.json','endpoint.json','ngn','base.nnue','handler.log','defaults.json','final-vector.json']})
(p/'live-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
print(json.dumps(dict(searches=12,changed_positions=len(witnesses),verdict=receipt['verdict'])))
