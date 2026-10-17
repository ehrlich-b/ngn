"""Root relaunch: the held width8-vs4 probe loaded the unfixed session87 probe (-maxmoves 2, zero searched plies).

Uses the fixed session88 probe in a fresh directory. Zero scored games existed. Same jobs, A/A, decision rule.
"""
import importlib.util
from pathlib import Path

R = Path('/home/ehrli/ngn-data/night-20261007')
D = R / 'session94'
spec = importlib.util.spec_from_file_location('chain', D / 'claim-chain.py')
chain = importlib.util.module_from_spec(spec)
spec.loader.exec_module(chain)

held = chain.read(D / 'claim-chain-result.json')
assert held['state'] == 'HELD_REVIEW' and 'peak_5s_cpu_wall_ratio' in held['error'], held
attempt = Path('/home/ehrli/ngn-data/claim-v4-20261016/attempt-001')
assert (attempt / 'screen/analysis.json').exists()
assert not (attempt / 'width8-vs4/aa').exists() and not (attempt / 'width8-vs4/measurement').exists()
(D / 'claim-chain-result.json').rename(D / 'claim-chain-result-held-probe.json')

original = chain.load_module


def load_module(name, path):
    if name == 'width_probe':
        path = R / 'session88/width-probe.py'
    return original(name, path)


chain.load_module = load_module
retry = attempt / 'width-retry1'
retry.mkdir()
candidate = Path(chain.read(D / 'selected-package.json')['identity'])
runner = chain.load_module('queue', R / 'session93/campaign_queue.py')
chain.status('root-width-retry', dict(reason='session87 probe passed -maxmoves 2, zero searched plies; fixed session88 probe; zero scored games', directory=str(retry)))
try:
    ok = chain.width_check(candidate, retry, runner)
except BaseException as error:
    chain.write(D / 'claim-chain-result.json', dict(state='HELD_REVIEW', error=str(error), at_utc=chain.now()))
    chain.status('held-review', str(error))
    raise
if not ok:
    chain.write(D / 'claim-chain-result.json', dict(state='NEEDS_SMP_PROFILE', width=str(retry),
                next='Profile Threads8 at this natural boundary before code;no tune/trainer launched', at_utc=chain.now()))
    chain.status('needs-smp-profile', str(retry))
else:
    chain.resume_work()
    chain.write(D / 'claim-chain-result.json', dict(state='FAILED_CLAIM_WORK_RESUMED', attempt=str(attempt), width=str(retry), finished_utc=chain.now()))
