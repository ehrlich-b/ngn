#!/usr/bin/env python3
"""Run the pinned wide SPSA campaign with bounded, lock-sharing chunks.

The user manager starts this controller at nice 0. Builds and repository checks
run separately at nice 10. The manifest and both binaries are immutable inputs;
all campaign outputs live under the manifest's dedicated runtime directory.
"""
import datetime
import fcntl
import hashlib
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import time

SOURCE = '6d3168b8c5f15532cc7a352d05465a77ab50f361'
ROOT = Path('/home/ehrli/ngn-data/night-20261007/burn-1104/spsa')
DATA = ROOT.parents[1]
NET = DATA / 'burn-1104/h768/reexport.nnue'
BOOK = DATA / 'session79-admission-see/openings.txt'
LOCK = DATA / 'session138/cpu0-11.lock'
NET_SHA = '2dcb6d936c0275927cb9e881f42e22558e618ad7e6fd998df6c4e5046d42c443'
BOOK_SHA = 'f2fed55a1645dba85a9ef3e047eb5183b5b6342878c6dc8483e6768973470acc'
OPTIONS = [f'EvalFile={NET}', 'UseNNUE=true', 'Threads=1', 'Hash=64', 'OwnBook=false']
TOTAL, CHUNK, REST, CAP = 1200, 100, 60, 86400


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def write_new(path, value):
    with Path(path).open('x') as stream:
        json.dump(value, stream, indent=2, allow_nan=False)
        stream.write('\n')
        stream.flush()
        os.fsync(stream.fileno())


def read_rows(path):
    data = Path(path).read_bytes()
    if not data or not data.endswith(b'\n'):
        raise RuntimeError(f'incomplete history: {path}')
    rows = [json.loads(line) for line in data.splitlines()]
    for i, row in enumerate(rows, 1):
        if row['version'] != 2 or row['iter'] != i:
            raise RuntimeError(f'nonconsecutive history: {path}, row {i}')
        if row['recipe'] != rows[0]['recipe']:
            raise RuntimeError(f'changed recipe: {path}, row {i}')
    return rows


def argv(root, history, count, resume):
    args = ['taskset', '-c', '0-11', str(root / 'spsa'),
            '-engine', str(root / 'ngn'), '-tc', '10+0.1', '-pairs', '6',
            '-iters', str(TOTAL), '-chunk-iters', str(count), '-report', '1',
            '-seed', '20261104', '-rend', '.05', '-afrac', '.1',
            '-lowpower=false', '-openings', str(BOOK), '-out', str(history)]
    for option in OPTIONS:
        args.extend(['-engineoption', option])
    if resume:
        args.append('-resume')
    return args


def clean_env():
    return {'HOME': '/home/ehrli', 'PATH': '/usr/local/go/bin:/usr/bin:/bin',
            'GOMAXPROCS': '2', 'GOAMD64': 'v3', 'LANG': 'C.UTF-8',
            'TMPDIR': str(ROOT / 'tmp')}


class BudgetExpired(Exception):
    pass


