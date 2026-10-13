"""Transparent rig-only launcher recording actual SMP admission, not game scores."""
import json
import os
from pathlib import Path
import subprocess
import sys
import threading

lane = Path(__file__).resolve().parent
role = sys.argv[1]
os.sched_setaffinity(0, {'new': {0, 2}, 'base': {4, 6}}[role])
environment = dict(os.environ)
environment.pop('GOMAXPROCS', None)  # main.go installs the Threads scheduler.
engine = subprocess.Popen([str(lane / 'ngn')], cwd=lane, env=environment,
    stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, bufsize=1)
trace = (lane / f'{role}-{os.getpid()}.jsonl').open('w')
lock = threading.Lock()


def record(kind, payload):
    with lock:
        trace.write(json.dumps(dict(kind=kind, payload=payload)) + '\n')


record('process', dict(pid=engine.pid, role=role, cpus=sorted(os.sched_getaffinity(engine.pid)),
                       GOMAXPROCS='unset: production UCI Threads scheduler'))


def output():
    for line in engine.stdout:
        if line.startswith(('info string threads ', 'bestmove ', 'info string error ')):
            record('output', line.strip())
        sys.stdout.write(line)
        sys.stdout.flush()


def errors():
    for line in engine.stderr:
        record('stderr', line.strip())


readers = [threading.Thread(target=output), threading.Thread(target=errors)]
for reader in readers:
    reader.start()
try:
    for line in sys.stdin:
        if line.startswith(('setoption name Threads ', 'go ', 'stop', 'quit')):
            record('input', line.strip())
        engine.stdin.write(line)
        engine.stdin.flush()
        if line.strip() == 'quit':
            break
finally:
    try:
        engine.stdin.close()
    except BrokenPipeError:
        pass
    try:
        code = engine.wait(timeout=15)
    except subprocess.TimeoutExpired:
        engine.kill()
        code = engine.wait()
    for reader in readers:
        reader.join()
    record('exit', code)
    trace.close()
sys.exit(code)
