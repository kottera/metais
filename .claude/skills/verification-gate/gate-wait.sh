#!/usr/bin/env bash
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# gate-wait.sh watches the newest gate.sh run, of either tier. It blocks for
# up to 570 seconds (METAIS_GATE_WAIT overrides), so one call fits inside a
# foreground shell call's ten-minute bound, and returns as soon as there is
# something to act on. Its first line is one word:
#
#   DONE         the run finished; its summary follows (exit 0 green, 1 red)
#   RUNNING      nothing new in this window; call it again (exit 2)
#   NEW-FAILURE  a failure appeared and the run goes on (exit 3)
#   DIED         the run is gone and never finished (exit 4)
#   NO-RUN       there is no run to watch (exit 5)
#
# Each failure line is reported once per run.

set -u

root=$(git rev-parse --show-toplevel 2>/dev/null) || {
	echo "NO-RUN: not inside a git checkout"
	exit 5
}
state="$root/.local/gate"
[ -e "$state/latest.log" ] || {
	echo "NO-RUN: start one with gate.sh"
	exit 5
}
log=$(readlink "$state/latest.log")
seen="$log.seen"
touch "$seen"
pattern='^(--- FAIL|FAIL|=== FAIL|panic: )'
deadline=$(($(date +%s) + ${METAIS_GATE_WAIT:-570}))

done_report() {
	echo DONE
	sed -n '/^=== SUMMARY/,$p' "$log"
	echo "--- failure lines"
	sed '/^=== SUMMARY/,$d' "$log" | grep -E "$pattern" | head -60
	echo "log: $log"
	grep -q '^GATE DONE: PASS' "$log"
}

while :; do
	if grep -q '^GATE DONE' "$log"; then
		done_report && exit 0
		exit 1
	fi
	new=$(grep -E "$pattern" "$log" | grep -vxF -f "$seen")
	if [ -n "$new" ]; then
		echo NEW-FAILURE
		echo "$new" | head -60
		echo "$new" >>"$seen"
		echo "log: $log"
		exit 3
	fi
	pid=$(cat "$state/pid" 2>/dev/null)
	if [ -z "$pid" ] || kill -0 "$pid" 2>&1 | grep -q 'No such process'; then
		if grep -q '^GATE DONE' "$log"; then
			continue
		fi
		echo DIED
		tail -30 "$log"
		echo "log: $log"
		exit 4
	fi
	if [ "$(date +%s)" -ge "$deadline" ]; then
		echo RUNNING
		grep -E '^=== (STEP|PASS|FAIL|REUSED|COVERED) ' "$log" | tail -4
		exit 2
	fi
	sleep 5
done
