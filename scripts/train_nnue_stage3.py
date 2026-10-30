#!/usr/bin/env python3
"""Stream stage-two game shards into bounded scratch tensors and run ablations."""
import argparse
from array import array
import csv
import gzip
import json
import math
from pathlib import Path
import struct
import time

import torch
from torch import nn
from train_nnue_stage2 import features

SPECS = [('lr002',128,.002,False), ('lr0005',128,.0005,False),
         ('hidden256',256,.002,False), ('search',128,.002,True)]

def prepare(paths, root, start=0):
    root.mkdir(parents=True, exist_ok=True)
    sample_positions = sample_games = disagree = decisive = capped_positions = capped_games = 0
    for number, path in enumerate(paths, start):
        rows = [(array('H'),array('H'),array('f'),array('f')) for _ in range(2)]
        last = None; count = 0; result = None; game_scores = []
        def sample_game():
            nonlocal sample_positions, sample_games, disagree, decisive, capped_positions, capped_games
            if sample_positions >= 200000 or not game_scores: return
            sample_games += 1; sample_positions += len(game_scores)
            for score, outcome in game_scores:
                if score != 0 and outcome != .5:
                    decisive += 1
                    disagree += (score > 0) != (outcome > .5)
            if len(game_scores) == 192 and result == .5:
                capped_games += 1; capped_positions += len(game_scores)
        with gzip.open(path, 'rt') as src:
            for line in src:
                game, fen, score, outcome = line.rstrip().split('\t')
                if last is not None and game != last:
                    sample_game(); game_scores = []
                last = game; score = max(-4000,min(4000,int(score))); outcome = float(outcome); result = outcome
                game_scores.append((score,outcome))
                search = 1/(1+math.exp(-math.log(10)*score/400))
                u,v = features(fen); row = rows[int(int(game)%50 == 0)]
                row[0].fromlist(u); row[1].fromlist(v); row[2].append((search+outcome)/2); row[3].append(search)
                count += 1
        sample_game()
        for split, row in enumerate(rows):
            for name, data in zip(('u','v','blend','search'),row):
                with open(root/f'{number}-{split}-{name}.bin','xb') as f: data.tofile(f)
        print(f'shard={path} positions={count} train={len(rows[0][2])} holdout={len(rows[1][2])}',flush=True)
    stats = dict(positions=sample_positions,games=sample_games,decisive_nonzero_positions=decisive,
                 sign_disagreements=disagree,capped_games=capped_games,capped_positions=capped_positions)
    (root/('sample.json' if start == 0 else f'sample-{start}.json')).write_text(json.dumps(stats,indent=2)+'\n')
    print('labels:',json.dumps(stats),flush=True)

class Net(nn.Module):
    def __init__(self, h):
        super().__init__(); self.embed=nn.Embedding(769,h,padding_idx=768)
        nn.init.uniform_(self.embed.weight,-.02,.02)
        with torch.no_grad(): self.embed.weight[768].zero_()
        self.bias=nn.Parameter(torch.full((h,),.1)); self.output=nn.Linear(2*h,1)
    def forward(self,u,v):
        x=torch.cat(((self.embed(u.long()).sum(1)+self.bias).clamp(0,1).square(),
                     (self.embed(v.long()).sum(1)+self.bias).clamp(0,1).square()),1)
        return torch.sigmoid(self.output(x).squeeze(1)*math.log(10))

def mapped(root, i, split, target):
    size=(root/f'{i}-{split}-blend.bin').stat().st_size//4
    def read(name,dtype,n):
        return torch.from_file(str(root/f'{i}-{split}-{name}.bin'),size=n,dtype=dtype)
    return (read('u',torch.uint16,size*32).view(-1,32).to(device='cuda',dtype=torch.int32),
            read('v',torch.uint16,size*32).view(-1,32).to(device='cuda',dtype=torch.int32),
            read(target,torch.float32,size).to('cuda'))

