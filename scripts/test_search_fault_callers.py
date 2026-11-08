"""A transient search error must never produce usable match results."""
import datetime
import fcntl
import json
import os
from pathlib import Path
import shlex
import subprocess
import sys
import tempfile
import unittest


def peer():
    """Mate-in-one UCI peer with one shared, consumable search fault."""
    switch, trace = map(Path, sys.argv[2:])
    for line in sys.stdin:
        line = line.strip()
        if line in ('ucinewgame', 'quit'):
            with trace.open('a') as output:
                output.write(json.dumps({'pid': os.getpid(), 'event': line, 'fault': False}) + '\n')
        if line == 'uci':
            print('uciok', flush=True)
        elif line == 'isready':
            print('readyok', flush=True)
        elif line.startswith('go '):
            if line == 'go movetime 200':
                print('bestmove e2e4', flush=True)
                continue
            with switch.open('r+') as state:
                fcntl.flock(state, fcntl.LOCK_EX)
                fault = state.read() == 'fault'
                if fault:
                    state.seek(0)
                    state.write('healthy')
                    state.truncate()
            if fault:
                print('info string error search session failed: transient fixture', flush=True)
            with trace.open('a') as output:
                output.write(json.dumps({'pid': os.getpid(), 'event': 'search', 'fault': fault}) + '\n')
            print('bestmove g6g7', flush=True)
        elif line == 'quit':
            break


class SearchFaultCallersTests(unittest.TestCase):
    def test_transient_fault_excludes_results_and_stops_poisoned_process(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ('pairmatch', 'gauntlet'):
                subprocess.run(['go', 'build', '-p', '2', '-o', str(root / name),
                                './cmd/' + name], check=True)
            book = root / 'openings.txt'
            book.write_text(('7k/8/5KQ1/8/8/8/8/8 w - - 0 1\n') * 2)
            peers = {}
            for name in ('ngn', 'opponent'):
                switch, trace = root / (name + '.mode'), root / (name + '.trace')
                path = root / name
                path.write_text('#!/bin/sh\nexec ' + shlex.join(
                    [sys.executable, str(Path(__file__).resolve()), '--peer',
                     str(switch), str(trace)]) + '\n')
                path.chmod(0o700)
                peers[name] = (path, switch, trace)
            anchors = root / 'anchors.json'
            anchors.write_text(json.dumps([{'Version': 'fixture',
                                           'Path': str(peers['opponent'][0]),
                                           'CCRL': 2500}]))
            for caller, concurrency, restart in [('pairmatch', 1, 0),
                                                 ('gauntlet', 1, 0),
                                                 ('gauntlet', 1, 1),
                                                 ('gauntlet', 2, 1)]:
                for faulty in (None, 'ngn', 'opponent'):
                    with self.subTest(caller=caller, concurrency=concurrency,
                                      restart=restart, faulty=faulty):
                        tag = f'{caller}-{concurrency}-{restart}-{faulty}'
                        for name, (_, switch, trace) in peers.items():
                            switch.write_text('fault' if name == faulty else 'healthy')
                            trace.write_text('')
                        output = root / (tag + '.json')
                        command = [str(root / caller), '-ngn', str(peers['ngn'][0]),
                                   '-openings', str(book), '-concurrency', str(concurrency)]
                        if caller == 'pairmatch':
                            deadline = datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(hours=1)
                            command += ['-opponent', str(peers['opponent'][0]),
                                        '-until', deadline.isoformat(), '-out', str(output)]
                        else:
                            command += ['-anchors', str(anchors), '-games', '4', '-tc', '10+0.1',
                                        '-restart-every', str(restart), '-lowpower=false',
                                        '-no-record', '-tally-out', str(output)]
                        run = subprocess.run(command, text=True, capture_output=True, timeout=15)
                        print(tag, 'exit', run.returncode, run.stdout.strip(), run.stderr.strip(), flush=True)
                        if caller == 'pairmatch':
                            summary = json.loads(run.stdout)
                            self.assertEqual(run.returncode, 1 if faulty else 0)
                            self.assertEqual(summary['games'], 0 if faulty else 4)
                            self.assertEqual(summary['pairs'], 0 if faulty else 2)
                            self.assertEqual(summary['operational_faults'], 1 if faulty else 0)
                            records = [json.loads(line) for line in output.read_text().splitlines()]
                            if faulty:
                                self.assertEqual(len(records), 1)
                                self.assertIn('search-error', records[0]['error'])
                        else:
                            self.assertEqual(run.returncode, 0)
                            tallies = json.loads(output.read_text())
                            if faulty:
                                self.assertFalse(tallies)
                                self.assertIn('EXCLUDED: fixture', run.stdout)
                            else:
                                self.assertEqual((tallies[0]['w'], tallies[0]['d'], tallies[0]['l']), (2, 0, 2))
                        searches = [json.loads(line) for _, _, trace in peers.values()
                                    for line in trace.read_text().splitlines()]
                        faults = [entry for entry in searches if entry['fault']]
                        self.assertEqual(len(faults), 1 if faulty else 0)
                        for fault in faults:
                            process = [entry for entry in searches if entry['pid'] == fault['pid']]
                            self.assertEqual([entry['event'] for entry in process[process.index(fault)+1:]],
                                             ['quit'], 'the poisoned process was reused or left running')


if __name__ == '__main__':
    if len(sys.argv) > 1 and sys.argv[1] == '--peer':
        peer()
    else:
        unittest.main()
