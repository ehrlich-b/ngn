#!/usr/bin/env bash
# Fixed 10+0.1, eight games total, 80 per anchor, indices 0-39 both colors.
set -euo pipefail
exec bash "$(dirname "$0")/launch.sh" screen "$@"
