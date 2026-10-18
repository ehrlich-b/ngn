#!/usr/bin/env bash
# Run only at the next natural exclusive boundary after higher-priority gates.
# If the accepted source/net/vector changes first, rebuild/recheck before use.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session108-speed
exec taskset -c 0-15 nice -n 19 python3 "$D/speed-gate.py" \
    --candidate s4 --profile-admission "$D/s4-profile-admission.json" \
    --out "$D/s4-timing" --base-binary "$R/session104-speed/s1-ngn" \
    --candidate-binary "$D/s4-ngn" --identity "$D/identity.json"
