"""Fixed Rodent confirmation after a manual final-package nomination."""
import datetime
import hashlib
import importlib.util
import json
import math
import os
from pathlib import Path
import platform
import subprocess
import sys

import chess


def clock_audit(rows):
    searched = 0
    for pair in rows:
        opening = pair['opening']
        fen = '/' in opening
        free = [] if fen else opening.split()
        for game in pair['games']:
            board = chess.Board(opening) if fen else chess.Board()
            for move in free:
                board.push_uci(move)
            moves = game['moves'] or []
            clocks = game.get('clocks') or []
            assert len(clocks) == len(moves) - len(free), 'missing clock plies'
            white, black = 120000, 120000
            for i, row in enumerate(clocks, len(free)):
                assert row['ply'] == i + 1 and row['white'] is board.turn
                assert row['move'] == moves[i] and not row['flagged']
                assert row['white_before_ms'] == white and row['black_before_ms'] == black
                assert row['go'] == f'go wtime {white} btime {black} winc 1000 binc 1000'
                elapsed = row['elapsed_ms']
                assert isinstance(elapsed, int) and elapsed >= 0
                assert row['increment_applied_ms'] == 1000
                if board.turn:
                    assert white - elapsed >= -300, 'unreported white flag'
                    white += 1000 - elapsed
                else:
                    assert black - elapsed >= -300, 'unreported black flag'
                    black += 1000 - elapsed
                assert row['white_after_ms'] == white and row['black_after_ms'] == black
                board.push_uci(moves[i])
                searched += 1
    return dict(searched_plies=searched, command_and_bank_conservation=True,
                tc='120+1', opening_plies_free=True, flag_grace_ms=300)


def review_stage(p, stage, expected, reporter):
    rows = [json.loads(line) for line in (p / (stage + '-pairs.jsonl')).read_text().splitlines()]
    openings = [' '.join(line.split()) for line in (p / 'openings.txt').read_text().splitlines()
                if line.strip() and not line.lstrip().startswith('#')]
    result = reporter.recount(rows, openings)
    assert result['games'] == expected and result['pairs'] == expected // 2
    assert not result['excluded_pairs'] and not result['failures']
    assert not any(result['flag_outs'].values())
    summary = json.loads((p / (stage + '-summary.json')).read_text())
    assert summary['complete'] and summary['requested_games'] == expected
    assert not summary['operational_faults'] and not summary['ngn_flag_outs'] and not summary['opponent_flag_outs']
    for key in ['games', 'pairs', 'wins', 'draws', 'losses', 'penta']:
        assert summary[key] == result[key]
    for key, value in zip(['elo', 'elo_lower95', 'elo_upper95'], result['exact_elo_ci95']):
        assert math.isclose(summary[key], value, rel_tol=1e-12, abs_tol=1e-9)
    assert len(summary['worker_receipts']) == 6
    for worker in summary['worker_receipts']:
        for side in ['ngn', 'opponent']:
            receipt = worker[side]
            assert receipt['exited'] and not receipt.get('exit_error')
            assert not receipt.get('read_error') and not receipt['startup_failures']
    crash_path = p / 'ngn_crashes.log'
    if crash_path.exists():
        assert 'PANIC RECOVERED' not in crash_path.read_text()
    result['clock_audit'] = clock_audit(rows)
    result['worker_receipts'] = summary['worker_receipts']
    result['verdict'] = ('aa-pass' if stage == 'aa' else
                         'superiority-confirmed' if result['exact_elo_ci95'][1] > 0 else 'not-confirmed')
    if stage == 'aa':
        assert result['exact_elo_ci95'][1] <= 0 <= result['exact_elo_ci95'][2]
    else:
        result['implied_performance_ci95'] = [3532 + value for value in result['exact_elo_ci95']]
    result['limitation'] = 'fixed-label single-opponent match; normal paired CI; no general rating floor or3600 claim'
    (p / (stage + '-review.json')).write_text(json.dumps(result, indent=2) + '\n')
    return result


