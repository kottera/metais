#!/usr/bin/env bash
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# gate.sh runs one tier of the gate (ADR-010 Amendment A5) from the
# repository root:
#
#   gate.sh                 the scoped tier, for a Work Order's change
#   gate.sh wave            the wave tier: AGENTS.md's full list
#   gate.sh [tier] --resume re-run only what the tier's last run left
#                           unverified (below)
#   gate.sh --base <ref>    measure the scoped tier's change from <ref>, not
#                           from the merge base with origin/main
#
# Run it in the background and watch it with gate-wait.sh. Every step runs
# even after one goes red, so one run reports everything that is wrong.
#
# Files, under .local/gate/ (git-ignored):
#
#   <run>.log        everything the run printed; latest.log names the newest
#   <run>.record     one line per verdict: status, item, the tree it was
#                    reached on, and the run that reached it
#   latest-<tier>    the name of the tier's newest run, which --resume reads
#   pid              the live run's process, while it runs
#
# An item is a step (build, vet, ...) or one package's tests ("test <pkg>").
# A verdict's tree is a git tree id of the working tree, untracked files
# included, taken when the verdict was reached.
#
# --resume reuses a PASS from the tier's last run when the tree is unchanged
# for that item since its verdict: the paths that differ between that tree
# and this one do not reach it, by gate-select.sh's rules. A FAIL, an item
# the run never reached (killed, died, never started), and a PASS the change
# reaches all run again. A resume of a resume carries every verdict forward
# with the tree it was reached on, so a chain of resumes never launders a
# stale PASS.
#
# The shared-host cap (ADR-010 Amendment A5 item 7): the gate's share of the
# host is a third of its cores (at least 2), the share AGENTS.md gives a
# fuzzer, because the hosts that run this gate run about three sessions at
# once. `go test -p` (package binaries at once) and -parallel (parallel tests
# at once in one binary, and internal/conformance's tool processes) are both
# the share. METAIS_GATE_P sets the share, and is the one override.
#
# Every run first runs merge-setup.sh, which installs the generated files'
# merge drivers in this clone's configuration and says so, on the terminal
# and in the log, when they were missing.
#
# Exit 0 when every item passes, 1 when any fails, 2 when it cannot start.

set -u

here=$(cd "$(dirname "$0")" && pwd) || exit 2
root=$(git rev-parse --show-toplevel 2>/dev/null) || {
	echo "gate: not inside a git checkout" >&2
	exit 2
}
cd "$root" || exit 2
state="$root/.local/gate"
mkdir -p "$state" || exit 2

tier=scoped
resume=""
base=""
while [ $# -gt 0 ]; do
	case "$1" in
	scoped | wave) tier=$1 ;;
	--resume) resume=1 ;;
	--base)
		[ -n "${2:-}" ] || { echo "gate: --base needs a ref" >&2; exit 2; }
		base=$2
		shift
		;;
	*)
		echo "gate: unknown argument $1 (usage: gate.sh [scoped|wave] [--resume] [--base <ref>])" >&2
		exit 2
		;;
	esac
	shift
done

if [ -f "$state/pid" ]; then
	old=$(cat "$state/pid")
	if ! kill -0 "$old" 2>&1 | grep -q 'No such process'; then
		echo "gate: a run is already live (pid $old). Watch it with gate-wait.sh, or kill it first." >&2
		exit 2
	fi
fi

prev=""
if [ -n "$resume" ]; then
	prev=$(cat "$state/latest-$tier" 2>/dev/null)
	[ -n "$prev" ] && { [ -f "$state/$prev.record" ] || [ -f "$state/$prev.reuse" ]; } || {
		echo "gate: --resume found no earlier $tier run to resume; start a plain run" >&2
		exit 2
	}
fi

setup=$("$here/merge-setup.sh" 2>&1)
echo "$setup"

run="$tier-$(date -u +%Y%m%dT%H%M%SZ)"
log="$state/$run.log"
record="$state/$run.record"
: >"$record"
ln -sf "$log" "$state/latest.log"
: >"$log.seen"
echo "$run" >"$state/latest-$tier"
echo $$ >"$state/pid"
trap 'rm -f "$state/pid"' EXIT
echo "gate: $tier tier, logging to $log"
exec >"$log" 2>&1

