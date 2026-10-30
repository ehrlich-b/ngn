#!/usr/bin/env python3
"""Regression checks for Chess768 feature indexing in the stage-two trainer."""
import struct
import tempfile
import unittest
import zlib
from pathlib import Path
from plot_loss import plot
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


if __name__ == '__main__':
    unittest.main()
