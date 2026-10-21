#!/usr/bin/env bash
# Nightly render of the DORA incident report package from JSON inputs.
# Usage: render-report.sh [package] [data-dir] [out-dir]
# data-dir must hold stage.json, entity.json and incident.json.
set -euo pipefail

PKG="${1:-report.metais}"
DATA="${2:-incident}"
OUT="${3:-report}"

for f in stage entity incident; do
  [[ -f "$DATA/$f.json" ]] || { echo "missing $DATA/$f.json" >&2; exit 1; }
done

echo "$(date -Is) validating $PKG with $DATA"
metais validate "$PKG" --data "$DATA"

echo "$(date -Is) rendering to $OUT"
metais render "$PKG" --data "$DATA" --out "$OUT" --overwrite
echo "$(date -Is) done"
