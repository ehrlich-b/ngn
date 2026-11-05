#!/usr/bin/env bash
set -euo pipefail
# From the repository root; output directory must be outside the checkout.
exec nice -n 10 "$HOME/ngn-original-nnue-night-20261001/venv/bin/python" \
  experiments/wave2-1104/qat-model/run.py "$@"
