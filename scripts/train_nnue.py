#!/usr/bin/env python3
"""Train an NGN Chess768 network on nnuedata TSV and export Bullet raw floats."""
import argparse
import math
import random
import struct

import torch
from torch import nn

P = argparse.ArgumentParser()
P.add_argument('data')
P.add_argument('--raw', required=True)
P.add_argument('--loss', required=True)
P.add_argument('--plot', required=True)
P.add_argument('--epochs', type=int, default=25)
P.add_argument('--seed', type=int, default=1030)
a = P.parse_args()
random.seed(a.seed)
torch.manual_seed(a.seed)
torch.set_num_threads(2)
if not torch.cuda.is_available():
    raise RuntimeError('CUDA is required')
print('device:', torch.cuda.get_device_name())

def features(fen):
    board, side, *_ = fen.split()
    pieces = []
    for rank, row in enumerate(board.split('/')):
        file = 0
        for c in row:
            if c.isdigit():
                file += int(c)
                continue
            piece = 'pnbrqk'.index(c.lower())
            colour = int(c.islower())
            sq = (7-rank)*8+file
            pieces.append((piece, colour, sq))
            file += 1
    stm = int(side == 'b')
    def perspective(p):
        ids = [(colour ^ p)*384 + piece*64 + (sq ^ (56 if p else 0)) for piece, colour, sq in pieces]
        return ids + [768]*(32-len(ids))
    return perspective(stm), perspective(stm ^ 1)

rows = [[], []]
for line in open(a.data):
    game, fen, score, outcome = line.rstrip().split('\t')
    score = max(-4000, min(4000, int(score)))
    target = 0.5 * (1/(1+math.exp(-math.log(10)*score/400)) + float(outcome))
    u, v = features(fen)
    rows[int(game) % 10 == 0].append((u, v, target))
if not all(rows):
    raise ValueError('empty training or validation split')
def tensors(rows):
    return (torch.tensor([r[0] for r in rows], device='cuda'),
            torch.tensor([r[1] for r in rows], device='cuda'),
            torch.tensor([r[2] for r in rows], device='cuda'))
train, valid = map(tensors, rows)
print('positions train/valid:', len(rows[0]), len(rows[1]))
class Net(nn.Module):
    def __init__(self):
        super().__init__()
        self.embed = nn.Embedding(769, 128, padding_idx=768)
        nn.init.uniform_(self.embed.weight, -0.02, 0.02)
        with torch.no_grad(): self.embed.weight[768].zero_()
        self.bias = nn.Parameter(torch.full((128,), 0.1))
        self.output = nn.Linear(256, 1)
    def forward(self, u, v):
        x = torch.cat(((self.embed(u).sum(1)+self.bias).clamp(0,1).square(),
                       (self.embed(v).sum(1)+self.bias).clamp(0,1).square()), 1)
        return torch.sigmoid(self.output(x).squeeze(1)*math.log(10))
net = Net().cuda()
opt = torch.optim.Adam(net.parameters(), lr=0.002)
curve=[]
best=float('inf')
for epoch in range(a.epochs):
    net.train(); perm=torch.randperm(len(train[2]), device='cuda'); total=0
    for ix in perm.split(1024):
        opt.zero_grad(set_to_none=True)
        loss=(net(train[0][ix],train[1][ix])-train[2][ix]).square().mean()
        loss.backward(); opt.step(); total += loss.item()*len(ix)
    net.eval()
    with torch.no_grad():
        val=sum((net(u,v)-y).square().sum().item() for u,v,y in zip(valid[0].split(2048), valid[1].split(2048), valid[2].split(2048)))/len(valid[2])
    tr=total/len(train[2]); curve.append((tr,val)); print(f'epoch={epoch+1} train={tr:.8f} validation={val:.8f}', flush=True)
    if val < best:
        best=val
        # Bullet raw: feature-major, then feature bias, output weights and bias.
        values = torch.cat((net.embed.weight[:768].flatten(), net.bias, net.output.weight.flatten(), net.output.bias)).detach().cpu().tolist()
        with open(a.raw, 'wb') as f: f.write(struct.pack('<%df'%len(values), *values))
with open(a.loss,'w') as f:
    f.write('epoch,train,validation\n')
    for i,(tr,val) in enumerate(curve,1): f.write(f'{i},{tr:.8f},{val:.8f}\n')
from plot_loss import plot
plot(a.loss, a.plot)
