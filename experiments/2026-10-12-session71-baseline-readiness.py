"""Check the archived match reference on the same owned net and final14."""
import datetime
import hashlib
import json
import os
from pathlib import Path
import queue
import re
import subprocess
import threading

p = Path.cwd()
vector = json.loads((p / 'vector.json').read_text())
positions = [
    'r1bq1rk1/pp2bppp/2n2n2/2pp4/3P4/2N1PN2/PPQ1BPPP/R1B2RK1 w - - 0 10',
    'r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1',
    'r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10',
    '8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1',
    '8/5k2/2p2p2/3p4/3P1P2/2P2K2/8/8 w - - 0 1',
    '6k1/5pp1/7p/3q4/8/3Q2P1/5P1P/6K1 w - - 0 1',
]


def probe(name, binary, overrides, searches):
    q = queue.Queue()
    lines, commands, results, evaluations = [], [], [], []
    with (p / (name + '.stderr')).open('w') as err:
        proc = subprocess.Popen(['taskset', '-c', '8', str(p / binary)],
            env=dict(os.environ, GOMAXPROCS='1'), stdin=subprocess.PIPE,
            stdout=subprocess.PIPE, stderr=err, text=True, bufsize=1)
        def drain():
            for line in proc.stdout:
                q.put(line.rstrip('\n'))
            q.put(None)
        reader = threading.Thread(target=drain, daemon=True)
        reader.start()
        def send(line):
            commands.append(line)
            proc.stdin.write(line + '\n')
            proc.stdin.flush()
        def until(prefix):
            while True:
                line = q.get(timeout=30)
                assert line is not None, 'premature exit'
                lines.append(line)
                assert not re.search(r'error|failed|unknown option|panic|fatal|illegal', line, re.I), line
                if line.startswith(prefix):
                    return line
        try:
            send('uci')
            until('uciok')
            advertised = {name: (int(lo), int(hi)) for name, lo, hi in
                re.findall(r'^option name (.+?) type spin default -?\d+ min (-?\d+) max (-?\d+)',
                           '\n'.join(lines), re.M)}
            assert len(vector) == 14
            for key, value in vector.items():
                assert key in advertised and advertised[key][0] <= value <= advertised[key][1]
            for key, value in dict(EvalFile=str(p / 'base.nnue'), UseNNUE='true',
                                   Hash=64, Threads=1, OwnBook='false', **overrides).items():
                send(f'setoption name {key} value {value}')
            send('isready')
            until('readyok')
            for fen in positions:
                send('ucinewgame')
                send('isready')
                until('readyok')
                send('position fen ' + fen)
                send('eval')
                evaluation = until('info string eval backend ')
                match = re.search(r'backend (\S+) score_cp (-?\d+)', evaluation)
                assert match and match[1] == 'ngnn3', evaluation
                evaluations.append(int(match[2]))
                if searches:
                    start = len(lines)
                    send('go depth 14 nodes 1000000')
                    best = until('bestmove ').split()[1]
                    final = [line for line in lines[start:] if line.startswith('info depth ')][-1].split()
                    result = dict(depth=int(final[final.index('depth') + 1]),
                                  nodes=int(final[final.index('nodes') + 1]),
                                  score_kind=final[final.index('score') + 1],
                                  score=int(final[final.index('score') + 2]),
                                  bestmove=best, pv=final[final.index('pv') + 1:])
                    assert result['nodes'] > 0 and result['pv'][0] == best and best not in ('0000', '(none)')
                    results.append(result)
            send('quit')
            proc.wait(timeout=10)
            reader.join(timeout=5)
            assert proc.returncode == 0 and not reader.is_alive()
        finally:
            if proc.poll() is None:
                proc.kill()
                proc.wait(timeout=10)
            (p / (name + '.json')).write_text(json.dumps(dict(
                commands=commands, stdout=lines, returncode=proc.returncode,
                overrides=overrides, evaluations=evaluations, results=results), indent=2) + '\n')
    assert (p / (name + '.stderr')).stat().st_size == 0
    return evaluations, results


defaults_eval, defaults = probe('archived-defaults', 'baseline-ngn', {}, True)
tuned_eval, tuned = probe('archived-final14', 'baseline-ngn', vector, True)
current_eval, _ = probe('accepted-eval', 'candidate-ngn', vector, False)
assert defaults_eval == tuned_eval == current_eval
assert defaults != tuned, 'no final-vector behavioral witness'
receipt = dict(
    utc=datetime.datetime.now(datetime.timezone.utc).isoformat(), ready=True,
    baseline_source='5b882e2', candidate_source='f1d0a03', searches=12,
    positions=6, evaluation_triplets_equal=True, backend='ngnn3',
    vector=vector, all14_advertised_in_range=True,
    handler='archived and accepted original setTunableParam source; same registry names and limits',
    source_imported_or_rebuilt=False, current_source_checks_reused=True,
    changed_search_positions=[i for i, (a, b) in enumerate(zip(defaults, tuned)) if a != b],
    files_sha256={name: hashlib.sha256((p / name).read_bytes()).hexdigest()
                  for name in ['baseline-ngn', 'candidate-ngn', 'base.nnue', 'vector.json']})
(p / 'baseline-readiness.json').write_text(json.dumps(receipt, indent=2) + '\n')
print(json.dumps(receipt))