export GOTOOLCHAIN=auto
ncpu=$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4)
third=$((ncpu / 3))
p=${METAIS_GATE_P:-$((third > 2 ? third : 2))}
parallel=$p

# snapshot prints the working tree's git tree id, untracked files included,
# through a scratch index so the real index is never touched.
snapshot() {
	local idx
	idx=$(mktemp "${TMPDIR:-/tmp}/gate-index.XXXXXX") || return 1
	GIT_INDEX_FILE=$idx git read-tree HEAD &&
		GIT_INDEX_FILE=$idx git add --all . &&
		GIT_INDEX_FILE=$idx git write-tree
	local rc=$?
	rm -f "$idx"
	return $rc
}

tree=$(snapshot) || {
	echo "gate: could not take the working tree's snapshot"
	echo "GATE DONE: FAIL: snapshot"
	exit 2
}

echo "$setup"
echo "tier: $tier"
echo "go:   $(go version)"
echo "head: $(git rev-parse HEAD)"
echo "tree: $tree ($(git status --porcelain | wc -l | tr -d ' ') uncommitted path(s))"
echo "cap:  -p $p -parallel $parallel (host has $ncpu cores)"
[ -z "$prev" ] || echo "resumes: $prev"

failed=""

# verdict records an item's status against the tree it was reached on.
verdict() {
	printf '%s\t%s\t%s\t%s\n' "$1" "$2" "${3:-$tree}" "${4:-$run}" >>"$record"
}

# What --resume may reuse: the last verdict for each item in the previous
# run's own candidates and record, newest last, so a run that died before
# reaching a verdict still hands its predecessor's verdicts on. For each tree
# those verdicts were reached on, what the change since that tree reaches.
reuse="$state/$run.reuse"
: >"$reuse"
if [ -n "$prev" ]; then
	cat "$state/$prev.reuse" "$state/$prev.record" 2>/dev/null |
		awk -F'\t' '{last[$2] = $0} END {for (k in last) print last[k]}' >"$reuse"
	for t in $(cut -f3 "$reuse" | sort -u); do
		[ "$t" = "$tree" ] && continue
		git diff-tree -r --name-only --no-renames "$t" "$tree" >"$state/$run.diff.$t" 2>/dev/null &&
			"$here/gate-select.sh" --paths "$state/$run.diff.$t" >"$state/$run.reach.$t" ||
			: >"$state/$run.reach.$t.unknown"
	done
fi

# reusable <item> prints the earlier verdict's tree and run, and succeeds,
# when that verdict is a PASS the change since it does not reach.
reusable() {
	local item=$1 line st t from facts
	[ -n "$prev" ] || return 1
	line=$(awk -F'\t' -v k="$item" '$2 == k' "$reuse")
	[ -n "$line" ] || return 1
	st=$(printf '%s' "$line" | cut -f1)
	t=$(printf '%s' "$line" | cut -f3)
	from=$(printf '%s' "$line" | cut -f4)
	[ "$st" = PASS ] || return 1
	if [ "$t" != "$tree" ]; then
		[ -f "$state/$run.reach.$t" ] || return 1
		facts="$state/$run.reach.$t"
		case "$item" in
		"test "*)
			grep -qE "^(test|blind) ${item#test }\$" "$facts" && return 1
			;;
		tripwire-*)
			grep -qE "^(test|blind) $(tripwire_pkg "$item")\$" "$facts" && return 1
			;;
		build | vet | static | cross)
			grep -q '^code ' "$facts" && return 1
			;;
		*)
			# gofmt, tidy and the specdata drift are cheap; they run again
			# whenever the tree changed.
			return 1
			;;
		esac
	fi
	printf '%s\t%s\n' "$t" "$from"
}

