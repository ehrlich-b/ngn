"""New outcome-stratified deployed-net diagnosis; no training or net selection."""
import collections
import datetime
import json
from pathlib import Path

import numpy as np
import torch
from trainer.ngnp import RECORD_DTYPE, decode_record, encode_record, parse_fen
from trainer.king_buckets import feature_indices
from trainer.refeval import Network

root = Path('/home/ehrli/ngn-data/night-20261007')
p = root / 'session74-clock-panel'
torch.set_num_threads(2)
assert torch.cuda.is_available()
rows = json.loads((p / 'panel.json').read_text())
archive_records = np.fromfile(root / 'session66-data-diagnostic/panel.ngnp', dtype=RECORD_DTYPE)
archive_metadata = json.loads((root / 'session66-data-diagnostic/panel-metadata.json').read_text())
assert len(archive_records) == len(archive_metadata) == 8192
records = []
metadata = []
for row in rows:
    pos = parse_fen(row['fen'])
    result_ngn = {'loss': 0, 'draw': 1, 'win': 2}[row['outcome']]
    result_white = result_ngn if pos.stm == 0 else 2 - result_ngn
    records.append(encode_record(pos, 0, result_white, len(row['moves'])))
    metadata.append(dict(id=row['id'], domain='Rodent-game', game=f"{row['pair_id']}-{'w' if row['ngn_white'] else 'b'}",
                         color='white' if row['ngn_white'] else 'black', outcome=row['outcome'],
                         move_band=row['move_band'], material_stratum=row['material_stratum']))
records.extend(archive_records)
metadata.extend(dict(id=f'archive-{i}', domain='archive-holdout', source=meta['source'],
                     us_king_bucket=meta['us_king_bucket'], material_bucket=meta['material_bucket'])
                for i, meta in enumerate(archive_metadata))
records = np.asarray(records, dtype=RECORD_DTYPE)
net_paths = dict(accepted=root/'session72-see-rig/base.nnue', shelved=root/'session57-data-step2/trained/best.nnue')


def evaluate(net):
    w = torch.from_numpy(net.w1).cuda()
    b = torch.from_numpy(net.b1).cuda()
    o = torch.from_numpy(net.o).cuda()
    bias = torch.from_numpy(np.asarray(net.ob)).cuda()
    scores, saturated = [], []
    with torch.inference_mode():
        for start in range(0, len(records), 128):
            batch = records[start:start+128]
            positions = [decode_record(record) for record in batch]
            indexes = np.zeros((len(batch), 2, 32), dtype=np.int64)
            mask = np.zeros((len(batch), 2, 32), dtype=np.int64)
            for i, pos in enumerate(positions):
                for side in (0, 1):
                    idx = feature_indices(pos, side, net.king_map)
                    indexes[i,side,:len(idx)] = idx
                    mask[i,side,:len(idx)] = 1
            idx = torch.from_numpy(indexes).cuda()
            valid = torch.from_numpy(mask).cuda()
            acc = (w[idx] * valid[...,None]).sum(dim=2) + b
            saturation = ((acc <= 0) | (acc >= net.qa)).float().mean(dim=(1,2))
            activation = acc.clamp(0,net.qa).square()
            stm = torch.tensor([pos.stm for pos in positions], device='cuda')
            count = torch.arange(len(positions), device='cuda')
            us, them = activation[count,stm], activation[count,1-stm]
            buckets = torch.tensor([net.bucket(pos) for pos in positions], device='cuda')
            weights = o[buckets]
            total = (us*weights[:,:net.hidden] + them*weights[:,net.hidden:]).sum(dim=1)
            intermediate = torch.div(total,net.qa,rounding_mode='trunc') + bias[buckets]
            cp = torch.div(intermediate*net.scale,net.qa*net.qb,rounding_mode='trunc').clamp(-24999,24999)
            scores.extend(cp.cpu().tolist())
            saturated.extend(saturation.cpu().tolist())
    result = np.asarray(scores,dtype=np.float64)
    # New GPU instrument admission only, not another quantization experiment.
    for i in (0,len(rows)-1,len(records)-1):
        assert result[i] == net.evaluate_record(records[i]), (i,result[i],net.evaluate_record(records[i]))
    return result,np.asarray(saturated)

values, saturations = {}, {}
for name,path in net_paths.items():
    values[name], saturations[name] = evaluate(Network(path))
assert all(np.isfinite(value).all() for value in values.values())
white_result = records['result'].astype(np.float64)/2
result_stm = np.where(records['stm']==0,white_result,1-white_result)
score_stm = records['score'].astype(np.float64)*(1-2*records['stm'].astype(np.float64))
sigmoid = lambda score: 1/(1+np.exp(-np.clip(score/400,-60,60)))
blended = .75*sigmoid(score_stm)+.25*result_stm


def metrics(indexes, archive=False):
    idx=np.asarray(indexes)
    target = blended[idx] if archive else result_stm[idx]
    return dict(n=len(idx), target='archive blended label' if archive else 'eventual result, descriptive only',
                mean_score_cp={name:float(value[idx].mean()) for name,value in values.items()},
                brier={name:float(np.mean((sigmoid(value[idx])-target)**2)) for name,value in values.items()},
                saturation_share={name:float(value[idx].mean()) for name,value in saturations.items()})

clusters=collections.defaultdict(list)
for i,meta in enumerate(metadata):
    domain=meta['domain'];clusters[('domain',domain)].append(i)
    for key in ('color','outcome','move_band','material_stratum','source','us_king_bucket','material_bucket'):
        if key in meta: clusters[(domain+'/'+key,str(meta[key]))].append(i)
groups={str(key):metrics(ids,key[0].startswith('archive') or key==('domain','archive-holdout')) for key,ids in clusters.items()}
summary=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(), gpu=torch.cuda.get_device_name(),
             outcome_rows=len(rows), archive_rows=len(archive_records), deployed_integer_gpu_parity=True,
             groups=groups, verdict='HOLD: no training treatment automatically admitted',
             limits=['Eventual game outcome is not a ground-truth value at an earlier position.',
                     'Two H512 nets cannot isolate representational capacity from distribution.',
                     'Archive split is occupied-board holdout; game/mirror split provenance is insufficient for pilot admission.',
                     'Outcome panel selection is stratified; its aggregate is not population calibration.',
                     'Compare deep-search residuals after CPU panel; deep NGN remains an owned diagnostic, not ground truth.'],
             automatic_training=False, automatic_net_selection=False)
(p/'gpu-panel-values.json').write_text(json.dumps(dict(metadata=metadata, scores={k:v.tolist() for k,v in values.items()},
                                                       saturation={k:v.tolist() for k,v in saturations.items()}))+'\n')
(p/'gpu-panel-summary.json').write_text(json.dumps(summary,indent=2)+'\n')
print(json.dumps({k:v for k,v in summary.items() if k!='groups'},indent=2))
