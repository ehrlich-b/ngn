"""Fixed dense-versus-sparse complete decode/forward/backward CUDA cost gate."""
import copy
import json
import math
from pathlib import Path
import statistics
import time

import numpy as np
import torch

from trainer.king_buckets import DEFAULT_MAP
from trainer.model import NGNN, decode_gpu
from trainer.tests.synthetic import random_records

torch.set_num_threads(2)
assert torch.cuda.is_available()
root = Path('/home/ehrli/ngn-data/night-20261007/session17s')
started = time.time()
batch = 16384
records = random_records(batch, np.random.default_rng(17013))
raw = torch.from_numpy(records.view(np.uint8).reshape(-1, 32)).cuda()
results = []
for name, factorizer, hidden2 in [('ngnn3', False, 0), ('factorizer', True, 0), ('ngnn4', False, 16)]:
    torch.manual_seed(17014)
    candidate = NGNN(512, 8, DEFAULT_MAP, factorizer, hidden2, True).cuda()
    baseline = copy.deepcopy(candidate)
    models = dict(baseline=baseline, candidate=candidate)
    upstream = torch.linspace(-1, 2, batch, device='cuda') / batch

    def step(kind):
        model = models[kind]
        model.zero_grad(set_to_none=True)
        with torch.autocast(device_type='cuda', dtype=torch.bfloat16):
            features, stm, _, _, bucket = decode_gpu(raw, 8, model.king_map, model.king_buckets,
                                                    sparse=kind == 'candidate')
            output = model(features, stm, bucket)
        output.backward(upstream)

    for kind in models:
        for _ in range(5):
            step(kind)
        models[kind].zero_grad(set_to_none=True)
    torch.cuda.synchronize()
    rounds = []
    for index in range(20):
        order = ['baseline', 'candidate'] if index % 2 == 0 else ['candidate', 'baseline']
        elapsed, memory = {}, {}
        for kind in order:
            for model in models.values():
                model.zero_grad(set_to_none=True)
            torch.cuda.synchronize()
            torch.cuda.reset_peak_memory_stats()
            start, end = torch.cuda.Event(enable_timing=True), torch.cuda.Event(enable_timing=True)
            start.record()
            for _ in range(5):
                step(kind)
            end.record()
            end.synchronize()
            elapsed[kind] = start.elapsed_time(end) / 1000
            memory[kind] = torch.cuda.max_memory_allocated()
        rounds.append(dict(index=index, order=order, seconds=elapsed, peak_bytes=memory,
                           ratio=elapsed['baseline'] / elapsed['candidate']))
    logs = [math.log(row['ratio']) for row in rounds]
    mean = statistics.mean(logs)
    margin = 2.093024054408263 * statistics.stdev(logs) / math.sqrt(20)
    ci = [math.exp(mean - margin), math.exp(mean + margin)]
    strata = [math.exp(statistics.mean(logs[i::2])) for i in range(2)]
    peak = {kind: max(row['peak_bytes'][kind] for row in rounds) for kind in models}
    results.append(dict(name=name, hidden=512, hidden2=hidden2, king_factorizer=factorizer,
                        batch=batch, buckets=8, rounds=rounds, geometric_ratio=math.exp(mean),
                        ci95=ci, order_strata=strata, peak_bytes=peak,
                        passed=ci[0] > 1 and min(strata) > 1 and peak['candidate'] < peak['baseline']))
    del baseline, candidate, models, model, upstream
    torch.cuda.empty_cache()

receipt = dict(completed_seconds=time.time() - started, torch=torch.__version__,
               cuda=torch.version.cuda, gpu=torch.cuda.get_device_name(), precision='bf16',
               iterations_per_block=5, blocks=20, results=results,
               passed=all(row['passed'] for row in results))
(root / 'benchmark.json').write_text(json.dumps(receipt, indent=2, allow_nan=False) + '\n')
print(json.dumps({**receipt, 'results': [{k: v for k, v in row.items() if k != 'rounds'}
                                       for row in results]}, indent=2))
raise SystemExit(0 if receipt['passed'] else 3)
