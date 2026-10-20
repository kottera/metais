#!/usr/bin/env bash
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# gate-exec.sh runs one test binary for gate.sh, as `go test -exec`, and
# measures what the host gave it. It appends one tab-separated line to the
# file METAIS_GATE_CPU names:
#
#   <directory> <start> <end> <CPU seconds> <exit> <load>
#
# - The directory is the package's: go test runs its binary there.
# - Start and end are epoch seconds: to the microsecond where bash has
#   EPOCHREALTIME, to the second where it does not.
# - The CPU is user plus system time of the binary and of every process it
#   waited for, as the kernel accounts them to this shell's reaped children
#   (bash's `times`). Measuring runs nothing beside the binary, so it burns
#   no CPU of its own.
# - The load is the host's one-minute load average when the binary ended,
#   or "?" when the host exposes none.
#
# The binary gets the environment go test gave it, less this script's own
# variables, and its exit status is this script's.
#
#   gate-exec.sh --load   prints the host's one-minute load average, or "?"
#
# The load is read from what the host already exposes: /proc/loadavg,
# darwin's vm.loadavg, uptime, or the sysinfo(2) reader gate.sh builds from
# loadavg.go (METAIS_GATE_LOADAVG) where a sandbox hides the other three.

set -u

loadavg=${METAIS_GATE_LOADAVG:-}

host_load() {
	local l=""
	if [ -r /proc/loadavg ]; then
		l=$(cut -d' ' -f1 /proc/loadavg 2>/dev/null)
	fi
	[ -n "$l" ] || l=$(sysctl -n vm.loadavg 2>/dev/null | tr -d '{}' | awk '{print $1}')
	[ -n "$l" ] || l=$(uptime 2>/dev/null | sed -n 's/.*load averages*: *\([0-9.,]*\).*/\1/p' | sed 's/,$//; s/,/./')
	[ -n "$l" ] || { [ -n "$loadavg" ] && l=$("$loadavg" 2>/dev/null); }
	printf '%s\n' "${l:-?}"
}

if [ "${1:-}" = --load ]; then
	host_load
	exit 0
fi

out=${METAIS_GATE_CPU:-}
unset METAIS_GATE_CPU METAIS_GATE_LOADAVG

start=${EPOCHREALTIME:-}
[ -n "$start" ] || start=$(date +%s)
"$@"
rc=$?
end=${EPOCHREALTIME:-}
[ -n "$end" ] || end=$(date +%s)
[ -n "$out" ] || exit "$rc"

# times prints this shell's own times, then its reaped children's, each as
# user and system time in the form 1m2.345s. It runs in this shell, not in a
# command substitution, whose fork would start with no children's times.
times >"$out.$$"
cpu=$(LC_ALL=C awk 'NR == 2 {
	n = 0
	for (i = 1; i <= 2; i++) {
		gsub(",", ".", $i)
		split($i, a, "m")
		sub("s", "", a[2])
		n += a[1] * 60 + a[2]
	}
	printf "%.2f", n
}' "$out.$$")
rm -f "$out.$$"
printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$PWD" "${start/,/.}" "${end/,/.}" "${cpu:-?}" "$rc" "$(host_load)" >>"$out"
exit "$rc"
