# H768 embedded default adoption

The adopted NGNN3 file has H=768, KB=8, NB=8 and SHA-256
`2dcb6d936c0275927cb9e881f42e22558e618ad7e6fd998df6c4e5046d42c443`.
The H512 file remains available unchanged.

The existing fixed 1,000-game self-play gate at **10+0.1**, against
**81215229** (`812152293dad04184599b2f9ab82779168e6e134b221de136d8f8daae86d9f22`),
reported **KEEP: +13.6 [2.0, 25.0], W/D/L 215/609/176, zero faults**.
This is not a 120+1 rating claim. `result.json` preserves the exact gate
result, including its native INCONCLUSIVE SPRT verdict and the separate
fixed-game KEEP decision. No games were rerun and no service was started.
`training-manifest.json`, `export.json`, and `gate-manifest.json` are copied
unchanged from the training run. The manifest's `base_options` and `artifacts`
identify the actual H512 EvalFile and its full 81215229 digest; the separate
`base_net_sha256` field disagrees. This inconsistency is retained, not repaired.
The net change does not alter main's default search parameters or tune SPSA.

`identity.log` / `identity.json` record the depth-10, identity-only
`scripts/speedcheck.py` comparison pinned to CPUs 12-15 (probe CPU 12).
Each backend has six positions with iterative and fixed-depth searches:
identical nodes, scores, best moves, and full PVs, with no timed trials.
The ordinary `cmd/speedprobe` loads the explicit net with `LoadNGNN1`, the
same loader used by UCI EvalFile. The other probe uses a temporary Go overlay
of only `cmd/speedprobe/network.go` with `embedded-network.go`, which calls
`LoadDefaultNet` without opening the supplied path. Both preserve HCE for
an empty path. This adaptation is necessary because speedcheck passes
`--net` to both binaries. Production probe and engine sources are unchanged.
`uci-identity.log` / `uci-identity.json` additionally compare the real engine
with its embedded startup default against UCI `EvalFile=<the H768 file>`
on all six probe positions at depth 10, matching nodes, scores, moves, and
PVs. `uci_identity.py` reproduces this zero-game protocol proof.

Reproduce the proof from the repository root, writing outputs outside it:

```sh
export GOCACHE=/work/cache/go-build GOMODCACHE=/work/cache/go-mod GOPATH=/work/cache/go GOMAXPROCS=2
proof=experiments/burn-1104/h768
scratch=$(mktemp -d "${TMPDIR:?}/h768-proof.XXXXXXXX")
python3 - "$scratch/overlay.json" <<'PYTHON'
import json, pathlib, sys
r=pathlib.Path.cwd()
pathlib.Path(sys.argv[1]).write_text(json.dumps({'Replace': {
    str(r/'cmd/speedprobe/network.go'):
    str(r/'experiments/burn-1104/h768/embedded-network.go')}}))
PYTHON
nice -n 10 go build -p 2 -o "$scratch/file-probe" ./cmd/speedprobe
nice -n 10 go build -p 2 -overlay "$scratch/overlay.json" -o "$scratch/embedded-probe" ./cmd/speedprobe
nice -n 10 taskset -c 12-15 python3 scripts/speedcheck.py "$scratch/embedded-probe" "$scratch/file-probe" --net engine/nets/ngn-h768-2dcb6d93.nnue --cpu 12 --depth 10 --duration 1 --repeats 1 --identity-only --out "$scratch/identity.json"
NGN_NGNN3_PARITY_NET=nets/ngn-h768-2dcb6d93.nnue NGN_NGNN3_PARITY_EVALS=../experiments/burn-1104/h768/parity.json nice -n 10 go test -p 2 -short -count=1 -v ./engine -run '^TestNGNN3TrainerParity$'
nice -n 10 go test -p 2 -short -count=1 ./engine -run Net
```

`parity.json` preserves the trainer's 20 exact integer cases. `parity.log`
and `net-tests.log` are fresh passing runs. Proofs ran without credential,
token, secret, or password environment variables; native wait4 timing and
peak RSS appear in the logs. `time_v.py` is the accounting implementation.

The size-only wasm measurement is in `wasm-size.json` and the two build
logs: **11,030,046 -> 14,190,162 bytes (+3,160,116 bytes)** with Go 1.27.3.
One GOOS=js GOARCH=wasm build was run per revision. The extra baseline build
was needed because no existing NGN wasm artifact was available in other
NGN checkouts or shared caches. No browser assets or rungine files changed.
