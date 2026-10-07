#!/usr/bin/env bash
# Fixed 120+1, eight games total, 160 per anchor, indices 0-79 both colors.
set -euo pipefail
exec bash "$(dirname "$0")/launch.sh" confirmation "$@"
