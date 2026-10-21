#!/usr/bin/env bash
# Render report.metais (DORA incident report) from one JSON file.
#
# Usage: render-report.sh <inputs.json> [out-root]
#
# <inputs.json> is one JSON object holding the package's three inputs:
#   {"stage": "...", "entity": {...}, "incident": {...}}
# Each run renders into <out-root>/<YYYY-MM-DD>/ (default out-root: ./reports).
# A same-day re-run replaces that day's files.
#
# Environment:
#   METAIS   path to the metais binary (default: metais on PATH)
#   PACKAGE  path to the package (default: report.metais beside this script)
#
# Nightly via cron, e.g. at 02:17:
#   17 2 * * * /path/to/render-report.sh /path/to/report.json /path/to/reports >> /path/to/render-report.log 2>&1
set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
metais=${METAIS:-metais}
package=${PACKAGE:-$here/report.metais}

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: $(basename "$0") <inputs.json> [out-root]" >&2
  exit 2
fi
input=$1
out_root=${2:-$here/reports}

log() { echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $*" >&2; }

[[ -r $input ]] || { log "error: cannot read $input"; exit 1; }
[[ -r $package ]] || { log "error: cannot read package $package"; exit 1; }
command -v "$metais" >/dev/null || { log "error: metais not found (set METAIS)"; exit 1; }

mkdir -p "$out_root"

# Don't let two runs (e.g. a slow night plus a manual run) overlap.
exec 9>"$out_root/.render-report.lock"
if ! flock -n 9; then
  log "error: another render is running"
  exit 1
fi

now=$(date +%Y-%m-%dT%H:%M:%S%:z)
out="$out_root/$(date +%Y-%m-%d)"

# metais refuses a non-empty --out unless told to overwrite.
overwrite=()
[[ -d $out ]] && overwrite=(--overwrite)

log "rendering $input -> $out"
# render validates the inputs first and writes nothing if any finding is an error.
if ! "$metais" render "$package" --inputs-json - --out "$out" --now "$now" \
    "${overwrite[@]}" --color never < "$input"; then
  log "error: render failed (see findings above)"
  exit 1
fi
log "done"