def valid(net,root,n):
    loss=0.; count=0
    net.eval()
    with torch.no_grad():
        for i in range(n):
            u,v,y=mapped(root,i,1,'blend')
            for a,b,c in zip(u.split(4096),v.split(4096),y.split(4096)):
                loss += (net(a,b)-c).square().sum().item(); count += len(c)
    return loss/count

class Quantized(nn.Module):
    def __init__(self,path):
        super().__init__(); data=Path(path).read_bytes(); assert len(data)==197506 and data[:8]==b'NGNNUE\0\0'
        vals=torch.tensor(struct.unpack('<98689h',data[128:]),device='cuda',dtype=torch.int64)
        self.embed=torch.cat((vals[:768*128].reshape(768,128),torch.zeros((1,128),device='cuda',dtype=torch.int64)))
        j=768*128; self.bias=vals[j:j+128]; j+=128; self.output=vals[j:j+256]; self.outbias=vals[-1]
    def forward(self,u,v):
        a=(self.embed[u.long()].sum(1)+self.bias).clamp(0,255).square()
        b=(self.embed[v.long()].sum(1)+self.bias).clamp(0,255).square()
        dot=(a*self.output[:128]).sum(1)+(b*self.output[128:]).sum(1)
        x=torch.div(dot,255,rounding_mode='trunc')+self.outbias
        cp=torch.div(x*400,255*64,rounding_mode='trunc')
        return torch.sigmoid(cp.float()*(math.log(10)/400))

def train(root,n,out,epochs):
    out.mkdir(parents=True,exist_ok=True)
    for name,h,lr,search in SPECS:
        torch.manual_seed(1030); net=Net(h).cuda(); opt=torch.optim.Adam(net.parameters(),lr=lr)
        scheduler=torch.optim.lr_scheduler.CosineAnnealingLR(opt,T_max=epochs)
        curve=[]; best=float('inf'); best_values=None
        for epoch in range(epochs):
            net.train(); total=0.; seen=0
            for i in range(n):
                u,v,y=mapped(root,i,0,'search' if search else 'blend')
                perm=torch.randperm(len(y),device='cuda')
                for ix in perm.split(4096):
                    opt.zero_grad(set_to_none=True); loss=(net(u[ix],v[ix])-y[ix]).square().mean(); loss.backward(); opt.step()
                    total += loss.item()*len(ix); seen += len(ix)
            val=valid(net,root,n); curve.append((total/seen,val)); scheduler.step()
            print(f'{name} epoch={epoch+1} train={total/seen:.8f} validation={val:.8f} seen={seen}',flush=True)
            if val<best:
                best=val
                best_values=torch.cat((net.embed.weight[:768].flatten(),net.bias,net.output.weight.flatten(),net.output.bias)).detach().cpu().tolist()
            with open(out/f'{name}.csv','w',newline='') as f:
                writer=csv.writer(f); writer.writerow(('epoch','train','validation')); writer.writerows((j,*x) for j,x in enumerate(curve,1))
        with open(out/f'{name}.raw','xb') as f: f.write(struct.pack('<%df'%len(best_values),*best_values))
        print(f'{name} best={best:.8f}',flush=True)

def main():
    p=argparse.ArgumentParser(); p.add_argument('action',choices=['prepare','baseline','train']); p.add_argument('--root',type=Path,required=True); p.add_argument('--manifest',type=Path,required=True); p.add_argument('--output',type=Path); p.add_argument('--net',type=Path); p.add_argument('--epochs',type=int,default=10); p.add_argument('--start',type=int,default=0)
    a=p.parse_args(); paths=[s['path'] for s in json.loads(a.manifest.read_text())['shards']]
    torch.set_num_threads(2)
    if a.action=='prepare': prepare(paths,a.root,a.start)
    elif a.action=='baseline': print(f'exported net holdout blend MSE={valid(Quantized(a.net).cuda(),a.root,len(paths)):.8f}',flush=True)
    else: train(a.root,len(paths),a.output,a.epochs)
if __name__=='__main__': main()
