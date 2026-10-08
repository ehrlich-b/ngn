# NGNN4 narrow hidden layer — session 15 topic candidate

Prepared on `night/hidden-layer` from accepted runtime `5b882e2` via
campaign source `9b9266b`. It is independent of the deferred-accumulator
candidate. No trained net, playing-strength gate or acceptance is implied.
All computation runs on WSL; local work is editing only.

NGNN3 has a squared clipped accumulator followed directly by a material
output row. NGNN4 adds a small learned layer after the two accumulators:
clipped linear accumulator activation, STM ordering, a material-specific
affine layer of L=16/32/48/64 neurons, squared clipped activation, then a
material output row. This tests whether feature interactions can compensate
for inference cost. King features, W1/B1 and refresh/update mechanics stay
the same. No third-party code or weights were used.

Integer specification:

1. `x = clamp(acc, 0, 255)`, concatenate us then them.
2. `z[j] = trunc(sum(x[i] * W2[j,i]) / 64) + B2[j]`.
3. `a[j] = clamp(z[j], 0, 255)^2`.
4. `cp = trunc((trunc(sum(a[j]*O[j]) / 255) + OB) * 400 / (255*64))`,
   clamped to [-24999,24999]. Every signed division truncates toward zero.

W2 float weights are clipped to [-1,1] and quantized by 64 to int8
[-64,64]. B2 is int16 quantized by 255. O/OB retain their existing scales.
The complete W2 dot is bounded by `4096*255*64 = 66,846,720`, within int32.
The original AVX2 kernel processes four uint8 inputs for eight outputs with
packed signed weights: adjacent int16 pair bounds are 32,640, so saturation
cannot change a valid result. A portable/race Go kernel shares the same
immutable loader-derived layout. Files retain natural W2 row order.

All fields little-endian: `NGN4`; eight uint32s
`4,H,KB,NB,L,255,64,400`; 64-byte king map; W1 int16 [KB*768,H];
B1 int16 [H]; W2 int8 [NB,L,2H]; B2 int16 [NB,L]; O int16 [NB,L];
OB int32 [NB]; IEEE CRC32. Header is 100 bytes. H/KB/NB keep NGNN3's
bounds; L is a multiple of 16 in [16,64]. Loader/selection/UCI accept
this version under backend `ngnn4`; existing formats keep their old bytes.

Trainer option `--format 4 --hidden2 16` selects this architecture. W2
receives AdamW decay, B2 does not. Checkpoint/resume binds hidden2, format,
king map and existing recipe fields. Defaults preserve the previous model's
initialization, RNG and exports. The existing factorizer is optional but
must remain off for an initial single-change architecture trial.

Verification is queued after deferred controller 598985 finishes, using a
native pidfd wait. CPU tests use only 13,15 / nice 19 / GOMAXPROCS=2;
CUDA is hidden from the CPU suite so existing GPU training is untouched.
Require new forward/gradient/optimizer/export/resume tests and the full
trainer suite, exact Python integer/Go fixture parity, guarded AVX2/Go/scalar
differentials across every H and L, incremental special moves, worker/reload,
engine/full short/race and vet. A microbenchmark reports inference cost,
without a strength verdict. No tests run during the fixed speed sample.

Before any real GPU trial: run the new CUDA FP32/BF16/fused-AdamW test after
the existing admission/CUDA window, profile actual H256/512 L16/32 training
and inference cost, select one architecture prospectively, and write a full
manifest against the then-immediate accepted source/net. Preserve the owned
data, seed, full schedule and validation selection; parity and reexport,
fresh A/A40 and fixed paired gate400 at 10+0.1 are mandatory. Accept only
penta lower95 >0 with zero operational failures; otherwise shelve without
extension. No training, claim or playing-strength gate is queued here.

Controller **608741** queued **17:06:13 EDT**, native dependency **598985**,
results under `session15h2/`. Initial waiter **608645** in `session15h/` was
cancelled before tests to strengthen the negative signed-division control;
its original frozen inputs/logs are preserved. CPU results remain pending.

CPU verification **PASS**, controller 608741 completed **17:35:00 EDT**.
New trainer controls pass (five CPU tests, one CUDA skip); full suite has
**63 tests / 58 passes / 5 CUDA skips**, 78.714 s. Go AVX2 and purego controls
pass in 1.211/1.216 s, including **320 exact integer cases**, all widths,
guarded buffers, extreme affine sums, both turns and worker/reload controls.
Engine short/race **14.817 / 170.806 s**, full short/race, vet and build pass.
All 17 changed local files equal the tested source. Fixture `c91adc11` is
owned, seeded and reproducible. [CPU receipt](2026-10-08-session15-results/hidden-cpu.json),
[source](2026-10-08-session15-results/hidden-source.json),
[inference cost](2026-10-08-session15-results/hidden-inference.txt).
Verified candidate **ca3637a** is committed/pushed on `night/hidden-layer`.
No accepted runtime change or trained net. CUDA validation remains pending.

The initial output microbenchmarks show substantial added inference cost:
H512/L16 is approximately 0.75–0.83 microseconds versus the direct output
kernel. A further original AVX2 int32-to-byte input clamp is being tested
on the topic branch before any training. It keeps all integer values exact;
new guarded comparisons cover signed int32 endpoints and lane permutations.

Prospective input-pack rule, before launch: required NGNN4 AVX2/purego,
engine/full short/race/vet checks; exact 320-case H512/L16 parity; **48 search
identities** (NGNN4 H512/L16 depth 12, accepted NGNN3 and NGNN2 depth 14,
HCE depth 11, twelve each). Then warm both probes and **20 fixed AB/BA timing
blocks**, six positions × three seconds per side, CPU 13 / nice 19 /
GOMAXPROCS=1, same explicit PGO. Use paired aggregate NPS, geometric mean
and two-sided 95% t interval on log ratios, df=19. Keep the input-pack edit
on the topic only if lower ratio >1 and both order strata >1; otherwise
restore it without extension. Baseline **ca3637a**, same frozen owned random
H512/L16 net on both sides; this refactor has no net-strength verdict.
Scripts [controller](2026-10-08-ngnn4-pack-controller.sh) and
[fixed check](2026-10-08-ngnn4-pack-check.py), box `session15h3/`.
