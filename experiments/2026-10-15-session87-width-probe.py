"""Bounded occupancy probe through the exact pinned game launcher and roles."""
import collections
import datetime
import json
import os
from pathlib import Path
import subprocess
import sys
import time

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R/'session87'
HZ = os.sysconf('SC_CLK_TCK')


def probe(job, name, widths, masks):
    directory = D/name
    directory.mkdir()
    argv = ['taskset','-c','0-11',job['harness'],'-new',job['candidate'],'-base',job['base'],
            '-movetime','5000','-maxgames','2','-mingames','1000000','-maxmoves','2',
            '-concurrency','1','-lowpower=false','-gametimeout','60','-openings',job['book'],
            '-pgn',str(directory/'games.pgn')]
    for role, options in [('new',job['candidate_options']),('base',job['base_options'])]:
        for option in options:
            argv += ['-'+role+'option',option]
    manifest = dict(argv=argv,environment={'GOMAXPROCS':'1'},
                    purpose='Unscored five-second searches through actual sprt/role launchers before A/A',
                    started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    (directory/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
    samples = collections.defaultdict(list)
    children = {}
    with (directory/'probe.log').open('w') as log:
        child = subprocess.Popen(argv,env=dict(os.environ,GOMAXPROCS='1'),stdout=log,stderr=subprocess.STDOUT)
        started = time.monotonic()
        while child.poll() is None:
            now = time.monotonic()
            if now-started > 90:
                child.kill()
                child.wait()
                raise RuntimeError('Unscored width probe exceeded90s')
            for path in Path('/proc').glob('[0-9]*/stat'):
                try:
                    fields = path.read_text().rsplit(')',1)[1].split()
                    if int(fields[1]) != child.pid:
                        continue
                    command = (path.parent/'cmdline').read_text().split('\0')
                    if command[0] != job['engine']:
                        continue
                    pid = int(path.parent.name)
                    samples[pid].append((now,(int(fields[11])+int(fields[12]))/HZ))
                    if pid not in children:
                        environment = dict(item.split('=',1) for item in (path.parent/'environ').read_text().split('\0') if '=' in item)
                        children[pid] = dict(pid=pid,gomaxprocs=environment.get('GOMAXPROCS'),
                                             cpus=sorted(os.sched_getaffinity(pid)),
                                             command=' '.join(command))
                except (FileNotFoundError,ProcessLookupError,PermissionError):
                    continue
            time.sleep(.10)
    assert child.returncode == 0, f'probe harness exit{child.returncode}'
    for pid, rows in samples.items():
        peak = 0
        for index,(wall,cpu) in enumerate(rows):
            previous = index-1
            while previous >= 0 and wall-rows[previous][0] < 4.8:
                previous -= 1
            if previous >= 0 and wall-rows[previous][0] <= 5.2:
                peak = max(peak,(cpu-rows[previous][1])/(wall-rows[previous][0]))
        children[pid]['peak_5s_cpu_wall_ratio'] = peak
    rows = list(children.values())
    assert len(rows) == 2, f'Expected two live engine children: {rows}'
    # The two workers alternate searches; use each child's busiest five-second
    # window, rather than diluting CPU occupancy with its opponent's turns.
    for width,mask in zip(widths,masks):
        matching = [row for row in rows if row['cpus'] == mask and row['gomaxprocs'] == str(width)]
        assert matching, f'No child with GOMAXPROCS{width}/CPUs{mask}: {rows}'
        assert all(row['peak_5s_cpu_wall_ratio'] >= .78*width for row in matching), matching
    result = dict(passed=True,children=rows,expected_widths=widths,expected_masks=masks,
                  method='Peak measured CPU seconds / wall seconds over4.8–5.2s windows in go movetime5000 searches; no score verdict')
    (directory/'result.json').write_text(json.dumps(result,indent=2)+'\n')
    return result


def main():
    spec = json.loads((D/'width-probe.json').read_text())
    result = [probe(spec['aa'],'width-probe-aa',[4,4],[[0,2,4,6],[0,2,4,6]]),
              probe(spec['measurement'],'width-probe-measurement',[4,1],[[0,2,4,6],[8]])]
    (D/'width-probe-result.json').write_text(json.dumps(result,indent=2)+'\n')


if __name__ == '__main__':
    main()
