#!/usr/bin/env python3
"""Verify the pinned native GoChess v5 model at startup and every refresh."""
import json
import os
import re
import sys
from collections import Counter
from pathlib import Path
import time

import ownedmatch as base
sys.path.insert(0, '/home/ehrli/nnue-owned-continuation-20260928/counterdraw-deps/chess-1.11.2')
import chess
from counter_draw_compat import DrawPVChecker, filter_draw_warnings

ENGINE_SHA = 'f98888b058dfb62060f0bf992adab2f6ae59eeb47f795a463ffae0b9f846c96c'
NETWORK_SHA = 'be22a20bfde78d7915cc0cc5fe84e917a76d5df130b9555884a570a04b3ab2fc'
FINGERPRINT = '9b04e49c9c567240'
NETWORK_NAME = 'net-v5-120sb-sb120.nnue'
NET_TXT = 'https://github.com/adamtwiss/gochess/releases/download/v0.5.0/' + NETWORK_NAME + '\n'
FINGERPRINT_LINE = 'info string SetNNUEV5 fingerprint ' + FINGERPRINT
USE_LINE = 'info string UseNNUE = true'
CORE_SHA = '1fde6a8e932508d14b31f6a584497b49c2969fc3c7cdc675347f819f0d667a5b'
if chess.__version__ != '1.11.2' or base.sha256(Path(chess.__file__)) != CORE_SHA:
    raise RuntimeError('independent chess validator differs from the pinned dependency')
INFO = re.compile(r'^info depth (\d+) score cp (-?\d+) nodes (\d+) nps (\d+) time (\d+) hashfull (\d+) pv (.+)$')

original_validate = base.validate_one_worker_role
original_stage = base.stage_role_inputs
original_preflight = base.preflight
original_trace = base.operational_trace
original_instrument = base.instrument


class GoChessPVChecker(DrawPVChecker):
    def verify(self, history, info, kind, warning_move):
        match = INFO.fullmatch(info)
        if not match:
            raise ValueError('unrecognized exact GoChess info grammar')
        depth, score, nodes, nps, elapsed, _, pv = match.groups()
        canonical = f'info depth {depth} score cp {score} nodes {nodes} time {elapsed} nps {nps} pv {pv}'
        return super().verify(history, canonical, kind, warning_move)


def filter_gochess_draw_warnings(lines, roles):
    return filter_draw_warnings(lines, roles, base.TRACE_RE,
                                artifact_sha256=ENGINE_SHA, checker=GoChessPVChecker())


def is_gochess(role):
    engine = role.get('engine')
    return (engine.get('sha256') if isinstance(engine, dict) else role.get('sha256')) == ENGINE_SHA


def validate_role(role):
    original_validate(role)
    if not is_gochess(role):
        return
    options = dict(role['options'])
    required = {'Threads', 'Hash', 'Ponder', 'OwnBook', 'BookFile', 'UseNNUE', 'NNUEFile', 'MoveOverhead'}
    if not role.get('network') or role['network']['sha256'] != NETWORK_SHA or set(options) != required:
        raise base.CandidateMatchError('exact GoChess role/model/options differ')
    if options['Ponder'] is not False or options['OwnBook'] is not False or options['UseNNUE'] is not True or \
            options['BookFile'] != '<empty>' or options['NNUEFile'] != '$NETWORK':
        raise base.CandidateMatchError('GoChess requires explicit NNUE and book-free options')
    for key, high in (('Hash', 4096), ('MoveOverhead', 1000)):
        low = 1 if key == 'Hash' else 0
        if type(options[key]) is not int or not low <= options[key] <= high:
            raise base.CandidateMatchError('GoChess option outside advertised domain: ' + key)


def check_staged_network(cwd):
    cwd = Path(cwd)
    if (cwd / 'net.txt').read_text() != NET_TXT:
        raise base.CandidateMatchError('staged GoChess net.txt differs')
    for name in ('ngn.nnue', NETWORK_NAME):
        if base.sha256(cwd / name) != NETWORK_SHA:
            raise base.CandidateMatchError('staged GoChess model differs: ' + name)
    return f'NNUE v5 loaded from {cwd / NETWORK_NAME} (fingerprint {FINGERPRINT})'


