"""Compare the NGNN4 GEMM path with per-position heads and their gradients."""

import copy
import unittest

import torch

from trainer.king_buckets import DEFAULT_MAP
from trainer.model import NGNN


def gathered_forward(model, features, stm, bucket):
    white, black = (features @ model.effective_w1() + model.b1).clamp(0, 1).chunk(2)
    us = torch.where((stm == 0)[:, None], white, black)
    them = torch.where((stm == 0)[:, None], black, white)
    ordered = torch.cat((us, them), 1)
    middle = torch.bmm(model.w2[bucket], ordered.unsqueeze(2)).squeeze(2)
    activation = (middle.float() + model.b2[bucket]).clamp(0, 1).square()
    bias = model.ob if model.buckets == 1 else model.ob[bucket]
    return ((activation * model.o[bucket]).sum(1).float() + bias) * 400


class NGNN4BatchedTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        torch.set_num_threads(2)

    def check_gradients(self, device, amp=False):
        for buckets in (1, 2, 4, 8):
            for hidden2 in (16, 32):
                torch.manual_seed(16008)
                model = NGNN(32, buckets, DEFAULT_MAP, hidden2=hidden2).to(device)
                reference = copy.deepcopy(model)
                # Sparse synthetic features, mixed STM, shuffled heads and an
                # omitted head check the whole path and zero unselected grads.
                features = torch.zeros(2 * 19, 6144, device=device)
                indices = torch.randint(6144, (2 * 19, 12), device=device)
                features.scatter_(1, indices, 1)
                stm = torch.arange(19, device=device) % 2
                bucket = (torch.arange(19, device=device) * 5 + 2) % max(1, buckets - 1)
                with torch.autocast(device_type=device, dtype=torch.bfloat16, enabled=amp):
                    actual = model(features, stm, bucket)
                    expected = gathered_forward(reference, features, stm, bucket)
                torch.testing.assert_close(actual, expected,
                                           rtol=.015 if amp else 2e-5,
                                           atol=.025 if amp else 2e-5)
                # Signed, nonuniform upstream gradients catch head/order errors
                # that can cancel under a plain sum loss.
                upstream = torch.linspace(-1, 2, 19, device=device)
                actual.backward(upstream)
                expected.backward(upstream)
                for name, parameter in model.named_parameters():
                    other = dict(reference.named_parameters())[name]
                    torch.testing.assert_close(parameter.grad, other.grad,
                                               rtol=.025 if amp else 3e-5,
                                               atol=.1 if amp else 3e-5,
                                               msg=lambda m: name + ': ' + m)
                if buckets > 1:
                    self.assertEqual(model.w2.grad[-1].count_nonzero().item(), 0)
                optimizer = torch.optim.AdamW(model.parameters(), lr=.001, fused=device == 'cuda')
                optimizer.step()
                model.clip_quantized()
                self.assertTrue(all(torch.isfinite(p).all() for p in model.parameters()))

    def test_cpu_forward_backward_all_bucket_counts(self):
        self.check_gradients('cpu')

    @unittest.skipUnless(torch.cuda.is_available(), 'CUDA unavailable')
    def test_cuda_fp32_bf16_forward_backward_all_bucket_counts(self):
        self.check_gradients('cuda')
        self.check_gradients('cuda', True)


if __name__ == '__main__':
    unittest.main()
