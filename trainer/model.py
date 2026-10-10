"""NGN float model and GPU NGNP1 decoding, derived from the shared spec."""

import torch
from torch import nn

from trainer.king_buckets import validate_map

QA, QB, SCALE = 255, 64, 400


def validate_hidden(hidden: int) -> None:
    if not 16 <= hidden <= 2048 or hidden % 16:
        raise ValueError("Hidden size must be a multiple of 16 in [16, 2048]")


def validate_buckets(buckets: int) -> None:
    if buckets not in (1, 2, 4, 8):
        raise ValueError("Buckets must be one of 1, 2, 4, 8")


def validate_hidden2(hidden2: int) -> None:
    if hidden2 and (not 16 <= hidden2 <= 64 or hidden2 % 16):
        raise ValueError("Second hidden size must be 0 or a multiple of 16 in [16,64]")


def decode_gpu(raw: torch.Tensor, buckets: int | None = None,
               king_map: torch.Tensor | None = None, king_buckets: int = 8) -> tuple:
    """Return dense white/black perspectives and STM-relative labels on device.

    Input is contiguous uint8 [B,32]. Signed shifts still extract bit 63 exactly.
    Shards validates record values once, before this hot path is called.
    If requested, also return material bucket indices computed on device:
    min(NB-1, (popcount(occupancy)-2)*NB//32). Empty records map to zero.
    """
    batch = raw.shape[0]
    occupancy = raw[:, :8].view(torch.int64).reshape(batch, 1)
    squares = torch.arange(64, device=raw.device, dtype=torch.int64).reshape(1, 64)
    occupied = ((occupancy >> squares) & 1)
    order = (occupied.cumsum(dim=1) - 1).clamp_min(0)
    nibble = torch.arange(32, device=raw.device).reshape(1, 32)
    codes = (raw[:, 8:24][:, nibble[0] // 2].to(torch.int64) >> ((nibble % 2) * 4)) & 15
    board = torch.where(occupied.bool(), codes.gather(1, order), 0)
    white = (board // 6) * 384 + (board % 6) * 64 + squares
    black = (1 - board // 6) * 384 + (board % 6) * 64 + (squares ^ 56)
    if king_map is not None:
        white_king = ((board == 5) & occupied.bool()).to(torch.int64).argmax(dim=1)
        black_king = ((board == 11) & occupied.bool()).to(torch.int64).argmax(dim=1) ^ 56
        white_mirror = torch.where((white_king & 7) >= 4, 7, 0)
        black_mirror = torch.where((black_king & 7) >= 4, 7, 0)
        white = (white ^ white_mirror[:, None]) + king_map[white_king ^ white_mirror][:, None] * 768
        black = (black ^ black_mirror[:, None]) + king_map[black_king ^ black_mirror][:, None] * 768
    indices = torch.cat((white, black), dim=0)
    features = torch.zeros((2 * batch, 768 * (king_buckets if king_map is not None else 1)),
                           device=raw.device, dtype=torch.float32)
    features.scatter_add_(1, indices, occupied.float().repeat(2, 1))
    stm = raw[:, 27].to(torch.int64)
    score = raw[:, 24:26].view(torch.int16).reshape(batch).float() * (1 - 2 * stm)
    result = raw[:, 26].float() * 0.5
    result = torch.where(stm == 0, result, 1 - result)
    decoded = features, stm, score, result
    if buckets is None:
        return decoded
    validate_buckets(buckets)
    bucket = ((occupied.sum(dim=1) - 2) * buckets // 32).clamp(0, buckets - 1)
    return (*decoded, bucket)


class NGNN(nn.Module):
    def __init__(self, hidden: int = 256, buckets: int = 1, king_map=None,
                 king_factorizer: bool = False, hidden2: int = 0):
        super().__init__()
        validate_hidden(hidden)
        validate_buckets(buckets)
        validate_hidden2(hidden2)
        if hidden2 and king_map is None:
            raise ValueError("Second hidden layer requires king inputs (NGNN4)")
        self.hidden2 = hidden2
        self.hidden, self.buckets = hidden, buckets
        self.king_buckets = 1
        self.king_map = None
        if king_map is not None:
            mapping = validate_map(king_map)
            self.king_buckets = max(mapping) + 1
            del self.king_map
            self.register_buffer("king_map", torch.tensor(mapping, dtype=torch.int64))
        if king_factorizer and (king_map is None or self.king_buckets < 2):
            raise ValueError("King factorizer requires NGNN3 with at least two king buckets")
        self.w1 = nn.Parameter(torch.empty(768 * self.king_buckets, hidden))
        self.b1 = nn.Parameter(torch.full((hidden,), 0.25))
        # Keep single-bucket checkpoint shapes and floating arithmetic unchanged.
        self.o = nn.Parameter(torch.empty((buckets, hidden2) if hidden2 else
                                         2 * hidden if buckets == 1 else (buckets, 2 * hidden)))
        self.ob = nn.Parameter(torch.zeros(() if buckets == 1 else (buckets,)))
        nn.init.normal_(self.w1, mean=0, std=0.02)
        nn.init.normal_(self.o, mean=0, std=0.05 / hidden ** 0.5)
        self.register_parameter("w2", nn.Parameter(torch.empty(buckets, hidden2, 2 * hidden))
                                if hidden2 else None)
        self.register_parameter("b2", nn.Parameter(torch.full((buckets, hidden2), 0.25))
                                if hidden2 else None)
        if hidden2:
            nn.init.normal_(self.w2, mean=0, std=0.5 / (2 * hidden) ** 0.5)
        # Zero initialization preserves the ordinary model's effective weights
        # and RNG stream. Each bucket learns a residual over the shared rows.
        self.register_parameter("w1_shared", nn.Parameter(torch.zeros(768, hidden))
                                if king_factorizer else None)

    def effective_w1(self) -> torch.Tensor:
        if self.w1_shared is None:
            return self.w1
        return (self.w1.reshape(self.king_buckets, 768, self.hidden)
                + self.w1_shared).reshape(-1, self.hidden)

    def forward(self, features: torch.Tensor, stm: torch.Tensor,
                bucket: torch.Tensor | None = None) -> torch.Tensor:
        acc = features @ self.effective_w1() + self.b1
        if self.hidden2:
            if bucket is None:
                if self.buckets > 1:
                    raise ValueError("Material bucket indices are required")
                bucket = torch.zeros_like(stm)
            white, black = acc.clamp(0, 1).chunk(2, dim=0)
            us = torch.where((stm == 0)[:, None], white, black)
            them = torch.where((stm == 0)[:, None], black, white)
            ordered = torch.cat((us, them), dim=1)
            # Compute all material heads as one GEMM, then select the head for
            # each position. Gathering W2 first duplicates [L,2H] per record;
            # this stores only [NB,L] activations per record instead.
            all_middle = ordered @ self.w2.flatten(0, 1).T
            middle = all_middle.reshape(-1, self.buckets, self.hidden2).gather(
                1, bucket[:, None, None].expand(-1, 1, self.hidden2)).squeeze(1)
            activation = (middle.float() + self.b2[bucket]).clamp(0, 1).square()
            bias = self.ob if self.buckets == 1 else self.ob[bucket]
            return ((activation * self.o[bucket]).sum(dim=1).float() + bias) * SCALE
        activation = acc.clamp(0, 1).square()
        # Compute both STM orderings without materializing a gathered [B,2H].
        white, black = activation.chunk(2, dim=0)
        if self.buckets > 1:
            if bucket is None:
                raise ValueError("Material bucket indices are required")
            weights = self.o[bucket]
            us = torch.where((stm == 0)[:, None], white, black)
            them = torch.where((stm == 0)[:, None], black, white)
            output = (us * weights[:, :self.hidden] + them * weights[:, self.hidden:]).sum(dim=1)
            return (output.float() + self.ob[bucket]) * SCALE
        white_cp = white @ self.o[:self.hidden] + black @ self.o[self.hidden:]
        black_cp = black @ self.o[:self.hidden] + white @ self.o[self.hidden:]
        return (torch.where(stm == 0, white_cp, black_cp).float() + self.ob) * SCALE

    @torch.no_grad()
    def clip_quantized(self) -> None:
        if self.w1_shared is None:
            self.w1.clamp_(-32768 / QA, 32767 / QA)
        else:
            self.w1_shared.clamp_(-32768 / QA, 32767 / QA)
            # Bound the sum that is used by forward and export; independently
            # clipping each component would allow an overflowing merged row.
            self.w1.reshape(self.king_buckets, 768, self.hidden).clamp_(
                -32768 / QA - self.w1_shared, 32767 / QA - self.w1_shared)
        self.b1.clamp_(-32768 / QA, 32767 / QA)
        if self.hidden2:
            self.w2.clamp_(-1, 1)
            self.b2.clamp_(-32768 / QA, 32767 / QA)
        self.o.clamp_(-1.98, 1.98)
        # Leave a representable float32 margin below int32's positive boundary.
        self.ob.clamp_(-2147483520 / (QA * QB), 2147483520 / (QA * QB))


def blended_loss(evaluation: torch.Tensor, score: torch.Tensor, result: torch.Tensor,
                 lam: float = 0.75, k: float = 400) -> torch.Tensor:
    target = lam * torch.sigmoid(score / k) + (1 - lam) * result
    return (torch.sigmoid(evaluation / k) - target).square().mean()
