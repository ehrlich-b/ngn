import datetime, hashlib, json, os, queue, re, subprocess, threading
from pathlib import Path
p=Path.cwd()
vector=json.loads((p/'vector.json').read_text())
old=json.loads((p/'old-live.json').read_text())
old_defaults={r['name']:r['default'] for r in old['parameters']}
positions=[r['fen'] for r in old['witnesses']]
assert len(positions)==6
def probe(name, binary, overrides):
    q = queue.Queue()
    lines, commands, results = [], [], []
    with (p / (name + '.stderr')).open('w') as err:
        process = subprocess.Popen(['taskset', '-c', '15', 'nice', '-n', '19', str(p/binary)],
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
            advertised = {name: int(value) for name, value in re.findall(r'^option name (.+?) type spin default (-?\d+)', '\n'.join(lines), re.M)}
            for option_name, value in vector.items():
                expected = value if binary == 'candidate-ngn' else old_defaults[option_name]
                assert advertised[option_name] == expected, (binary, option_name, advertised[option_name], expected)
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

baseline=probe('current-overrides','baseline-ngn',vector)
candidate=probe('compiled-defaults','candidate-ngn',{})
assert baseline==candidate, 'compiled default/override behavior differs'
assert candidate==[w['tuned'] for w in old['witnesses']], 'current exact-speed source differs from tested tune'
result=dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),searches=12,positions=6,depth=14,node_limit=1000000,compiled_defaults_equal_current_overrides=True,compiled_defaults_equal_accepted_old_source_tune=True,advertised_defaults_match_all14=True,fields=['depth','nodes','score_kind','score','pv','bestmove'],normal_exits=True,base_source='14f7635 (engine7bbb2bf)',candidate='final14 compiled defaults only')
(p/'default-parity.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result))