class Campaign:
    def __init__(self, manifest):
        self.manifest = manifest
        self.root = ROOT
        self.start = time.monotonic()
        # Leave five seconds for process cleanup and the final synced vector.
        self.deadline = self.start + CAP - 5
        self.stopping = None
        self.last = None

    def event(self, event, **detail):
        row = dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                   campaign='ngn-1104-spsa', event=event, **detail)
        line = json.dumps(row, allow_nan=False) + '\n'
        for path in [self.root / 'events.jsonl', DATA / 'status.jsonl']:
            with path.open('a') as stream:
                stream.write(line)
                stream.flush()
                os.fsync(stream.fileno())
        print(line, end='', flush=True)

    def check_budget(self):
        if self.stopping:
            raise RuntimeError(f'signal {self.stopping}')
        if time.monotonic() >= self.deadline:
            raise BudgetExpired('24-hour wall-clock cap, including proof, lock waits and rests')

    def verify_inputs(self):
        if self.manifest['source'] != SOURCE:
            raise RuntimeError('wrong pinned engine source')
        expected = {str(NET): NET_SHA, str(BOOK): BOOK_SHA,
                    str(self.root / 'ngn'): self.manifest['engine_sha256'],
                    str(self.root / 'spsa'): self.manifest['tuner_sha256'],
                    str(self.root / 'controller.py'): self.manifest['controller_sha256']}
        for path, sha in expected.items():
            if digest(path) != sha:
                raise RuntimeError(f'input hash changed: {path}')

    def locked(self, action):
        # The shared lock already exists. Never create, truncate or replace it.
        with LOCK.open('r') as lock:
            self.event('WAIT_LOCK', path=str(LOCK))
            while True:
                self.check_budget()
                try:
                    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                    break
                except BlockingIOError:
                    time.sleep(.5)
            self.event('ACQUIRE_LOCK')
            try:
                return action()
            finally:
                fcntl.flock(lock, fcntl.LOCK_UN)
                self.event('RELEASE_LOCK')

    def run_chunk(self, history, count, resume, label):
        self.check_budget()
        self.verify_inputs()
        previous = read_rows(history)[-1]['iter'] if resume else 0
        args = argv(self.root, history, count, resume)
        logpath = self.root / f'{label}.log'
        self.event('CHUNK_START', label=label, previous=previous, count=count, argv=args)
        began = time.monotonic()
        with logpath.open('x') as log:
            proc = subprocess.Popen(args, env=clean_env(), stdout=log,
                                    stderr=subprocess.STDOUT, start_new_session=True)
            self.event('TUNER_PID', label=label, pid=proc.pid)
            try:
                while proc.poll() is None:
                    self.check_budget()
                    time.sleep(.25)
            finally:
                # Kill the entire child session, including engines after faults.
                for sig in (signal.SIGTERM, signal.SIGKILL):
                    try:
                        os.killpg(proc.pid, sig)
                    except ProcessLookupError:
                        pass
                    if sig == signal.SIGTERM:
                        try:
                            proc.wait(timeout=2)
                        except subprocess.TimeoutExpired:
                            pass
                proc.wait()
        if proc.returncode != 0:
            raise RuntimeError(f'{label}: tuner exit {proc.returncode}; {logpath}: '
                               + logpath.read_text()[-1500:])
        rows = read_rows(history)
        endpoint = rows[-1]
        if endpoint['iter'] != min(previous + count, TOTAL):
            raise RuntimeError(f'{label}: wrong endpoint {endpoint["iter"]}')
        recipe = endpoint['recipe']
        if (recipe['iterations'], recipe['pairs'], recipe['single'], recipe['time_ms'],
                recipe['inc_ms'], recipe['report'], recipe['gomaxprocs']) != (TOTAL, 6, False, 10000, 100, 1, 2):
            raise RuntimeError('incorrect campaign recipe')
        if recipe['engine_sha256'] != self.manifest['engine_sha256'] or recipe['tuner_sha256'] != self.manifest['tuner_sha256']:
            raise RuntimeError('checkpoint binary pins differ')
        if recipe['engine_options'] != ['setoption name ' + s.replace('=', ' value ', 1) for s in OPTIONS]:
            raise RuntimeError('incorrect fixed engine options')
        if resume and f'resumed exact recipe from iter {previous} ' not in logpath.read_text():
            raise RuntimeError('missing exact resume confirmation')
        elapsed = time.monotonic() - began
        self.event('CHUNK_DONE', label=label, iteration=endpoint['iter'],
                   games=(endpoint['iter'] - previous) * 12, elapsed_seconds=elapsed,
                   games_per_hour=(endpoint['iter'] - previous) * 12 * 3600 / elapsed)
        return endpoint

    def rest(self):
        self.event('REST', seconds=REST)
        end = time.monotonic() + REST
        while time.monotonic() < end:
            self.check_budget()
            time.sleep(min(.5, end - time.monotonic()))

    def execute(self):
        state, reason = 'FAULT', 'startup incomplete'
        try:
            if socket.gethostname() != 'NewPC' or os.getuid() == 0:
                raise RuntimeError('requires unprivileged NewPC user')
            if os.getpriority(os.PRIO_PROCESS, 0) != 0:
                raise RuntimeError('controller and engines must inherit nice 0')
            stat = os.statvfs('/mnt/c')
            free = stat.f_bavail * stat.f_frsize
            if free < 25 * 2**30:
                raise RuntimeError(f'C: free bytes {free} below 25 GiB; no games started')
            self.verify_inputs()
            write_new(self.root / 'start.json', dict(pid=os.getpid(), free_bytes=free,
                      started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                      cap_seconds=CAP, total=TOTAL, chunk=CHUNK, rest=REST))
            self.event('START', pid=os.getpid(), free_bytes=free, cap_seconds=CAP)
            proof = self.root / 'resume-proof.jsonl'
            stopped = self.locked(lambda: self.run_chunk(proof, 2, False, 'proof-stop2'))
            write_new(self.root / 'proof-stop2.json', stopped)
            resumed = self.locked(lambda: self.run_chunk(proof, 1, True, 'proof-resume3'))
            if read_rows(proof)[1] != stopped or resumed['rng_draws'] <= stopped['rng_draws']:
                raise RuntimeError('resume changed prefix or failed to advance RNG')
            write_new(self.root / 'resume-proof.json', dict(stopped=stopped['iter'],
                      resumed=resumed['iter'], recipe_unchanged=stopped['recipe'] == resumed['recipe'],
                      prefix_unchanged=True, before_draws=stopped['rng_draws'], after_draws=resumed['rng_draws']))
            self.event('RESUME_PROVED', stopped=2, resumed=3)
            history = self.root / 'spsa.jsonl'
            for first in range(1, TOTAL + 1, CHUNK):
                self.last = self.locked(lambda: self.run_chunk(history, CHUNK, first > 1,
                                       f'chunk-{first:04d}-{min(first + CHUNK - 1, TOTAL):04d}'))
                write_new(self.root / f'vector-{self.last["iter"]:04d}.json', self.last)
                if self.last['iter'] < TOTAL:
                    self.rest()
            state, reason = 'COMPLETE', '1200 iterations complete; vector requires a separate strength gate'
        except BudgetExpired as exc:
            state, reason = 'CAPPED', str(exc)
        except BaseException as exc:
            state, reason = 'FAULT', f'{type(exc).__name__}: {exc}'
        finally:
            history = self.root / 'spsa.jsonl'
            if history.exists() and history.stat().st_size:
                try:
                    self.last = read_rows(history)[-1]
                except BaseException as exc:
                    state, reason = 'FAULT', reason + f'; invalid history: {exc}'
            write_new(self.root / 'final-vector.json', dict(state=state, reason=reason,
                      elapsed_seconds=time.monotonic() - self.start, checkpoint=self.last,
                      automatic_adoption=False))
            self.event(state, reason=reason, iteration=self.last['iter'] if self.last else 0)
        return 1 if state == 'FAULT' else 0


def main():
    manifest = json.loads(Path(sys.argv[1]).read_text())
    campaign = Campaign(manifest)
    for sig in (signal.SIGTERM, signal.SIGINT):
        signal.signal(sig, lambda signum, frame: setattr(campaign, 'stopping', signum))
    return campaign.execute()


if __name__ == '__main__':
    sys.exit(main())
