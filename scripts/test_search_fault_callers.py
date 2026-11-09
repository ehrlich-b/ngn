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


def matrix_peer():
    mode, trace = sys.argv[2:]
    in_game = False
    def record(event):
        with Path(trace).open('a') as output:
            output.write(json.dumps({'pid': os.getpid(), 'event': event}) + '\n')
    record('start')
    for raw in sys.stdin:
        line = raw.strip()
        if line in ('ucinewgame', 'quit') or line.startswith('go '):
            record(line)
        if line == 'uci':
            print('uciok', flush=True)
        elif line == 'ucinewgame':
            in_game = True
        elif line == 'isready':
            if mode == 'readiness-error' and in_game:
                record('fault')
                print('info string error rejected game configuration', flush=True)
            else:
                print('readyok', flush=True)
        elif line.startswith('go '):
            if not in_game:
                print('bestmove e2e4', flush=True)
            elif mode == 'eof':
                record('fault')
                break
            else:
                if mode == 'search-error':
                    record('fault')
                    print('info string error search session failed: local fixture', flush=True)
                print('bestmove g6g7', flush=True)
        elif line == 'quit':
            break


class SearchFaultCallersTests(unittest.TestCase):
    def test_sprt_aborts_on_search_fault_before_scoring(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binary = root / 'sprt'
            subprocess.run(['go', 'build', '-p', '2', '-o', str(binary), './cmd/sprt'], check=True)
            book = root / 'openings.txt'
            book.write_text(('7k/8/5KQ1/8/8/8/8/8 w - - 0 1\n') * 2)
            peers = {}
            for name in ('new', 'base'):
                switch, trace = root / (name + '.mode'), root / (name + '.trace')
                path = root / name
                path.write_text('#!/bin/sh\nexec ' + shlex.join(
                    [sys.executable, str(Path(__file__).resolve()), '--peer',
                     str(switch), str(trace)]) + '\n')
                path.chmod(0o700)
                peers[name] = (path, switch, trace)
            for strict in (False, True):
                for faulty in (None, 'new', 'base'):
                    with self.subTest(strict=strict, faulty=faulty):
                        for name, (_, switch, trace) in peers.items():
                            switch.write_text('fault' if name == faulty else 'healthy')
                            trace.write_text('')
                        pgn = root / 'games.pgn'
                        command = [str(binary), '-new', str(peers['new'][0]),
                                   '-base', str(peers['base'][0]), '-openings', str(book),
                                   '-concurrency', '1', '-maxgames', '4', '-mingames', '4',
                                   '-lowpower=false', '-pgn', str(pgn)]
                        if strict:
                            command.append('-noretries')
                        run = subprocess.run(command, text=True, capture_output=True, timeout=15)
                        self.assertEqual(run.returncode == 0, faulty is None, run.stdout + run.stderr)
                        if faulty:
                            self.assertIn('search-error', run.stderr)
                            self.assertNotIn('ACCEPTED', run.stdout)
                            self.assertEqual(pgn.read_text().count('[Event '), 1 if faulty == 'base' else 0)
                            events = [json.loads(line) for line in peers[faulty][2].read_text().splitlines()]
                            fault_index = next(i for i, event in enumerate(events) if event['fault'])
                            self.assertEqual([event['event'] for event in events[fault_index + 1:]], ['quit'])
                        else:
                            self.assertEqual(pgn.read_text().count('[Event '), 4)

    def test_sprt_strict_engine_fault_matrix(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binary = root / 'sprt'
            subprocess.run(['go', 'build', '-p', '2', '-o', str(binary), './cmd/sprt'], check=True)
            book = root / 'openings.txt'
            book.write_text('7k/8/5KQ1/8/8/8/8/8 w - - 0 1\n' * 2)
            cases = [(True, 'healthy', None)]
            cases += [(strict, 'search-error', side) for strict in (False, True)
                      for side in ('new', 'base')]
            cases += [(True, mode, side) for mode in ('readiness-error', 'eof')
                      for side in ('new', 'base')]
            for strict, mode, faulty in cases:
                with self.subTest(strict=strict, mode=mode, faulty=faulty):
                    paths, traces = {}, {}
                    for side in ('new', 'base'):
                        paths[side] = root / side
                        traces[side] = root / (side + '.trace')
                        traces[side].write_text('')
                        paths[side].write_text('#!/bin/sh\nexec ' + shlex.join([
                            sys.executable, str(Path(__file__).resolve()), '--matrix-peer',
                            mode if side == faulty else 'healthy', str(traces[side])]) + '\n')
                        paths[side].chmod(0o700)
                    pgn = root / 'matrix.pgn'
                    command = [str(binary), '-new', str(paths['new']), '-base', str(paths['base']),
                               '-openings', str(book), '-concurrency', '1', '-maxgames', '4',
                               '-mingames', '4', '-lowpower=false', '-pgn', str(pgn)]
                    if strict:
                        command.append('-noretries')
                    run = subprocess.run(command, capture_output=True, text=True, timeout=15)
                    text = pgn.read_text()
                    events = {side: [json.loads(line) for line in trace.read_text().splitlines()]
                              for side, trace in traces.items()}
                    if not faulty:
                        self.assertEqual(run.returncode, 0, run.stderr)
                        self.assertEqual(text.count('[Event '), 4)
                        self.assertIn('W-D-L: 2-0-2', run.stdout)
                    else:
                        self.assertNotEqual(run.returncode, 0, run.stdout)
                        self.assertIn('search-error' if mode == 'search-error' else 'engine-error', run.stderr)
                        self.assertNotIn('ACCEPTED', run.stdout)
                        expected = 1 if faulty == 'base' and mode != 'readiness-error' else 0
                        self.assertEqual(text.count('[Event '), expected)
                        self.assertNotIn('engine-error', text)
                        self.assertLessEqual(sum(line.startswith('G') for line in run.stdout.splitlines()), expected)
                        self.assertNotIn('Pentanomial', run.stdout)
                        for side_events in events.values():
                            self.assertEqual(sum(e['event'] == 'start' for e in side_events), 1)
                            self.assertLessEqual(sum(e['event'] == 'ucinewgame' for e in side_events), expected + 1)
                        fault_events = events[faulty]
                        index = next(i for i, event in enumerate(fault_events) if event['event'] == 'fault')
                        self.assertTrue(all(e['event'] == 'quit' for e in fault_events[index + 1:]))

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
    elif len(sys.argv) > 1 and sys.argv[1] == '--matrix-peer':
        matrix_peer()
    else:
        unittest.main()
