#!/bin/sh
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# conformance.sh is how CI runs internal/conformance's tests (R-W1-4, plan
# orchestrator, 2026-10-22). The package's one test binary cannot finish
# within go test's timeout on a 2-core GitHub-hosted runner, so CI splits it
# by -run, as R-W6-3 split the SC7 reach: each of its top-level tests in its
# own invocation, and TestConformance once per area of the suite. From the
# repository root:
#
#   conformance.sh skip   print a go test -skip pattern naming those tests,
#                         for the run of every other package
#   conformance.sh run    run each of them in its own go test -race
#
# The tests are the package's own list (go test -list) and the areas are
# spec/conformance/vectors' directories, both read when the job runs, so a
# test or an area added later is never skipped. TestHostIsolation is not
# split out: gate.sh's reach steps run it by name, so a pattern skipping it
# would skip the reach, and the gate's run of the package runs it.
#
# skip fails when another package declares a test of one of those names,
# since the pattern would skip that test too. run fails when an invocation
# fails, or when it did not run the test or the area it selects.
#
# Exit 0 on success, 1 on a failure, 2 when it cannot start.
set -u

pkg=./internal/conformance

# names prints the package's top-level tests, examples and fuzz targets on
# this platform, less TestHostIsolation.
names() {
	out=$(go test -race -list '.*' "$pkg") || {
		printf '%s\n' "$out" >&2
		echo "conformance: cannot list the tests of $pkg" >&2
		exit 2
	}
	printf '%s\n' "$out" | grep -E '^(Test|Example|Fuzz)' | grep -vx TestHostIsolation
}

# one runs one test, or with an area one area of it, in its own go test.
one() {
	if [ -n "$2" ]; then
		label="$1/$2"
		sel="^$1\$/^$2\$"
		ran="^    --- (PASS|FAIL|SKIP): $1/$2 "
	else
		label=$1
		sel="^$1\$"
		ran="^--- (PASS|FAIL|SKIP): $1 "
	fi
	echo "=== $label"
	out=$(go test -race -count=1 -v -run "$sel" "$pkg" 2>&1) || rc=1
	printf '%s\n' "$out" | grep -vE '^(=== (RUN|PAUSE|CONT|NAME)|[[:space:]]*--- PASS)' || :
	if ! printf '%s\n' "$out" | grep -qE "$ran"; then
		echo "conformance: $label did not run"
		rc=1
	fi
}

case "${1:-}" in
skip)
	list=$(names) || exit 2
	rc=0
	for t in $list; do
		if git grep -qE "^func $t\(" -- '*_test.go' ":(exclude)${pkg#./}"; then
			echo "conformance: another package declares $t, which the pattern would skip" >&2
			rc=1
		fi
	done
	[ "$rc" -eq 0 ] || exit 1
	echo "^($(printf '%s\n' "$list" | paste -sd '|' -))\$"
	;;
run)
	list=$(names) || exit 2
	rc=0
	for t in $list; do
		if [ "$t" = TestConformance ]; then
			for d in spec/conformance/vectors/*/; do
				one "$t" "$(basename "$d")"
			done
		else
			one "$t" ""
		fi
	done
	exit "$rc"
	;;
*)
	echo "usage: conformance.sh skip|run" >&2
	exit 2
	;;
esac
