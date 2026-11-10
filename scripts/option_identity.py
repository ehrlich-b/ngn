#!/usr/bin/env python3
"""Offline depth-12 node identity against an explicit original engine checkout."""
import argparse
import json
from pathlib import Path
import queue
import re
import subprocess
import threading

from scripts import match_queue as mq
from scripts import option_verdict as ov

BASE = 'bc87352d94c58f58eeae42fb39989d003094252d'
CARRY = '7c2776ffd38c503f636133a015e14eacd3c0ab1b'


def prepare_base(base_repo, carry):
    """Create the original worktree if absent; never repair a dirty/wrong one."""
    def git(repo, *args):
        return subprocess.check_output(['git', '-C', str(repo), *args],
                                       env=mq.clean_env(), text=True).strip()

    base_repo = base_repo.resolve()
    if carry is not None:
        carry = carry.resolve()
        if git(carry, 'rev-parse', 'HEAD') != CARRY:
            raise ValueError('wrong queue carry')
        if git(carry, 'status', '--porcelain', '--untracked-files=all'):
            raise ValueError('queue carry is dirty')
        subprocess.run(['git', '-C', str(carry), 'merge-base', '--is-ancestor', BASE, CARRY],
                       env=mq.clean_env(), check=True)
        if not base_repo.exists():
            subprocess.run(['git', '-C', str(carry), 'worktree', 'add', '--detach',
                            str(base_repo), BASE], env=mq.clean_env(), check=True)
        if git(base_repo, 'rev-parse', '--path-format=absolute', '--git-common-dir') != git(
                carry, 'rev-parse', '--path-format=absolute', '--git-common-dir'):
            raise ValueError('original engine must be a worktree of the queue carry')
    if git(base_repo, 'rev-parse', 'HEAD') != BASE:
        raise ValueError('wrong original engine baseline')
    if git(base_repo, 'status', '--porcelain', '--untracked-files=all'):
        raise ValueError('original engine baseline is dirty')
    print(f'Clean original engine worktree verified at {BASE}', flush=True)


class Engine:
    def __init__(self, binary):
        self.process = subprocess.Popen([str(binary)], stdin=subprocess.PIPE,
                                        stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                        text=True, bufsize=1, env=mq.clean_env())
        self.lines = queue.Queue()

        def read():
            for line in self.process.stdout:
                self.lines.put(line.strip())
            self.lines.put(None)

        threading.Thread(target=read, daemon=True).start()
        self.send('uci')
        advertised = self.wait('uciok')
        self.names = set(re.findall(r'option name (\S+) type spin', '\n'.join(advertised)))
        for name, value in [('Threads', '1'), ('Hash', '16'), ('OwnBook', 'false'), ('UseNNUE', 'false')]:
            self.send(f'setoption name {name} value {value}')
        self.send('debug on')
        self.send('isready')
        self.wait('readyok')

    def send(self, line):
        self.process.stdin.write(line + '\n')
        self.process.stdin.flush()

    def wait(self, token):
        lines = []
        while True:
            line = self.lines.get(timeout=180)
            if line is None:
                raise ValueError('engine exited before ' + token)
            if 'info string error' in line:
                raise ValueError(line)
            lines.append(line)
            if line.startswith(token):
                return lines

    def search(self, fen, option=None):
        if option:
            if option not in self.names:
                raise ValueError('missing UCI option: ' + option)
            # Exercise the on -> off transition before the measured cold search.
            for value in (1, 0):
                self.send(f'setoption name {option} value {value}')
        self.send('ucinewgame')
        self.send('isready')
        self.wait('readyok')
        self.send('position fen ' + fen)
        self.send('go depth 12')
        lines = self.wait('bestmove')
        counts = re.findall(r'^info string stats depth 12 seldepth \d+ nodes (\d+) ', '\n'.join(lines), re.M)
        if len(counts) != 1:
            raise ValueError('missing completed depth-12 node count')
        return int(counts[0])

    def close(self):
        if self.process.poll() is None:
            self.send('quit')
            self.process.wait(timeout=10)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base-repo', type=Path, required=True)
    parser.add_argument('--base-carry', type=Path,
                        help='clean pinned queue checkout from which to create/verify the original worktree')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    prepare_base(args.base_repo, args.base_carry)
    output = args.output.resolve()
    if output == mq.REPO or mq.REPO in output.parents:
        raise ValueError('identity evidence must be outside checkout')
    output.mkdir(parents=True, exist_ok=True)
    engines = []
    try:
        for name, repo in [('base', args.base_repo), ('candidate', mq.REPO)]:
            binary = output / name
            subprocess.run(['go', 'build', '-p', '2', '-o', str(binary), '.'],
                           cwd=repo, env=mq.clean_env(), check=True)
            engines.append(Engine(binary))
        # The three canonical nodecheck bench positions; all use depth 12 here.
        bench = (mq.REPO / 'scripts/nodecheck.sh').read_text()
        positions = re.findall(r'^  "([^|]+)\|([^|]+)\|\d+\|\d+"', bench, re.M)
        if len(positions) != 3:
            raise ValueError('expected three canonical bench positions')
        reference = {name: engines[0].search(fen) for name, fen in positions}
        comparisons = []
        for option in ov.options():
            for name, fen in positions:
                actual = engines[1].search(fen, option)
                row = dict(option=option, position=name, depth=12, base=reference[name], candidate=actual)
                comparisons.append(row)
                print(json.dumps(row, sort_keys=True), flush=True)
                if actual != reference[name]:
                    raise ValueError('node identity mismatch')
        (output / 'comparisons.json').write_text(json.dumps(dict(
            baseline=BASE, comparisons=comparisons), indent=2) + '\n')
        print(f'{len(comparisons)} depth-12 comparisons passed', flush=True)
    finally:
        for engine in engines:
            engine.close()


if __name__ == '__main__':
    main()
