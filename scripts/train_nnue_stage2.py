#!/usr/bin/env python3
"""Train NGN-v1 from gzip self-play shards with a game-level 2% holdout."""
import argparse
from array import array
import gzip
import math
import random
import signal
import struct
import time

import torch
from torch import nn


def features(fen):
    board, side, *_ = fen.split()
    pieces = []
    for rank, row in enumerate(board.split('/')):
        file = 0
        for c in row:
            if c.isdigit():
                file += int(c)
            else:
                pieces.append(('pnbrqk'.index(c.lower()), int(c.islower()), (7-rank)*8+file))
                file += 1
    stm = int(side == 'b')
    def perspective(p):
        ids = [(colour ^ p)*384 + piece*64 + (sq ^ (56 if p else 0)) for piece, colour, sq in pieces]
        if len(ids) > 32:
            raise ValueError('more than 32 pieces')
        return ids + [768]*(32-len(ids))
    return perspective(stm), perspective(stm ^ 1)


def main():
    p = argparse.ArgumentParser()
    p.add_argument('data', nargs='+')
    p.add_argument('--raw', required=True)
    p.add_argument('--loss', required=True)
    p.add_argument('--plot', required=True)
    p.add_argument('--epochs', type=int, default=10)
    p.add_argument('--seed', type=int, default=1030)
    p.add_argument('--batch', type=int, default=4096)
    p.add_argument('--seconds', type=int, default=1100)
    a = p.parse_args()
    random.seed(a.seed)
    torch.manual_seed(a.seed)
    torch.set_num_threads(2)
    # Compact arrays keep 15M positions within host and GPU memory. Split by
    # game, not position, so positions from one game never cross the holdout.
    preparation_start = time.monotonic()
    rows = [(array('H'), array('H'), array('f')) for _ in range(2)]
    for path in sorted(a.data):
        count = 0
        with gzip.open(path, 'rt') as src:
            for line in src:
                game, fen, score, outcome = line.rstrip().split('\t')
                score = max(-4000, min(4000, int(score)))
                target = 0.5 * (1/(1+math.exp(-math.log(10)*score/400)) + float(outcome))
                u, v = features(fen)
                dst = rows[int(game) % 50 == 0]
                dst[0].fromlist(u); dst[1].fromlist(v); dst[2].append(target)
                count += 1
        print(f'shard={path} positions={count}', flush=True)
    if not all(r[2] for r in rows):
        raise ValueError('empty training or validation split')
    print('positions train/valid:', len(rows[0][2]), len(rows[1][2]), flush=True)
    print(f'preparation_seconds={time.monotonic()-preparation_start:.1f}', flush=True)
    def tensors(r):
        return (torch.frombuffer(r[0], dtype=torch.uint16).reshape(-1,32).to(device='cuda', dtype=torch.int32),
                torch.frombuffer(r[1], dtype=torch.uint16).reshape(-1,32).to(device='cuda', dtype=torch.int32),
                torch.frombuffer(r[2], dtype=torch.float32).to('cuda'))
    # Hard wall-clock cap includes transfer, training, validation and export.
    def timeout_handler(signum, frame):
        raise TimeoutError('GPU time budget exhausted')
    signal.signal(signal.SIGALRM, timeout_handler)
    signal.alarm(1100)
    start = time.monotonic()
    if not torch.cuda.is_available():
        raise RuntimeError('CUDA is required')
    print('device:', torch.cuda.get_device_name(), flush=True)
    train, valid = map(tensors, rows)
    class Net(nn.Module):
        def __init__(self):
            super().__init__()
            self.embed = nn.Embedding(769, 128, padding_idx=768)
            nn.init.uniform_(self.embed.weight, -0.02, 0.02)
            with torch.no_grad(): self.embed.weight[768].zero_()
            self.bias = nn.Parameter(torch.full((128,), 0.1))
            self.output = nn.Linear(256, 1)
        def forward(self, u, v):
            x = torch.cat(((self.embed(u.long()).sum(1)+self.bias).clamp(0,1).square(),
                           (self.embed(v.long()).sum(1)+self.bias).clamp(0,1).square()), 1)
            return torch.sigmoid(self.output(x).squeeze(1)*math.log(10))
    net = Net().cuda()
    opt = torch.optim.Adam(net.parameters(), lr=0.002)
    curve = []
    best = float('inf')
    for epoch in range(a.epochs):
        net.train(); perm = torch.randperm(len(train[2]), device='cuda'); total = 0; seen = 0
        for ix in perm.split(a.batch):
            if time.monotonic() - start >= a.seconds:
                print('time limit reached during epoch; validating partial checkpoint', flush=True)
                break
            opt.zero_grad(set_to_none=True)
            loss = (net(train[0][ix], train[1][ix])-train[2][ix]).square().mean()
            loss.backward(); opt.step(); total += loss.item()*len(ix); seen += len(ix)
        else:
            ix = None
        partial = ix is not None
        if not seen:
            break
        net.eval()
        with torch.no_grad():
            val = sum((net(u,v)-y).square().sum().item() for u,v,y in zip(valid[0].split(a.batch), valid[1].split(a.batch), valid[2].split(a.batch)))/len(valid[2])
        tr = total/seen; curve.append((tr,val))
        print(f'epoch={epoch+1} train={tr:.8f} validation={val:.8f} positions_seen={seen} seconds={time.monotonic()-start:.1f}', flush=True)
        if val < best:
            best = val
            values = torch.cat((net.embed.weight[:768].flatten(), net.bias, net.output.weight.flatten(), net.output.bias)).detach().cpu().tolist()
            with open(a.raw, 'wb') as f: f.write(struct.pack('<%df' % len(values), *values))
        with open(a.loss, 'w') as f:
            f.write('epoch,train,validation\n')
            for i,(tr,val) in enumerate(curve,1): f.write(f'{i},{tr:.8f},{val:.8f}\n')
        from plot_loss import plot
        plot(a.loss, a.plot)
        if partial or time.monotonic()-start >= a.seconds:
            break
    print(f'training_seconds={time.monotonic()-start:.1f}', flush=True)
    signal.alarm(0)


if __name__ == '__main__':
    main()
