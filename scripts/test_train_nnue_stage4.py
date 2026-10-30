import gzip
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch
import train_nnue_stage3 as stage3

class AdditionalShardTest(unittest.TestCase):
    def test_prepares_at_offset_without_changing_original_holdout(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            shard = root / 'new.gz'
            with gzip.open(shard, 'wt') as f:
                f.write('1\tposition\t100\t1.0\n49\tposition\t-100\t0.0\n51\tposition\t0\t0.5\n')
            output = root / 'prepared'
            output.mkdir()
            (output / 'sample.json').write_text('original')
            with patch.object(stage3, 'features', return_value=([0]*32, [1]*32)):
                stage3.prepare([shard], output, start=11)
            self.assertEqual((output / '11-0-blend.bin').stat().st_size, 12)
            self.assertEqual((output / '11-1-blend.bin').stat().st_size, 0)
            self.assertEqual((output / 'sample.json').read_text(), 'original')
            self.assertTrue((output / 'sample-11.json').exists())

class Stage4TrainingTest(unittest.TestCase):
    def test_training_uses_only_original_holdout(self):
        import torch
        import train_nnue_stage4 as stage4
        if not torch.cuda.is_available():
            self.skipTest('CUDA unavailable')
        with tempfile.TemporaryDirectory() as d:
            root = Path(d); prepared = root/'prepared'; output = root/'output'; output.mkdir()
            fen = '8/8/8/8/8/8/8/K6k w - - 0 1'
            first, second = root/'original.gz', root/'new.gz'
            with gzip.open(first, 'wt') as f:
                f.write(f'1\t{fen}\t100\t1.0\n50\t{fen}\t-100\t0.0\n')
            with gzip.open(second, 'wt') as f:
                f.write(f'1\t{fen}\t200\t1.0\n')
            stage3.prepare([first], prepared)
            stage3.prepare([second], prepared, start=1)
            with patch.object(stage4, 'valid', wraps=stage4.valid) as validate:
                stage4.train(prepared, 1, 2, output, 1)
            self.assertEqual(validate.call_args.args[2], 1)
            self.assertEqual((output/'stage4.raw').stat().st_size, 98689*4)
            self.assertEqual(len((output/'stage4.csv').read_text().splitlines()), 2)
