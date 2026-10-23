#!/usr/bin/env bash
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# regenerate.sh rewrites every committed generated file from its sources,
# from the repository root:
#
#   regenerate.sh
#
# Run it after every rebase or merge, and after any change to a generated
# file's sources, then commit what it changed. The merge drivers keep one
# side of a generated file (merge-generated.sh); this is what makes it
# right again, and each file's drift test fails until it has run.
#
# The finding chain runs in its order: the registry table, then
# spec/findings/registry.json, then the embedded copies, then
# spec/statements.json and the code catalog in spec/12-findings.md. The
# files it writes are the ones .gitattributes gives a merge=metais-* driver.
#
# Exit 0 when every generator ran, 1 when one failed, 2 when it cannot run.

set -u

here=$(cd "$(dirname "$0")" && pwd) || exit 2
# shellcheck source=git-env.sh
. "$here/git-env.sh" || exit 2
root=$(git rev-parse --show-toplevel 2>/dev/null) || {
	echo "regenerate: not inside a git checkout" >&2
	exit 2
}
cd "$root" || exit 2
"$here/merge-setup.sh" || exit 2
export GOTOOLCHAIN=auto

failed=""
run() {
	echo "--- $*"
	"$@" || failed="$failed
  $*"
}

run go generate ./internal/finding ./internal/specdata ./internal/guarantees
run go test -count=1 -run '^TestHelpGolden$' ./internal/cli -args -update
run go test -count=1 -run '^TestScaffoldGolden$' ./internal/author -args -update
run go test -count=1 -run '^TestGolden$' ./internal/tarw -args -update
run go test -count=1 -run '^TestWriteGolden$' ./internal/yamlx -args -update
run go test -count=1 -run '^TestFormulaGolden$' ./internal/tools/release -args -update
run go test -count=1 -run '^TestPackages$' ./internal/vectorgen -args -update
run go test -count=1 -run '^TestContainerVectors$' ./internal/vectorgen -args -update
run go test -count=1 -run '^TestPackOutputs$' ./internal/vectorgen -args -update
run go test -count=1 -run '^TestValues$' ./internal/vectorgen -args -update
run go test -count=1 -run '^TestThirdPartyNotices$' ./internal/guarantees -args -notices.update

echo "--- generated files that differ from HEAD"
git status --short -- spec internal THIRD-PARTY-NOTICES.txt | grep -v '^??' || echo "(none)"
if [ -n "$failed" ]; then
	echo "regenerate: FAILED:$failed"
	exit 1
fi
echo "regenerate: done; commit what changed"