def stage_role_inputs(role, cwd):
    validate_role(role)
    binary, options = original_stage(role, cwd)
    if is_gochess(role):
        os.link(cwd / 'ngn.nnue', cwd / NETWORK_NAME)
        (cwd / NETWORK_NAME).chmod(0o444)
        (cwd / 'net.txt').write_text(NET_TXT)
        (cwd / 'net.txt').chmod(0o444)
        check_staged_network(cwd)
    return binary, options


class GoChessProtocol(base.Protocol):
    def __init__(self, process, transcript, banner):
        self.banner = banner
        self.allowed_stderr = []
        self.allowed_stderr_kinds = set()
        super().__init__(process, transcript)

    def reader(self, stream, handle):
        if stream != 'stderr':
            return super().reader(stream, handle)
        while raw := handle.readline():
            line = raw.decode('utf-8', errors='replace').rstrip('\r\n')
            timestamp = time.monotonic()
            self.record('engine-stderr', line, timestamp)
            if line == self.banner and not self.allowed_stderr:
                self.allowed_stderr.append(line)
                self.allowed_stderr_kinds.add('gochess_v5_loaded')
                self.events.put(('allowed-startup-stderr', line, timestamp))
            else:
                self.stderr_lines.append(line)
                self.events.put(('stderr', line, timestamp))
        self.events.put(('stderr', '<EOF>', time.monotonic()))

    def until(self, predicate, timeout, label):
        lines = super().until(predicate, timeout, label)
        if any(line.casefold().startswith('info string error') for line in lines):
            raise base.CandidateMatchError('GoChess reported an operational error during ' + label)
        return lines


def validate_preflight_receipt(receipt, transcript, banner, cwd):
    if receipt['uci_identity'] != ['id name GoChess', 'id author Adam']:
        raise base.CandidateMatchError('exact GoChess UCI identity differs')
    if receipt.pop('allowed_exact_counter55_startup_stderr') != [banner]:
        raise base.CandidateMatchError('exact GoChess startup banner absent or duplicated')
    barriers = {row['name']: row for row in receipt['barriers']}
    if barriers['NNUEFile']['lines'].count(FINGERPRINT_LINE) != 1 or \
            barriers['UseNNUE']['lines'].count(USE_LINE) != 1:
        raise base.CandidateMatchError('GoChess model/UseNNUE acknowledgement differs')
    rows = [json.loads(line) for line in Path(transcript).read_text().splitlines()]
    stdout = [row['line'] for row in rows if row['direction'] == 'engine-stdout']
    if any(line.casefold().startswith('info string error') or
           line.startswith('info string UseNNUE = ') and line != USE_LINE for line in stdout):
        raise base.CandidateMatchError('GoChess preflight reported an error or disabled NNUE')
    if stdout.count(FINGERPRINT_LINE) != 2 or \
            any(line.startswith('info string SetNNUEV5 fingerprint ') and line != FINGERPRINT_LINE for line in stdout):
        raise base.CandidateMatchError('GoChess startup and explicit model fingerprints differ')
    check_staged_network(cwd)
    receipt['native_gochess'] = {'engine_sha256': ENGINE_SHA, 'network_sha256': NETWORK_SHA,
                                'fingerprint': FINGERPRINT, 'startup_stderr': [banner]}
    return receipt


def preflight(launcher, cwd, options, role, out, cpu):
    if not is_gochess(role):
        return original_preflight(launcher, cwd, options, role, out, cpu)
    validate_role(role)
    banner = check_staged_network(cwd)
    saved = base.CheckedProtocol
    # ownedmatch runs preflights sequentially; reader instances retain their class.
    base.CheckedProtocol = lambda process, transcript, counter55: GoChessProtocol(process, transcript, banner)
    try:
        receipt = original_preflight(launcher, cwd, options, role, out, cpu)
    finally:
        base.CheckedProtocol = saved
    return validate_preflight_receipt(receipt, out, banner, cwd)