def main(p):
    p = p.resolve()
    root = Path('/home/ehrli/ngn-data/night-20261007')
    opponent = root / 'session19-rodent3/opponent-only/rodent-v12'
    opponent_cwd = opponent.parent / 'rodent'
    nomination = json.loads((p / 'nomination.json').read_text())
    sha = lambda path: hashlib.sha256(Path(path).read_bytes()).hexdigest()
    for name, key in [('ngn', 'engine_sha256'), ('net.nnue', 'net_sha256'), ('pairmatch', 'pairmatch_sha256')]:
        assert sha(p / name) == nomination[key]
    external = {
        opponent: '10dd722e730a70887fcaa07157b86c5ba3579618f858f7519d44f55036e00949',
        opponent_cwd / 'nets/rodent_4kb_768hl_8ob_v2.bin': 'c35a1abc1b8c1cb1d5f4221454d494c1a6da1ed9088fd51ab27038bfa74b5053',
        opponent_cwd / 'personalities/rodent.txt': '489b2c5d0412a848083f0de004cda8ff968cd79d7907e4f24ed5ada161a9b093',
    }
    assert all(sha(path) == digest for path, digest in external.items())
    assert sha(p / 'openings.txt') == '974e4b5ab871a9e106d0c766bfa39fc83676222337fb7702f61782e2ac5b3222'
    env = dict(os.environ, GOMAXPROCS='1', GOAMD64='v3')
    ngn_options = [('EvalFile', str(p / 'net.nnue')), ('UseNNUE', 'true'),
                   ('Hash', '64'), ('Threads', '1'), ('OwnBook', 'false')]
    rodent_options = [('Hash', '64'), ('Threads', '1'), ('OwnBook', 'false'), ('UCI_LimitStrength', 'false')]
    commands = []
    for stage, games, seconds in [('aa', 40, 10800), ('rodent', 1000, 115200)]:
        argv = ['taskset', '-c', '0-11', 'nice', '-n', '19', str(p / 'pairmatch'),
                '-ngn', str(p / 'ngn'), '-opponent', str(p / 'ngn' if stage == 'aa' else opponent),
                '-openings', str(p / 'openings.txt'), '-concurrency', '6',
                '-games', str(games), '-out', str(p / (stage + '-pairs.jsonl'))]
        for side, options in [('ngn', ngn_options), ('opponent', ngn_options if stage == 'aa' else rodent_options)]:
            for key, value in options:
                argv += ['-' + side + 'option', key + '=' + value]
        commands.append(dict(stage=stage, games=games, timeout_seconds=seconds, argv=argv))
    names = ['ngn', 'net.nnue', 'pairmatch', 'openings.txt', 'nomination.json',
             'manifest.md', 'rodent-report.py', 'controller.py', 'controller.sh']
    hashes = {str(p / name): sha(p / name) for name in names}
    hashes.update({str(path): digest for path, digest in external.items()})
    manifest = dict(created_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                    nomination=nomination, machine='rig WSL2/Ryzen9800X3D', os=platform.platform(),
                    cpus='0-11', concurrency=6, nice=19, memory_max_bytes=3 * 1024**3,
                    memory_swap_max=0, tc='120+1', environment=dict(GOMAXPROCS='1', GOAMD64='v3'),
                    opponent_label=3532, cwd=str(opponent_cwd), commands=commands,
                    files_sha256=hashes, decision='ownAA40; fixed1000/500 clean/complete/intact; paired lower95>0 only',
                    automatic_adoption=False, claim_eligible=False)
    (p / 'game-manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    hashes[str(p / 'game-manifest.json')] = sha(p / 'game-manifest.json')
    (p / 'inputs.sha256').write_text(''.join(digest + '  ' + path + '\n' for path, digest in hashes.items()))
    spec = importlib.util.spec_from_file_location('reporter', p / 'rodent-report.py')
    reporter = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(reporter)

    def status(event, detail):
        with (p / 'status.jsonl').open('a') as stream:
            stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                        stage='rodent-confirmation', event=event, detail=detail)) + '\n')

    for command in commands:
        stage = command['stage']
        for suffix in ['before', 'after']:
            if suffix == 'before':
                with (p / (stage + '-summary.json')).open('w') as out, (p / (stage + '-stderr.log')).open('w') as err:
                    with (p / ('input-check-' + stage + '-before.txt')).open('w') as checks:
                        subprocess.run(['sha256sum', '-c', str(p / 'inputs.sha256')], stdout=checks, check=True)
                    status(stage + '-start', dict(games=command['games'], pid=os.getpid()))
                    subprocess.run(['timeout', '--signal=TERM', '--kill-after=15s', str(command['timeout_seconds']), *command['argv']],
                                   cwd=opponent_cwd, env=env, stdout=out, stderr=err, check=True)
            else:
                with (p / ('input-check-' + stage + '-after.txt')).open('w') as checks:
                    subprocess.run(['sha256sum', '-c', str(p / 'inputs.sha256')], stdout=checks, check=True)
        review = review_stage(p, stage, command['games'], reporter)
        status(stage + '-done', {key: review[key] for key in ['games', 'pairs', 'penta', 'exact_elo_ci95', 'verdict']})


if __name__ == '__main__':
    main(Path(sys.argv[1]))