step() {
	local name=$1
	shift
	local prior
	if prior=$(reusable "$name"); then
		echo "=== REUSED $name (PASS in ${prior#*	})"
		verdict PASS "$name" "${prior%%	*}" "${prior#*	}"
		return
	fi
	echo "=== STEP $name"
	local start rc secs
	start=$(date +%s)
	"$@"
	rc=$?
	secs=$(($(date +%s) - start))
	if [ "$rc" -eq 0 ]; then
		echo "=== PASS $name (${secs}s)"
		verdict PASS "$name"
	else
		echo "=== FAIL $name (exit $rc, ${secs}s)"
		verdict FAIL "$name"
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

# The tripwires the scoped tier runs on every change (ADR-010 Amendment A5
# item 2): the spec's and generated data's own tests, named exactly. A run
# that matches fewer tests than the list names fails, so a renamed test
# cannot drop out of the tier unnoticed. internal/finding and
# internal/specdata are tripwires whole, and are tested as packages.
tripwire_names() {
	case "$1" in
	tripwire-conformance) echo 'TestVectorsWellFormed TestVectorsWellFormedControl TestNoSentinelInSuite TestNoSentinelInSuiteControl' ;;
	tripwire-help) echo 'TestHelpGolden TestHelpCoverage TestHelpForm' ;;
	tripwire-spec) echo 'TestStatementsWellFormed TestStatementsWellFormedControl TestStatementIndex TestStatementIndexControl TestStatementIndexForm TestKeywordsOnlyInStatements TestKeywordsOnlyInStatementsControl TestRecommendationsNotObservable TestRecommendationsNotObservableControl TestProofNotes TestProofNotesControl TestProofNotesOnTheSpecification TestCoverage TestCoverageControl TestCoverageTable TestCitationsResolve TestCitationsResolveControl TestFindingCatalog TestFindingCatalogControl TestFindingCatalogForm TestFindingCodesCovered TestFindingCodesCoveredControl TestExamplesMatchVectors TestExamplesMatchVectorsControl TestReleasedFilesUnchanged TestReleasedFilesUnchangedControl TestChangelog TestChangelogControl TestSchemasPublished TestSchemasPublishedControl' ;;
	esac
}

tripwire_pkg() {
	case "$1" in
	tripwire-conformance) echo github.com/kottera/metais/internal/conformance ;;
	tripwire-help) echo github.com/kottera/metais/internal/cli ;;
	tripwire-spec) echo github.com/kottera/metais/internal/guarantees ;;
	esac
}

tripwire() {
	local names pkg pat n out missing="" rc
	names=$(tripwire_names "$1")
	pkg=$(tripwire_pkg "$1")
	pat="^($(echo $names | tr ' ' '|'))\$"
	out=$(go test -race -count=1 -p "$p" -parallel "$parallel" -v -run "$pat" "$pkg" 2>&1)
	rc=$?
	printf '%s\n' "$out" | grep -vE '^(=== (RUN|PAUSE|CONT|NAME)|    --- PASS|--- PASS)'
	for n in $names; do
		printf '%s\n' "$out" | grep -qE "^--- (PASS|FAIL|SKIP): $n " || missing="$missing $n"
	done
	if [ -n "$missing" ]; then
		echo "tripwire $1 ran none of:$missing (renamed or removed? update gate.sh with the tier's amendment)"
		return 1
	fi
	return $rc
}

# test_packages runs `go test -race -count=1` over the packages listed in $1
# and records one verdict per package from go test's own summary lines.
# Neither tier takes a verdict from Go's test cache: the cache cannot see the
# binary or the module listing that the blind packages' tests build, and
# --resume reuses what is still true, by gate-select.sh's rules.
test_packages() {
	local pkgs
	pkgs=$(cat "$1")
	echo "--- go test -race -count=1 -p $p -parallel $parallel ($(echo "$pkgs" | wc -l | tr -d ' ') package(s))"
	# shellcheck disable=SC2086
	go test -race -count=1 -p "$p" -parallel "$parallel" $pkgs 2>&1 | tee "$state/$run.out"
	awk '/^ok  	/ {print "PASS\t" $2} /^FAIL	[^ ]/ {print "FAIL\t" $2} /^\?   	/ {print "PASS\t" $2}' "$state/$run.out" |
		while IFS=$'\t' read -r st pkg; do
			verdict "$st" "test $pkg"
		done
	rm -f "$state/$run.out"
}

