#!/usr/bin/env bash
# Renders report.metais from a JSON file into a dated directory.
#
# Usage: render-report.sh <inputs.json> [out-root]
#
# <inputs.json> is one JSON object whose members are the package's inputs:
#   { "stage": "initial notification", "entity": {...}, "incident": {...} }
# Each night's report lands in <out-root>/<YYYY-MM-DD>/ (default: ./reports).
# Re-running on the same day replaces that day's files.
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: ${0##*/} <inputs.json> [out-root]" >&2
  exit 2
fi

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
package=${METAIS_PACKAGE:-$here/report.metais}
inputs=$1
out_root=${2:-$here/reports}

[[ -r $inputs ]]  || { echo "cannot read $inputs" >&2; exit 2; }
[[ -r $package ]] || { echo "cannot read $package" >&2; exit 2; }
command -v metais >/dev/null || { echo "metais is not on PATH" >&2; exit 2; }

out=$out_root/$(date +%Y-%m-%d)
now=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Validate first so a bad input file fails with every finding listed.
metais validate "$package" --inputs-json - < "$inputs"

mkdir -p "$out"
metais render "$package" --inputs-json - --out "$out" --overwrite --now "$now" < "$inputs"
