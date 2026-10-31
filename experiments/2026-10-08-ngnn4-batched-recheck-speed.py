"""Fixed complete-forward/backward CUDA benchmark for the NGNN4 topic."""
import copy
import json
import math
from pathlib import Path
import statistics
import time

import torch

from trainer.king_buckets import DEFAULT_MAP
from trainer.model import NGNN
from trainer.tests.test_ngnn4_batched import gathered_forward

torch.set_num_threads(2)
assert torch.cuda.is_available()
root = Path('/home/ehrli/ngn-data/night-20261007/session17b')
started = time.time()
results = []
batch = 16384
for hidden in (256, 512):
    for hidden2 in (16, 32):
        torch.manual_seed(16009)
        candidate = NGNN(hidden, 8, DEFAULT_MAP, hidden2=hidden2).cuda()
        baseline = copy.deepcopy(candidate)
        features = torch.zeros(2 * batch, 6144, device='cuda')
        indices = torch.randint(6144, (2 * batch, 24), device='cuda')
        features.scatter_(1, indices, 1)
        stm = torch.arange(batch, device='cuda') % 2
        bucket = (torch.arange(batch, device='cuda') * 5 + 1) % 8
        upstream = torch.linspace(-1, 2, batch, device='cuda') / batch
        models = dict(baseline=baseline, candidate=candidate)

        def step(kind):
            model = models[kind]
            model.zero_grad(set_to_none=True)
            with torch.autocast(device_type='cuda', dtype=torch.bfloat16):
                output = (gathered_forward(model, features, stm, bucket) if kind == 'baseline'
                          else model(features, stm, bucket))
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
            rounds.append(dict(index=index, order=order, seconds=elapsed,
                               peak_bytes=memory, ratio=elapsed['baseline'] / elapsed['candidate']))
        logs = [math.log(row['ratio']) for row in rounds]
        mean = statistics.mean(logs)
        margin = 2.093024054408263 * statistics.stdev(logs) / math.sqrt(20)
        ci = [math.exp(mean - margin), math.exp(mean + margin)]
        strata = [math.exp(statistics.mean(logs[i::2])) for i in range(2)]
        peak = {kind: max(row['peak_bytes'][kind] for row in rounds) for kind in models}
        results.append(dict(hidden=hidden, hidden2=hidden2, batch=batch, buckets=8, rounds=rounds,
                            geometric_ratio=math.exp(mean), ci95=ci, order_strata=strata,
                            peak_bytes=peak,
                            passed=ci[0] > 1 and min(strata) > 1 and peak['candidate'] < peak['baseline']))
        del baseline, candidate, models, model, features, indices, stm, bucket, upstream
        torch.cuda.empty_cache()

receipt = dict(completed_seconds=time.time() - started, torch=torch.__version__,
               cuda=torch.version.cuda, gpu=torch.cuda.get_device_name(), precision='bf16',
               iterations_per_block=5, blocks=20, results=results,
               passed=all(row['passed'] for row in results))
(root / 'benchmark.json').write_text(json.dumps(receipt, indent=2, allow_nan=False) + '\n')
print(json.dumps({**receipt, 'results': [{k: v for k, v in row.items() if k != 'rounds'}
                                       for row in results]}, indent=2))
raise SystemExit(0 if receipt['passed'] else 3)