# tests runs the tier's packages, reusing what --resume allows. The step
# passes only when every package it ran has a PASS of its own: a package
# whose binary was killed before it reported has no verdict, and fails it.
tests() {
	local want=$1 pkg todo="$state/$run.todo" rc=0 prior st
	: >"$todo"
	while IFS= read -r pkg; do
		[ -n "$pkg" ] || continue
		if prior=$(reusable "test $pkg"); then
			echo "=== REUSED test $pkg (PASS in ${prior#*	})"
			verdict PASS "test $pkg" "${prior%%	*}" "${prior#*	}"
		else
			echo "$pkg" >>"$todo"
		fi
	done <"$want"
	[ -s "$todo" ] && test_packages "$todo"
	while IFS= read -r pkg; do
		st=$(awk -F'\t' -v k="test $pkg" -v r="$run" '$2 == k && $4 == r {s = $1} END {print s}' "$record")
		case "$st" in
		PASS) ;;
		FAIL) echo "FAIL: $pkg"; rc=1 ;;
		*) echo "NO VERDICT: $pkg (it never reported)"; verdict FAIL "test $pkg"; rc=1 ;;
		esac
	done <"$todo"
	rm -f "$todo"
	return $rc
}

want="$state/$run.want"
if [ "$tier" = wave ]; then
	go list ./... >"$want" || exit 2
	step build go build ./...
	step vet go vet ./...
	echo "=== STEP test"
	start=$(date +%s)
	if tests "$want"; then
		echo "=== PASS test ($(($(date +%s) - start))s)"
	else
		echo "=== FAIL test ($(($(date +%s) - start))s)"
		failed="$failed test"
	fi
	step tidy go mod tidy -diff
	step gofmt gofmt_clean
	step static static_build
	step cross cross
	step specdata specdata_fresh
else
	sel="$state/$run.select"
	if [ -n "$base" ]; then
		"$here/gate-select.sh" --base "$base" >"$sel" || exit 2
	else
		"$here/gate-select.sh" >"$sel" || exit 2
	fi
	echo "--- the change (gate-select.sh)"
	grep -E '^(base|changed|seam|touched) ' "$sel"
	if grep -q '^seam ' "$sel"; then
		echo "NOTE: this change touches a shared seam; it also needs the wave tier (gate.sh wave)"
	fi
	{
		grep -E '^(test|blind) ' "$sel" | cut -d' ' -f2
		echo github.com/kottera/metais/internal/finding
		echo github.com/kottera/metais/internal/specdata
	} | sort -u >"$want"
	step build go build ./...
	step vet go vet ./...
	step gofmt gofmt_clean
	step tidy go mod tidy -diff
	step static static_build
	step cross cross
	echo "=== STEP test"
	start=$(date +%s)
	if tests "$want"; then
		echo "=== PASS test ($(($(date +%s) - start))s)"
	else
		echo "=== FAIL test ($(($(date +%s) - start))s)"
		failed="$failed test"
	fi
	for t in tripwire-conformance tripwire-help tripwire-spec; do
		if grep -qxF "$(tripwire_pkg "$t")" "$want"; then
			echo "=== COVERED $t (its package is tested whole)"
			continue
		fi
		step "$t" tripwire "$t"
	done
	step specdata specdata_fresh
fi

rm -f "$want" "$state/$run".diff.* "$state/$run".reach.*

# A tree that changed while the run was live leaves verdicts that describe no
# one tree. They stand for this run's report, but --resume never reuses them.
if [ "$(snapshot)" != "$tree" ]; then
	echo "WARNING: the working tree changed while this run was live; --resume will not reuse its verdicts"
	awk -F'\t' -v OFS='\t' -v r="$run" '$4 == r {$3 = "moved-" $3} {print}' "$record" >"$record.tmp" &&
		mv "$record.tmp" "$record"
fi
echo "=== SUMMARY"
grep -E '^=== (PASS|FAIL|REUSED|COVERED) ' "$log" | grep -v '^=== REUSED test '
echo "packages: $(grep -c '^PASS	test ' "$record") passed, $(grep -c '^FAIL	test ' "$record") failed ($(grep -c '^=== REUSED test ' "$log") reused)"
if [ -z "$failed" ]; then
	echo "GATE DONE: PASS ($tier)"
	exit 0
fi
echo "GATE DONE: FAIL ($tier):$failed"
exit 1
