#!/usr/bin/env bash
# Render report.metais from one JSON file holding the package's inputs.
#
# Usage: render-report-nightly.sh INPUTS.json OUT_DIR [PACKAGE]
#
# INPUTS.json is an object whose members are the package's inputs:
#   {"stage": "initial notification", "entity": {...}, "incident": {...}}
# Validates first; renders only if valid. Files in OUT_DIR that the render
# writes are replaced; nothing else there is touched.
#
# Nightly cron example (02:00):
#   0 2 * * * /path/to/render-report-nightly.sh /ux/host /ux/host >>/var/log/report.log 2>&1
set -euo pipefail

if [ $# -lt 2 ] || [ $# -gt 3 ]; then
  echo "usage: $0 INPUTS.json OUT_DIR [PACKAGE]" >&2
  exit 2
fi

inputs=$1
out=$2
package=${3:-$(dirname "$(readlink -f "$0")")/report.metais}

[ -r "$inputs" ] || { echo "cannot read $inputs" >&2; exit 2; }
[ -r "$package" ] || { echo "cannot read package $package" >&2; exit 2; }
command -v metais >/dev/null || { echo "metais is not installed" >&2; exit 2; }

json=$(cat "$inputs")
now=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "$now validating $inputs"
metais validate "$package" --inputs-json "$json"

echo "$now rendering to $out"
metais render "$package" --inputs-json "$json" --out "$out" --overwrite --now "$now"
