import gzip
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import torch
import train_nnue_stage3 as stage3

class StreamPreparationTest(unittest.TestCase):
    def test_separate_games_and_count_disagreements(self):
        fen = '8/8/8/8/8/8/8/K6k w - - 0 1'
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            shard = root/'shard.gz'
            with gzip.open(shard,'wt') as f:
                f.write(f'0\t{fen}\t100\t0.0\n0\t{fen}\t-100\t1.0\n1\t{fen}\t100\t1.0\n')
            with patch.object(stage3, 'features', return_value=([0]*32,[1]*32)):
                stage3.prepare([shard],root/'out')
            self.assertEqual((root/'out/0-1-blend.bin').stat().st_size, 8)
            self.assertEqual((root/'out/0-0-blend.bin').stat().st_size, 4)
            stats=json.loads((root/'out/sample.json').read_text())
            self.assertEqual((stats['positions'],stats['games'],stats['sign_disagreements'],stats['decisive_nonzero_positions']),(3,2,2,3))

    def test_forward_supports_both_network_sizes(self):
        features = torch.zeros((2, 32), dtype=torch.int64)
        for hidden in (128, 256):
            with self.subTest(hidden=hidden):
                result = stage3.Net(hidden)(features, features)
                self.assertEqual(tuple(result.shape), (2,))
                self.assertTrue(torch.isfinite(result).all().item())

if __name__ == '__main__': unittest.main()
