#!/usr/bin/env bash
# Natural exclusive boundary only, after due final-vector and search gates.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session113-speed
exec taskset -c 0-15 nice -n 19 python3 "$D/speed-gate.py" \
    --candidate s5 --profile-admission "$D/s5-profile-admission.json" \
    --out "$D/s5-timing" --base-binary "$R/session104-speed/s1-ngn" \
    --candidate-binary "$D/s5-ngn" --identity "$D/identity.json"
