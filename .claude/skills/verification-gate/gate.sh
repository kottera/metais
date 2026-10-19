#!/usr/bin/env bash
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# gate.sh runs AGENTS.md's gate from the repository root. Every step runs even
# after one goes red, so one run reports everything that is wrong. The run
# logs to .local/gate/<UTC time>.log (git-ignored); .local/gate/latest.log
# names the newest run, and gate-wait.sh watches it.
#
# Exit 0 when every step passes, 1 when any fails, 2 when it cannot start.
#
# METAIS_GATE_P overrides `go test -p`, the number of packages built and
# tested at once. The default is half the host's cores, because the hosts
# that run this gate run several gates at once.

set -u

root=$(git rev-parse --show-toplevel 2>/dev/null) || {
	echo "gate: not inside a git checkout" >&2
	exit 2
}
cd "$root" || exit 2
state="$root/.local/gate"
mkdir -p "$state" || exit 2

if [ -f "$state/pid" ]; then
	old=$(cat "$state/pid")
	if ! kill -0 "$old" 2>&1 | grep -q 'No such process'; then
		echo "gate: a run is already live (pid $old). Watch it with gate-wait.sh, or kill it first." >&2
		exit 2
	fi
fi

log="$state/$(date -u +%Y%m%dT%H%M%SZ).log"
ln -sf "$log" "$state/latest.log"
: >"$log.seen"
echo $$ >"$state/pid"
echo "gate: logging to $log"
exec >"$log" 2>&1

export GOTOOLCHAIN=auto
ncpu=$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4)
half=$((ncpu / 2))
p=${METAIS_GATE_P:-$((half > 2 ? half : 2))}

echo "go:   $(go version)"
echo "head: $(git rev-parse HEAD)"
echo "tree: $(git status --porcelain | wc -l | tr -d ' ') uncommitted path(s)"
echo "-p:   $p (host has $ncpu cores)"

failed=""

step() {
	local name=$1
	shift
	echo "=== STEP $name"
	local start rc secs
	start=$(date +%s)
	"$@"
	rc=$?
	secs=$(($(date +%s) - start))
	if [ "$rc" -eq 0 ]; then
		echo "=== PASS $name (${secs}s)"
	else
		echo "=== FAIL $name (exit $rc, ${secs}s)"
		failed="$failed $name"
	fi
}

gofmt_clean() {
	local out
	out=$(gofmt -l .)
	[ -z "$out" ] || {
		echo "$out"
		return 1
	}
}

static_build() {
	CGO_ENABLED=0 go build -trimpath -o "$state/metais-static" ./cmd/metais
}

cross() {
	local rc=0 t
	for t in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
		if GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go vet ./... &&
			GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go build ./...; then
			echo "cross $t ok"
		else
			echo "cross $t FAILED"
			rc=1
		fi
	done
	return $rc
}

specdata_fresh() {
	go generate ./internal/specdata && git diff --exit-code --stat internal/specdata
}

step build go build ./...
step vet go vet ./...
step test go test -race -p "$p" ./...
step tidy go mod tidy -diff
step gofmt gofmt_clean
step static static_build
step cross cross
step specdata specdata_fresh

echo "=== SUMMARY"
grep -E '^=== (PASS|FAIL) ' "$log"
if [ -z "$failed" ]; then
	echo "GATE DONE: PASS"
	rm -f "$state/pid"
	exit 0
fi
echo "GATE DONE: FAIL:$failed"
rm -f "$state/pid"
exit 1
