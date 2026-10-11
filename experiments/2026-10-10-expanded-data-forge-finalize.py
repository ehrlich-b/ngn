"""Finalize the unchanged rig training on forge after copying its terminal files.

Run inside the NGN forge allocation with the source on PYTHONPATH. This is a
manual post-training stage, never a trainer restart or automatic game launch.
Arguments: run directory, engine source, 20-record panel, optional terminal
receipt filename (defaults to the original session47 receipt).
"""
import datetime
import hashlib
import json
import math
import os
from pathlib import Path
import subprocess
import sys

import numpy as np
import torch
from trainer.export import export_state
from trainer.ngnp import RECORD_DTYPE, decode_record, to_fen
from trainer.refeval import Network

s=Path(sys.argv[1]).resolve()
source=Path(sys.argv[2]).resolve()
records_path=Path(sys.argv[3]).resolve()
out=s/'trained'
sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
terminal_name=sys.argv[4] if len(sys.argv)>4 else 'training-terminal-session47.json'
terminal=json.loads((s/terminal_name).read_text())
assert terminal['full58'] and terminal['postprocessing_machine']=='forge'
history=json.loads((out/'history.json').read_text())
assert [r['epoch'] for r in history]==list(range(1,59))
assert all(r['positions']==256000000 and math.isfinite(r['validation_loss']) for r in history)
torch.set_num_threads(2)
checkpoint=torch.load(out/'best.pt',map_location='cpu',weights_only=True)
best=min(history,key=lambda r:r['validation_loss'])
assert checkpoint['epoch']==best['epoch']
config=checkpoint['config']
assert (config['hidden'],config['buckets'],config['format'],config['king_factorizer'])==(512,8,3,False)
export_state(checkpoint['model'],out/'best-reexport.nnue',3)
assert (out/'best-reexport.nnue').read_bytes()==(out/'best.nnue').read_bytes()
network=Network(out/'best.nnue')
records=np.fromfile(records_path,dtype=RECORD_DTYPE)
assert len(records)==20
cases=[]
for record in records:
 position=decode_record(record)
 cases.append(dict(fen=to_fen(position),eval_cp=network.evaluate(position),bucket=network.bucket(position)))
fixture=dict(format='NGNN3',hidden=network.hidden,buckets=network.buckets,king_buckets=network.king_buckets,network_sha256=sha(out/'best.nnue'),cases=cases)
(out/'parity.json').write_text(json.dumps(fixture,indent=2)+'\n')
env=dict(os.environ,NGN_NGNN3_PARITY_NET=str(out/'best.nnue'),NGN_NGNN3_PARITY_EVALS=str(out/'parity.json'))
with (out/'parity.log').open('w') as log:
 subprocess.run(['go','test','-p','2','-count=1','./engine','-run','^TestNGNN3TrainerParity$','-v'],cwd=source,env=env,stdout=log,stderr=subprocess.STDOUT,check=True,timeout=600)
receipt=dict(completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),training_machine='rig WSL2',postprocessing_machine='forge',epochs=58,total_positions=sum(r['positions'] for r in history),best_epoch=best['epoch'],best=best,reexport_identical=True,parity_cases=20,net_sha256=sha(out/'best.nnue'),checkpoint_sha256=sha(out/'best.pt'),archive_split=config['archive_split'],automatic_games=False)
(s/'training-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
(s/'postprocess.exit').write_text('0\n')
(s/'postprocess.done').write_text(receipt['completed_utc']+'\n')
print(json.dumps(receipt))
