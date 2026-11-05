#!/usr/bin/env bash
set -euo pipefail
# Run from the repository root with an existing NumPy-enabled python3.
fixture=experiments/wave2-1104/integer-validation
exec nice -n 10 python3 -m trainer.quantized_validation \
  --network "$fixture/h32.nnue" --records "$fixture/records.ngnp" \
  --float-predictions "$fixture/float-cp.jsonl" --max-records 32 "$@"
