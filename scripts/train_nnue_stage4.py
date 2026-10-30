#!/usr/bin/env python3
"""Train on the original game split plus new games, validating only on the original holdout."""
import argparse
import csv
import math
from pathlib import Path
import struct
import torch
from train_nnue_stage3 import Net, mapped, valid


def train(root, original, total, output, epochs):
    if original < 1 or total <= original:
        raise ValueError('need original and new shards')
    torch.set_num_threads(2)
    print(f'original_shards={original} total_shards={total} units=128 learning_rate=0.0005 schedule=cosine device={torch.cuda.get_device_name()}', flush=True)
    torch.manual_seed(1030)
    net = Net(128).cuda()
    opt = torch.optim.Adam(net.parameters(), lr=.0005)
    scheduler = torch.optim.lr_scheduler.CosineAnnealingLR(opt, T_max=epochs)
    best = math.inf
    curve = []
    values = None
    for epoch in range(epochs):
        net.train(); loss_sum = 0.; seen = 0
        for i in range(total):
            u, v, y = mapped(root, i, 0, 'blend')
            for ix in torch.randperm(len(y), device='cuda').split(4096):
                opt.zero_grad(set_to_none=True)
                loss = (net(u[ix], v[ix])-y[ix]).square().mean()
                loss.backward(); opt.step()
                loss_sum += loss.item()*len(ix); seen += len(ix)
        val = valid(net, root, original)
        scheduler.step()
        curve.append((epoch+1, loss_sum/seen, val))
        print(f'epoch={epoch+1} train={loss_sum/seen:.8f} validation={val:.8f} seen={seen}', flush=True)
        if val < best:
            best = val
            values = torch.cat((net.embed.weight[:768].flatten(), net.bias, net.output.weight.flatten(), net.output.bias)).detach().cpu().tolist()
    with open(output/'stage4.raw', 'xb') as f:
        f.write(struct.pack('<%df' % len(values), *values))
    with open(output/'stage4.csv', 'x', newline='') as f:
        writer = csv.writer(f); writer.writerow(('epoch', 'train', 'validation')); writer.writerows(curve)
    print(f'best={best:.8f}', flush=True)


def main():
    p = argparse.ArgumentParser()
    p.add_argument('--root', type=Path, required=True)
    p.add_argument('--original', type=int, required=True)
    p.add_argument('--total', type=int, required=True)
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--epochs', type=int, default=10)
    a = p.parse_args()
    train(a.root, a.original, a.total, a.output, a.epochs)


if __name__ == '__main__':
    main()
