"""Rig-only bounded speed diagnosis through NGN's real UCI launcher.

Run only at an exclusive natural boundary. No engine changes, games, or Elo
verdict. Normal executable for timing; separate diagnostic binary for pprof.
"""
import argparse
import ctypes
import datetime
import fcntl
import hashlib
import json
import os
from pathlib import Path
import queue
import re
import subprocess
import struct
import threading
import time

R = Path('/home/ehrli/ngn-data/night-20261007')
S = R / 'session100-speed'
NET = R / 'session82/net-final115/candidate.nnue'
PHYSICAL = [0, 2, 4, 6, 8, 10, 12, 14]


def write(path, value):
    path.write_text(json.dumps(value, indent=2, allow_nan=False) + '\n')


def cpu_times():
    return {row[0]: [int(v) for v in row[1:9]] for line in Path('/proc/stat').read_text().splitlines()
            if re.match(r'^cpu\d+ ', line) for row in [line.split()]}


class Counters:
    """Linux/amd64 per-thread counters, inherited by later engine threads.

    Count user-space work only. Report unsupported events instead of changing
    privileges; enabled/running times scale any multiplexed event.
    """
    def __init__(self, pid):
        self.fds, self.errors = [], {}
        libc = ctypes.CDLL(None, use_errno=True)
        for name, kind, config in [('cycles',0,0),('instructions',0,1),
                ('cache_references',0,2),('cache_misses',0,3),('dtlb_read_misses',3,0x10003)]:
            for task in Path(f'/proc/{pid}/task').iterdir():
                attr = ctypes.create_string_buffer(128)
                struct.pack_into('IIQ',attr,0,kind,128,config)
                struct.pack_into('Q',attr,32,3)  # enabled/running time in reads
                struct.pack_into('Q',attr,40,1|2|(1<<5)|(1<<6))  # disabled,inherit,exclude kernel/hypervisor
                fd = libc.syscall(ctypes.c_long(298),ctypes.byref(attr),ctypes.c_int(int(task.name)),
                    ctypes.c_int(-1),ctypes.c_int(-1),ctypes.c_ulong(8))
                if fd < 0:
                    self.errors[name] = os.strerror(ctypes.get_errno())
                else:
                    self.fds.append((name,fd))

    def start(self):
        self.before_times = {}
        for _,fd in self.fds:
            fcntl.ioctl(fd,0x2403,0)  # RESET
            _,enabled,running = struct.unpack('QQQ',os.read(fd,24))
            self.before_times[fd] = (enabled,running)
            fcntl.ioctl(fd,0x2400,0)  # ENABLE

    def stop(self):
        counts = {}
        for name,fd in self.fds:
            fcntl.ioctl(fd,0x2401,0)  # DISABLE
            value,enabled,running = struct.unpack('QQQ',os.read(fd,24))
            before_enabled,before_running = self.before_times[fd]
            enabled -= before_enabled
            running -= before_running
            if running:
                counts[name] = counts.get(name,0)+value*enabled/running
        return dict(counts=counts,unavailable=self.errors)

    def close(self):
        for _,fd in self.fds:
            os.close(fd)


