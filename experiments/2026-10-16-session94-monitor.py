"""Descriptive claim sidecar: one snapshot every ten minutes, no game control."""
import datetime
import json
import os
from pathlib import Path
import select
import sys
import time


def snapshot():
    cpus = {}
    for line in Path('/proc/stat').read_text().splitlines():
        fields = line.split()
        if fields[0].startswith('cpu') and fields[0][3:].isdigit():
            values = list(map(int, fields[1:9]))
            cpus[fields[0]] = [sum(values), sum(values) - values[3] - values[4]]
    processes = {}
    for path in Path('/proc').glob('[0-9]*/stat'):
        try:
            raw = path.read_text()
            fields = raw.rsplit(')', 1)[1].split()
            exe = os.readlink(path.parent / 'exe')
            # Every engine is a Linux executable in these frozen directories.
            if '/anchors/' in exe or exe.endswith('/session94/playing/ngn'):
                continue
            processes[int(path.parent.name)] = dict(ticks=int(fields[11]) + int(fields[12]), exe=exe,
                                                   name=raw.split('(', 1)[1].rsplit(')', 1)[0])
        except (FileNotFoundError, ProcessLookupError, PermissionError):
            continue
    return cpus, processes


def main():
    parent, output = int(sys.argv[1]), Path(sys.argv[2])
    fd = os.pidfd_open(parent)
    previous_cpus, previous_processes = snapshot()
    previous_time = time.monotonic()
    with output.open('a') as stream:
        stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                     event='monitor-start', interval_seconds=600, cpu_ticks=previous_cpus,
                     interpretation='Descriptive only; no automatic stop or verdict effect')) + '\n')
    try:
        while not select.select([fd], [], [], 600)[0]:
            cpus, processes = snapshot()
            wall = time.monotonic()
            elapsed = wall - previous_time
            busy = {}
            for cpu, (total, active) in cpus.items():
                before = previous_cpus[cpu]
                busy[cpu] = 100 * (active - before[1]) / max(1, total - before[0])
            hz = os.sysconf('SC_CLK_TCK')
            top = []
            for pid, row in processes.items():
                before = previous_processes.get(pid, {}).get('ticks', 0)
                top.append(dict(pid=pid, exe=row['exe'], name=row['name'],
                                cpu_percent=100 * max(0, row['ticks'] - before) / hz / elapsed))
            top.sort(key=lambda row: row['cpu_percent'], reverse=True)
            with output.open('a') as stream:
                stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                     interval_seconds=elapsed, cpu_busy_percent=busy, top_non_engine=top[:3])) + '\n')
            previous_cpus, previous_processes, previous_time = cpus, processes, wall
    finally:
        os.close(fd)


if __name__ == '__main__':
    main()
