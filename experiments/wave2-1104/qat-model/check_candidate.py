"""Check candidate ancestry, commit custody, and the committed fixture inputs."""

import json
from pathlib import Path
import subprocess

BASE = 'b73df862fdfc2b72d30f55180050d467b429f45a'
IDENTITY = 'Bryan Ehrlich <ehrlich.bryan@gmail.com>'


def git(*args):
    return subprocess.check_output(['git', *args], text=True).strip()


def main():
    subprocess.run(['git', 'merge-base', '--is-ancestor', BASE, 'HEAD'], check=True)
    commits = git('rev-list', BASE + '..HEAD').splitlines()
    if not commits:
        raise SystemExit('Commit the candidate before checking custody')
    for commit in commits:
        identities = git('show', '-s', '--format=%an <%ae>%n%cn <%ce>', commit).splitlines()
        if identities != [IDENTITY, IDENTITY]:
            raise SystemExit('Unexpected commit identity: ' + commit)
    root = Path('experiments/wave2-1104/qat-model')
    for name in ('cases.json', 'fixture.py', 'run.sh', 'run.py', 'result.json'):
        subprocess.run(['git', 'ls-files', '--error-unmatch', str(root / name)], check=True,
                       stdout=subprocess.DEVNULL)
    if not (root / 'run.sh').stat().st_mode & 0o111:
        raise SystemExit('Fixture command must be executable')
    if git('status', '--porcelain', '--untracked-files=all'):
        raise SystemExit('Candidate checkout must be clean')
    print(json.dumps({'base': BASE, 'head': git('rev-parse', 'HEAD'),
                      'new_commits': commits, 'author_and_committer': IDENTITY,
                      'fixture_and_command_committed': True}, indent=2))


if __name__ == '__main__':
    main()
