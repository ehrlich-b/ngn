#!/usr/bin/env python3
"""Regression checks for Chess768 feature indexing in the stage-two trainer."""
import struct
import gzip
import sys
import tempfile
import unittest
import zlib
from pathlib import Path
from unittest.mock import patch
import torch
from plot_loss import plot
import train_nnue_stage2
from train_nnue_stage2 import features


class FeatureTests(unittest.TestCase):
    def test_perspective_and_padding(self):
        white, black = features('4k3/8/8/8/8/8/4P3/4K3 w - - 0 1')
        self.assertEqual(white[:3], [764, 12, 324])
        self.assertEqual(black[:3], [324, 436, 764])
        self.assertEqual(white[3:], [768] * 29)
        self.assertEqual(black[3:], [768] * 29)
        other_white, other_black = features('4k3/8/8/8/8/8/4P3/4K3 b - - 0 1')
        self.assertEqual((other_white, other_black), (black, white))


class LossPlotTests(unittest.TestCase):
    def test_single_epoch_has_both_data_markers(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            (path / 'loss.csv').write_text('epoch,train,validation\n1,0.05,0.04\n')
            plot(path / 'loss.csv', path / 'loss.png')
            png = (path / 'loss.png').read_bytes()
            size = struct.unpack('>I', png[33:37])[0]
            self.assertEqual(png[37:41], b'IDAT')
            pixels = zlib.decompress(png[41:41+size])
            # Both curves have data pixels well below their legends.
            graph = pixels[70 * (900*3+1):445 * (900*3+1)]
            self.assertTrue(bytes((35,100,220))*3 in graph, 'missing training marker')
            self.assertTrue(bytes((220,70,50))*3 in graph, 'missing validation marker')


class DeadlineTests(unittest.TestCase):
    def test_partial_epoch_is_validated_and_exported_before_hard_alarm(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            data = path / 'data.tsv.gz'
            fen = '4k3/8/8/8/8/8/4P3/4K3 w - - 0 1'
            with gzip.open(data, 'wt') as f:
                f.write(f'0\t{fen}\t0\t0.5\n1\t{fen}\t0\t0.5\n2\t{fen}\t0\t0.5\n')
            raw, loss, plot_path = (path / name for name in ('net.raw', 'loss.csv', 'loss.png'))
            args = ['train', str(data), '--raw', str(raw), '--loss', str(loss),
                    '--plot', str(plot_path), '--batch', '1']
            elapsed = [0]
            original_to = torch.Tensor.to
            original_randperm = torch.randperm
            original_step = torch.optim.Adam.step

            def cpu_to(tensor, *args, **kwargs):
                if kwargs.get('device') == 'cuda':
                    kwargs['device'] = 'cpu'
                args = tuple('cpu' if arg == 'cuda' else arg for arg in args)
                return original_to(tensor, *args, **kwargs)

            def step(optimizer, *args, **kwargs):
                result = original_step(optimizer, *args, **kwargs)
                elapsed[0] = 1100
                return result

            with patch.object(sys, 'argv', args), \
                 patch.object(train_nnue_stage2.time, 'monotonic', side_effect=lambda: elapsed[0]), \
                 patch.object(train_nnue_stage2.signal, 'alarm') as alarm, \
                 patch.object(torch.cuda, 'is_available', return_value=True), \
                 patch.object(torch.cuda, 'get_device_name', return_value='CPU test'), \
                 patch.object(torch.cuda, 'is_current_stream_capturing', return_value=False), \
                 patch.object(torch.Tensor, 'to', cpu_to), \
                 patch.object(torch.nn.Module, 'cuda', lambda self: self), \
                 patch.object(torch, 'randperm', side_effect=lambda n, **kw: original_randperm(n)), \
                 patch.object(torch.optim.Adam, 'step', step):
                train_nnue_stage2.main()
            self.assertEqual(alarm.call_args_list[0].args, (1200,))
            self.assertEqual(alarm.call_args_list[-1].args, (0,))
            self.assertTrue(raw.stat().st_size > 0)
            self.assertEqual(len(loss.read_text().splitlines()), 2)
            self.assertTrue(plot_path.stat().st_size > 0)


if __name__ == '__main__':
    unittest.main()