def check_trace(lines, trace_roles, roles):
    go_roles = {role['name']: role for role in roles if is_gochess(role)}
    expected = {role['display_name']: role for role in trace_roles}
    banners = {name: check_staged_network(role['cwd']) for name, role in go_roles.items()}
    counts = {name: Counter() for name in go_roles}
    filtered, accepted = [], []
    for number, line in enumerate(lines, 1):
        match = base.TRACE_RE.fullmatch(line)
        body = match.group('body').strip() if match else ''
        thread = match.group('thread').strip() if match else ''
        for name in go_roles:
            if body == 'fastchess --- Engine ' + name + ' refreshed.':
                counts[name][(thread, 'refresh')] += 1
        if ' ---> ' in body:
            actor, payload = body.split(' ---> ', 1)
            actor, payload = actor.strip(), payload.strip()
            stderr = actor.startswith('<stderr>')
            name = actor[len('<stderr>'):].strip() if stderr else actor
            if name in go_roles:
                tally = counts[name]
                if stderr and payload == banners[name] and match.group('level').strip().upper() in {'TRACE', 'DEBUG', 'INFO', 'ENGINE'}:
                    tally[(thread, 'banner')] += 1
                    accepted.append({'line': number, 'role': name, 'thread': thread, 'payload': payload})
                    continue
                if stderr and payload == 'Process exited normally with status 0':
                    tally[(thread, 'exit')] += 1
                if not stderr:
                    if payload.casefold().startswith('info string error'):
                        raise base.CandidateMatchError('GoChess operational error: ' + payload)
                    if payload.startswith('info string SetNNUEV5 fingerprint '):
                        if payload != FINGERPRINT_LINE:
                            raise base.CandidateMatchError('GoChess trace model fingerprint differs')
                        tally[(thread, 'fingerprint')] += 1
                    if payload.startswith('info string UseNNUE = '):
                        if payload != USE_LINE:
                            raise base.CandidateMatchError('GoChess trace disabled NNUE')
                        tally[(thread, 'use_nnue')] += 1
        filtered.append(line)
    for name, tally in counts.items():
        role = expected[name]
        threads = {thread for thread, _ in tally}
        if sum(tally[(t, 'banner')] for t in threads) != role['expected_processes'] or \
                sum(tally[(t, 'refresh')] for t in threads) != role['expected_refreshes'] or \
                sum(tally[(t, 'exit')] for t in threads) != role['expected_processes']:
            raise base.CandidateMatchError('GoChess trace process/refresh count differs')
        for thread in threads:
            # Fastchess performs final process cleanup from a different thread.
            if tally[(thread, 'fingerprint')] != tally[(thread, 'banner')] + tally[(thread, 'refresh')] or \
                    tally[(thread, 'use_nnue')] != tally[(thread, 'refresh')]:
                raise base.CandidateMatchError('GoChess per-thread model acknowledgements differ')
    return filtered, accepted


def operational_trace(lines, trace_roles, roles):
    drawn, verified_draws = filter_gochess_draw_warnings(lines, roles)
    filtered, accepted = check_trace(drawn, trace_roles, roles)
    report = original_trace(filtered, trace_roles, roles)
    report['verified_exact_gochess_v5_startup_banners'] = accepted
    report['verified_gochess_network_sha256'] = NETWORK_SHA
    report['verified_gochess_model_fingerprint'] = FINGERPRINT
    report['independently_verified_exact_gochess_post_draw_pvs'] = verified_draws
    report['unfiltered_trace_lines'] = len(lines)
    return report


def instrument():
    return dict(original_instrument(), **{Path(__file__).name: base.sha256(Path(__file__)),
        'counter_draw_compat.py': base.sha256(Path(__file__).with_name('counter_draw_compat.py')),
        'python_chess_core_1.11.2': CORE_SHA})


def activate():
    base.validate_one_worker_role = validate_role
    base.stage_role_inputs = stage_role_inputs
    base.preflight = preflight
    base.operational_trace = operational_trace
    base.instrument = instrument


if __name__ == '__main__':
    activate()
    base.main()