class UCI:
    def __init__(self, binary, width, hash_mib, output, prefix=None, smoke=False):
        mask = [13] if smoke else PHYSICAL[:width]
        env = dict(os.environ, GOMAXPROCS=str(width))
        if prefix is not None:
            env['NGN_S100_PROFILE_PREFIX'] = str(prefix)
        self.argv = ['taskset', '-c', ','.join(map(str, mask)), 'nice', '-n', '19', str(binary)]
        self.child = subprocess.Popen(self.argv, env=env, cwd=output, stdin=subprocess.PIPE,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, bufsize=1)
        self.q, self.errors = queue.Queue(), []

        def read():
            for line in self.child.stdout:
                self.q.put(line.strip())
            self.q.put(None)

        self.reader = threading.Thread(target=read)
        self.err_reader = threading.Thread(target=lambda: self.errors.extend(self.child.stderr.read().splitlines()))
        self.reader.start()
        self.err_reader.start()
        self.send('uci')
        self.until('uciok')
        for name, value in [('EvalFile', str(NET)), ('UseNNUE', 'true'), ('Threads', str(width)),
                            ('Hash', str(hash_mib)), ('OwnBook', 'false'), ('Move Overhead', '100')]:
            self.send(f'setoption name {name} value {value}')
        self.send('isready')
        self.until('readyok')
        # The final stats line includes nodes in the unfinished last iteration;
        # ordinary depth reports count only through the last completed one.
        self.send('debug on')
        self.counters = Counters(self.child.pid)

    def send(self, text):
        self.child.stdin.write(text + '\n')
        self.child.stdin.flush()

    def until(self, prefix):
        rows = []
        while True:
            line = self.q.get(timeout=180)
            assert line is not None, self.errors
            rows.append(line)
            if line.startswith(prefix):
                return rows

    def search(self, frame, milliseconds):
        self.send('ucinewgame')
        self.send('position startpos moves ' + ' '.join(frame['moves']))
        self.send('isready')
        self.until('readyok')
        before = cpu_times()
        stat = Path(f'/proc/{self.child.pid}/stat').read_text().split()
        ticks = int(stat[13]) + int(stat[14])
        self.counters.start()
        started = time.monotonic()
        self.send(f'go movetime {milliseconds}')
        lines = self.until('bestmove ')
        elapsed = time.monotonic() - started
        counters = self.counters.stop()
        stat = Path(f'/proc/{self.child.pid}/stat').read_text().split()
        cpu_seconds = (int(stat[13]) + int(stat[14]) - ticks) / os.sysconf('SC_CLK_TCK')
        after = cpu_times()
        per_cpu = {}
        for name, end in after.items():
            delta = [b-a for a,b in zip(before[name], end)]
            total = sum(delta)
            per_cpu[name] = (total-delta[3]-delta[4])/total if total else 0
        info = [line for line in lines if line.startswith('info depth ')][-1]
        stats = [line for line in lines if line.startswith('info string stats ')][-1]
        nodes = int(re.search(r'\bnodes (\d+)', stats)[1])
        return dict(id=frame['id'],requested_ms=milliseconds,seconds=elapsed,cpu_seconds=cpu_seconds,
            cpu_wall=cpu_seconds/elapsed,nodes=nodes,nps=nodes/elapsed,
            depth=int(re.search(r'\bdepth (\d+)', info)[1]),
            score=re.search(r'\bscore (cp|mate) (-?\d+)', info).group(0),
            bestmove=lines[-1].split()[1],per_cpu_busy=per_cpu,hardware_counters=counters,
            raw_info=info,raw_stats=stats)

    def close(self):
        self.send('quit')
        assert self.child.wait(timeout=30) == 0
        self.reader.join()
        self.err_reader.join()
        self.counters.close()
        assert all('Crash Handler Initialized' in line or 'Crash log:' in line for line in self.errors), self.errors


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--smoke', action='store_true')
    parser.add_argument('--out', type=Path, default=S/'panel')
    args = parser.parse_args()
    out = args.out
    out.mkdir(exist_ok=False)
    frames = json.loads((R/'session77-smp-panel/panel.json').read_text())
    manifest = dict(source='a0ebe32 / accepted9ec5d31',net=str(NET),hash_mib=256,
        widths=[1,4,8],frames=frames,blocks=8,milliseconds=20000,
        profiles='Separate diagnostic binary; 12 frames x20s at each width; CPU/mutex/GC',
        pmu='Per engine thread, inherited by later threads; user-space cycles/instructions/cache refs/misses/DTLB read misses; multiplex scaling; unsupported events reported',
        launcher='taskset physical subset;nice19;GOMAXPROCS=Threads;UCI Hash256/overhead100/bookfalse',
        purpose='diagnosis only; next one-hotspot speed candidate requires exact T1 identity and balanced timing',
        smoke=args.smoke,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        artifacts={str(path):hashlib.sha256(path.read_bytes()).hexdigest() for path in
            [S/'baseline-ngn',S/'profile-ngn',NET,R/'session77-smp-panel/panel.json',S/'panel.py']})
    write(out/'manifest.json', manifest)
    rows = []
    with (out/'samples.jsonl').open('x') as journal:
        for block in range(1 if args.smoke else 8):
            order = [1] if args.smoke else ([1,4,8] if block%2==0 else [8,4,1])
            for width in order:
                e = UCI(S/'baseline-ngn',width,256,out,smoke=args.smoke)
                try:
                    for frame in frames[:1] if args.smoke else frames:
                        row = dict(e.search(frame,500 if args.smoke else 20000),width=width,block=block)
                        rows.append(row)
                        journal.write(json.dumps(row)+'\n')
                        journal.flush()
                finally:
                    e.close()
    shares = {}
    for width in ([1] if args.smoke else [1,4,8]):
        prefix = out/f'profile-t{width}'
        e = UCI(S/'profile-ngn',width,256,out,prefix,smoke=args.smoke)
        try:
            for frame in frames[:1] if args.smoke else frames:
                e.search(frame,500 if args.smoke else 20000)
        finally:
            e.close()
        for kind in ['cpu','mutex']:
            command = ['/usr/local/go/bin/go','tool','pprof','-top','-nodecount=0','-nodefraction=0',
                       str(S/'profile-ngn'),str(prefix)+f'.{kind}.pprof']
            raw = subprocess.check_output(command, text=True, stderr=subprocess.STDOUT)
            (out/f'profile-t{width}-{kind}-top.txt').write_text(raw)
            if kind == 'cpu':
                categories = dict(tt=0.0,move_picking_history=0.0,nnue=0.0,gc=0.0)
                for line in raw.splitlines():
                    parts = line.split()
                    if len(parts)<6 or not parts[1].endswith('%') or not parts[2].endswith('%'):
                        continue
                    percent = float(parts[1][:-1])
                    name = ' '.join(parts[5:])
                    for key, pattern in dict(tt=r'\(\*Cache\)|ttRetention|\.Pack$|\.Unpack$',
                            move_picking_history=r'scoreMoves|selectNextMove|materializeCapture|History|historyResponse',
                            nnue=r'NGNN|ngnn|NNUE|nnue|Accumulator',
                            gc=r'runtime\.(?:gc|scan|mark|sweep)').items():
                        if re.search(pattern,name):
                            categories[key] += percent
                shares[width] = categories
    summary = {}
    for width in sorted({r['width'] for r in rows}):
        cells = [r for r in rows if r['width']==width]
        seconds = sum(r['seconds'] for r in cells)
        nodes = sum(r['nodes'] for r in cells)
        summary[width] = dict(samples=len(cells),nps=nodes/seconds,nps_per_core=nodes/seconds/width,
            cpu_wall=sum(r['cpu_seconds'] for r in cells)/seconds,
            mean_completed_depth=sum(r['depth'] for r in cells)/len(cells))
        counts = {}
        for row in cells:
            for key,value in row['hardware_counters']['counts'].items():
                counts[key] = counts.get(key,0)+value
        summary[width]['hardware_counters'] = counts
        summary[width]['counters_per_node'] = {key:value/nodes for key,value in counts.items()}
    write(out/'result.json',dict(state='SMOKE_PASS' if args.smoke else 'MEASURED',widths=summary,
        profile_flat_cpu_percent=shares,
        profiles='profile-t{1,4,8}-{cpu,mutex}-top.txt and *.gc.json; inspect before timing a candidate',
        finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),automatic_adoption=False))


if __name__ == '__main__':
    main()
