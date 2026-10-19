#!/usr/bin/env bash
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# gate-why.sh re-runs one test alone, the first instrument for a red the gate
# reported:
#
#   gate-why.sh <TestName[/subtest]> <package> [go test flags...]
#   gate-why.sh TestExistingWritesItsLimit ./internal/hostfs
#   gate-why.sh 'TestConformance/render/fan_out' ./internal/conformance -timeout 120s
#
# It runs `go test -race -count=1 -v` with every level of the name anchored,
# so TestA does not also run TestAB, and prints:
#
#   - whether the change on this branch reaches the package (gate-select.sh);
#   - the test's own output;
#   - the run's wall time beside its CPU time (user + system, the test binary
#     and everything it started).
#
# Read the times, not only the verdict. CPU time far below wall time means the
# test spent its time waiting (a sleep, a lock, a pipe, a child process), and
# a wait that fits alone can overrun on a loaded host: a timing assumption.
# CPU time near wall time times the cores in use means the test is bound by
# work, and a share of a busy host stretches it.
#
# A pass here closes nothing. It says the failure needs something the gate
# run had and this run did not: other tests beside it, a loaded host, a
# timing. That is the lead the root-cause skill starts from.
#
# Exit status is go test's.

set -u

name=${1:-}
pkg=${2:-}
[ -n "$name" ] && [ -n "$pkg" ] || {
	echo "usage: gate-why.sh <TestName[/subtest]> <package> [go test flags...]" >&2
	exit 2
}
shift 2

here=$(cd "$(dirname "$0")" && pwd) || exit 2
root=$(git rev-parse --show-toplevel 2>/dev/null) || {
	echo "gate-why: not inside a git checkout" >&2
	exit 2
}
cd "$root" || exit 2
export GOTOOLCHAIN=auto

pat=""
rest=$name
while :; do
	part=${rest%%/*}
	pat="${pat:+$pat/}^${part}\$"
	[ "$rest" = "$part" ] && break
	rest=${rest#*/}
done

import=$(go list "$pkg" 2>/dev/null) || {
	echo "gate-why: go list cannot resolve $pkg" >&2
	exit 2
}
if sel=$("$here/gate-select.sh" 2>/dev/null); then
	if printf '%s\n' "$sel" | grep -qE "^(test|blind) $import\$"; then
		echo "reach: this branch's change reaches $import:"
		printf '%s\n' "$sel" | grep -E "^touched " | sed 's/^/  /' | head -20
	else
		echo "reach: this branch's change does not reach $import, by gate-select.sh's rules."
		echo "  Read the failure before concluding anything from that: the rules see imports and named data,"
		echo "  not a shared host, a shared temporary directory or a time bound."
	fi
else
	echo "reach: gate-select.sh could not measure the change (no origin/main?)"
fi

echo "run:   go test -race -count=1 -v -run '$pat' $* $pkg"
out=$(mktemp "${TMPDIR:-/tmp}/gate-why.XXXXXX") || exit 2
if [ -x /usr/bin/time ]; then
	/usr/bin/time -p go test -race -count=1 -v -run "$pat" "$@" "$pkg" 2>"$out"
	rc=$?
	grep -vE '^(real|user|sys) ' "$out" >&2
	real=$(awk '$1 == "real" {print $2}' "$out")
	user=$(awk '$1 == "user" {print $2}' "$out")
	sys=$(awk '$1 == "sys" {print $2}' "$out")
	echo "times: wall ${real}s, CPU $(awk -v u="$user" -v s="$sys" 'BEGIN {printf "%.2f", u + s}')s (user ${user}s + system ${sys}s), build included"
else
	go test -race -count=1 -v -run "$pat" "$@" "$pkg"
	rc=$?
	echo "times: /usr/bin/time is not here; measure CPU time in the test (syscall.Getrusage)"
fi
rm -f "$out"
if [ "$rc" -eq 0 ]; then
	echo "PASSED alone. That closes nothing: find what the gate run had that this run did not (root-cause)."
else
	echo "FAILED alone (exit $rc): the failure reproduces without the rest of the gate."
fi
exit "$rc"
