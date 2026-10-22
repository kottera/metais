#!/usr/bin/env bash
# shellcheck disable=SC2317 # step runs each step's function through "$@", which shellcheck cannot follow
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
#                    reached on, the run that reached it, and for a
#                    package's tests what its run was given (below)
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
# The delivered share: every go test runs its test binaries through
# gate-exec.sh (go test -exec), which records each binary's CPU (user plus
# system time, its own and every process it waited for), its start and end,
# and the host's load average when it ended. Measuring runs nothing beside
# the binary, so it takes no CPU from the host. A package's verdict keeps
#
#   cpu=<s> wall=<s> cores=<cpu/wall> beside=<n> p=<share> load=<load>/<cores>
#
# where beside is how many other test binaries of the run ran beside it, on
# average over its wall time. --resume keeps the measure of a verdict it
# reuses. A red package's FAIL line names the share it got against p, its
# CPU, the host's load against its cores, and the same package's last green
# in .local/gate with its CPU, so one run tells a starved red from one that
# did more work (the root-cause skill reads it). The test step's PASS or
# FAIL line names the cores the whole go test got, builds included,
# measured the same way around it.
#
# Every run first runs merge-setup.sh, which installs the generated files'
# merge drivers in this clone's configuration and says so, on the terminal
# and in the log, when they were missing.
#
# Exit 0 when every item passes, 1 when any fails, 2 when it cannot start.

set -u

here=$(cd "$(dirname "$0")" && pwd) || exit 2
# shellcheck source=git-env.sh
. "$here/git-env.sh" || exit 2
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
	if [ -z "$prev" ] || { [ ! -f "$state/$prev.record" ] && [ ! -f "$state/$prev.reuse" ]; }; then
		echo "gate: --resume found no earlier $tier run to resume; start a plain run" >&2
		exit 2
	fi
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
gexec="$here/gate-exec.sh"
cpufile="$state/$run.cpu"
dirs="$state/$run.dirs"
: >"$cpufile"
go list -f '{{.Dir}}{{"\t"}}{{.ImportPath}}' ./... >"$dirs" || exit 2
# A sandbox can hide /proc/loadavg and uptime; linux still answers
# sysinfo(2), which loadavg.go reads.
if [ "$("$gexec" --load)" = "?" ] && [ "$(uname -s)" = Linux ] &&
	go build -o "$state/loadavg" "$here/loadavg.go"; then
	export METAIS_GATE_LOADAVG="$state/loadavg"
fi

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
	return "$rc"
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
echo "load: $("$gexec" --load) on $ncpu cores at the start"
echo "measure: per package, the CPU of its test binary and every process it waited for (gate-exec.sh)"
[ -z "$prev" ] || echo "resumes: $prev"

failed=""

# verdict records an item's status against the tree it was reached on, and
# the measure of the run that reached it, when it has one.
verdict() {
	printf '%s\t%s\t%s\t%s\t%s\n' "$1" "$2" "${3:-$tree}" "${4:-$run}" "${5:-}" >>"$record"
}

# gotest is go test under the gate's share, every test binary measured.
gotest() {
	METAIS_GATE_CPU=$cpufile go test -race -count=1 -p "$p" -parallel "$parallel" -exec "'$gexec'" "$@"
}

# cpu_secs prints the reaped children's user plus system seconds from a file
# bash's times wrote. times runs in the gate's own shell, never in a command
# substitution, whose fork has no children's times.
cpu_secs() {
	LC_ALL=C awk 'NR == 2 {
		n = 0
		for (i = 1; i <= 2; i++) {
			gsub(",", ".", $i)
			split($i, a, "m")
			sub("s", "", a[2])
			n += a[1] * 60 + a[2]
		}
		printf "%.1f", n
	}' "$1"
}

# measure <pkg> prints the record's measure of the package's newest test
# binary in this run, or nothing when no binary of it ran.
measure() {
	local dir
	dir=$(awk -F'\t' -v k="$1" '$2 == k {print $1; exit}' "$dirs")
	[ -n "$dir" ] || return 0
	LC_ALL=C awk -F'\t' -v d="$dir" -v p="$p" -v n="$ncpu" '
		{ s[NR] = $2; e[NR] = $3; c[NR] = $4; l[NR] = $6; if ($1 == d) me = NR }
		END {
			if (!me) exit
			w = e[me] - s[me]
			b = 0
			for (i = 1; i <= NR; i++) {
				if (i == me) continue
				lo = s[i] > s[me] ? s[i] : s[me]
				hi = e[i] < e[me] ? e[i] : e[me]
				if (hi > lo) b += hi - lo
			}
			if (w > 0) printf "cpu=%.1fs wall=%.1fs cores=%.2f beside=%.1f p=%d load=%s/%d\n", c[me], w, c[me] / w, b / w, p, l[me], n
			else printf "cpu=%.1fs wall=%.1fs cores=? beside=? p=%d load=%s/%d\n", c[me], w, p, l[me], n
		}' "$cpufile"
}

# share_line <item> <measure> prints a red's share, in words: what it got
# against p, its CPU, the host's load against its cores, and the item's last
# green in .local/gate, from any run but this one, with its CPU beside this.
share_line() {
	local item=$1 m=$2 green
	if [ -z "$m" ]; then
		echo "not measured (no test binary of it ran to its end: a build failure, or killed first)"
		return
	fi
	green=$(cat "$state"/*.record 2>/dev/null | awk -F'\t' -v k="$item" -v r="$run" '
		$1 == "PASS" && $2 == k && $4 != r && $5 ~ /cpu=/ {
			t = $4; sub(/^[^-]*-/, "", t)
			if (t > best) { best = t; line = $4 "\t" $5 }
		}
		END { if (line != "") print line }')
	LC_ALL=C awk -v m="$m" -v g="$green" 'BEGIN {
		split(m, f, " "); for (i in f) { split(f[i], kv, "="); v[kv[1]] = kv[2] }
		split(v["load"], ld, "/")
		printf "got %s of %s cores (CPU %s over %s wall, beside %s other test binaries); host load %s on %s cores; ", v["cores"], v["p"], v["cpu"], v["wall"], v["beside"], ld[1], ld[2]
		if (g == "") { print "no green of it in .local/gate to compare with"; exit }
		split(g, gr, "\t"); split(gr[2], h, " "); for (i in h) { split(h[i], kv, "="); x[kv[1]] = kv[2] }
		cn = v["cpu"] + 0; cg = x["cpu"] + 0
		printf "last green %s got %s of %s cores (CPU %s over %s wall)", gr[1], x["cores"], x["p"], x["cpu"], x["wall"]
		if (cg > 0) printf ", CPU now %.2fx it", cn / cg
		print ""
	}'
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
	cut -f3 "$reuse" | sort -u | while read -r t; do
		[ "$t" = "$tree" ] && continue
		if ! { git diff-tree -r --name-only --no-renames "$t" "$tree" >"$state/$run.diff.$t" 2>/dev/null &&
			"$here/gate-select.sh" --paths "$state/$run.diff.$t" >"$state/$run.reach.$t"; }; then
			: >"$state/$run.reach.$t.unknown"
		fi
	done
fi

# reusable <item> prints the earlier verdict's tree and run, and succeeds,
# when that verdict is a PASS the change since it does not reach.
reusable() {
	local item=$1 line st t from m facts
	[ -n "$prev" ] || return 1
	line=$(awk -F'\t' -v k="$item" '$2 == k' "$reuse")
	[ -n "$line" ] || return 1
	st=$(printf '%s' "$line" | cut -f1)
	t=$(printf '%s' "$line" | cut -f3)
	from=$(printf '%s' "$line" | cut -f4)
	m=$(printf '%s' "$line" | cut -f5)
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
	printf '%s\t%s\t%s\n' "$t" "$from" "$m"
}

step() {
	local name=$1
	shift
	local prior
	if prior=$(reusable "$name"); then
		echo "=== REUSED $name (PASS in $(printf '%s' "$prior" | cut -f2))"
		verdict PASS "$name" "$(printf '%s' "$prior" | cut -f1)" "$(printf '%s' "$prior" | cut -f2)" "$(printf '%s' "$prior" | cut -f3)"
		return
	fi
	echo "=== STEP $name"
	local start rc secs
	start=$(date +%s)
	step_measure=""
	"$@"
	rc=$?
	secs=$(($(date +%s) - start))
	if [ "$rc" -eq 0 ]; then
		echo "=== PASS $name (${secs}s)"
		verdict PASS "$name" "" "" "$step_measure"
	else
		echo "=== FAIL $name (exit $rc, ${secs}s)"
		verdict FAIL "$name" "" "" "$step_measure"
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
	return "$rc"
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
	tripwire-conformance) echo metais.org/metais/internal/conformance ;;
	tripwire-help) echo metais.org/metais/internal/cli ;;
	tripwire-spec) echo metais.org/metais/internal/guarantees ;;
	esac
}

tripwire() {
	local names pkg pat n out missing="" rc
	names=$(tripwire_names "$1")
	pkg=$(tripwire_pkg "$1")
	pat="^($(echo "$names" | tr ' ' '|'))\$"
	out=$(gotest -v -run "$pat" "$pkg" 2>&1)
	rc=$?
	printf '%s\n' "$out" | grep -vE '^(=== (RUN|PAUSE|CONT|NAME)|    --- PASS|--- PASS)'
	step_measure=$(measure "$pkg")
	[ "$rc" -eq 0 ] || echo "FAIL: $pkg ($1): $(share_line "$1" "$step_measure")"
	for n in $names; do
		printf '%s\n' "$out" | grep -qE "^--- (PASS|FAIL|SKIP): $n " || missing="$missing $n"
	done
	if [ -n "$missing" ]; then
		echo "tripwire $1 ran none of:$missing (renamed or removed? update gate.sh with the tier's amendment)"
		return 1
	fi
	return "$rc"
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
	gotest $pkgs 2>&1 | tee "$state/$run.out"
	awk '/^ok  	/ {print "PASS\t" $2} /^FAIL	[^ ]/ {print "FAIL\t" $2} /^\?   	/ {print "PASS\t" $2}' "$state/$run.out" |
		while IFS=$'\t' read -r st pkg; do
			verdict "$st" "test $pkg" "" "" "$(measure "$pkg")"
		done
	rm -f "$state/$run.out"
}

# tests runs the tier's packages, reusing what --resume allows. The step
# passes only when every package it ran has a PASS of its own: a package
# whose binary was killed before it reported has no verdict, and fails it.
tests() {
	local want=$1 pkg todo="$state/$run.todo" rc=0 prior st m
	: >"$todo"
	while IFS= read -r pkg; do
		[ -n "$pkg" ] || continue
		if prior=$(reusable "test $pkg"); then
			echo "=== REUSED test $pkg (PASS in $(printf '%s' "$prior" | cut -f2))"
			verdict PASS "test $pkg" "$(printf '%s' "$prior" | cut -f1)" "$(printf '%s' "$prior" | cut -f2)" "$(printf '%s' "$prior" | cut -f3)"
		else
			echo "$pkg" >>"$todo"
		fi
	done <"$want"
	[ -s "$todo" ] && test_packages "$todo"
	while IFS= read -r pkg; do
		st=$(awk -F'\t' -v k="test $pkg" -v r="$run" '$2 == k && $4 == r {s = $1} END {print s}' "$record")
		case "$st" in
		PASS) m=$(measure "$pkg"); echo "measured $pkg: ${m:-no test binary}" ;;
		FAIL) echo "FAIL: $pkg: $(share_line "test $pkg" "$(measure "$pkg")")"; rc=1 ;;
		*)
			echo "NO VERDICT: $pkg (it never reported)"
			echo "FAIL: $pkg: $(share_line "test $pkg" "$(measure "$pkg")")"
			verdict FAIL "test $pkg" "" "" "$(measure "$pkg")"
			rc=1
			;;
		esac
	done <"$todo"
	rm -f "$todo"
	return "$rc"
}

# The SC7 reach (R-W6-3, plan orchestrator, 2026-10-20, extending ADR-010
# Amendment A5 item 1): the wave tier runs ADR-009 D16 item 1's
# every-command reach over every vector, which every go test runs only over
# the canaries' probes. Over the whole suite it is more than one go test
# binary can run within its ten-minute timeout at the gate's share, so it
# runs as one step per group of at most reach_chunk vectors of one area,
# each its own `go test -run` invocation under the same timeout and share.
reach_chunk=${METAIS_GATE_REACH_CHUNK:-25}

# reach_groups prints one line per group: the area, then its vectors' slugs.
reach_groups() {
	local area
	for area in spec/conformance/vectors/*/; do
		area=$(basename "$area")
		find "spec/conformance/vectors/$area" -name vector.json |
			sed -e "s|^spec/conformance/vectors/$area/||" -e 's|/vector.json$||' | sort |
			awk -v a="$area" -v n="$reach_chunk" '{s = s " " $0} NR % n == 0 {print a s; s = ""} END {if (s != "") print a s}'
	done
}

# reach runs the reach over one group: TestHostIsolation with
# -conformance.reach, selecting the area and the group's vectors. A run that
# ran fewer vectors than the group names fails, so a selection that matches
# nothing cannot pass.
reach() {
	local area=$1 out rc n
	shift
	out=$(gotest -v \
		-run "^TestHostIsolation\$/^$area\$/^($(echo "$@" | tr ' ' '|'))\$" \
		./internal/conformance -args -conformance.reach 2>&1)
	rc=$?
	printf '%s\n' "$out" | grep -vE '^(=== (RUN|PAUSE|CONT|NAME)|[[:space:]]*--- PASS)'
	step_measure=$(measure metais.org/metais/internal/conformance)
	[ "$rc" -eq 0 ] || echo "FAIL: metais.org/metais/internal/conformance (reach $area $1): $(share_line "reach $area $1" "$step_measure")"
	n=$(printf '%s\n' "$out" | sed -n 's/.*ran \([0-9][0-9]*\) vectors.*/\1/p' | head -1)
	if [ "${n:-0}" -ne $# ]; then
		echo "reach $area ran ${n:-no} vectors of the $# it selects"
		return 1
	fi
	return "$rc"
}

# test_step runs the tier's packages as the step "test". Its PASS or FAIL
# line names the cores the whole go test got against p, builds included:
# the CPU of everything the gate's shell reaped while it ran, over its wall
# time.
test_step() {
	local start c0 c1 secs share rc
	echo "=== STEP test"
	times >"$state/$run.times"
	c0=$(cpu_secs "$state/$run.times")
	start=${EPOCHREALTIME:-$(date +%s)}
	if tests "$want"; then rc=0; else rc=1; fi
	secs=$(LC_ALL=C awk -v a="${start/,/.}" -v b="$(e=${EPOCHREALTIME:-$(date +%s)}; echo "${e/,/.}")" 'BEGIN {printf "%.0f", b - a}')
	times >"$state/$run.times"
	c1=$(cpu_secs "$state/$run.times")
	rm -f "$state/$run.times"
	share=$(LC_ALL=C awk -v c="$c1" -v d="$c0" -v w="$secs" -v p="$p" -v l="$("$gexec" --load)" -v n="$ncpu" 'BEGIN {
		if (w > 0) printf "the gate got %.2f of %d cores, CPU %.1fs; host load %s on %d cores", (c - d) / w, p, c - d, l, n
		else printf "CPU %.1fs; host load %s on %d cores", c - d, l, n
	}')
	if [ "$rc" -eq 0 ]; then
		echo "=== PASS test (${secs}s; $share)"
	else
		echo "=== FAIL test (${secs}s; $share)"
		failed="$failed test"
	fi
}

want="$state/$run.want"
if [ "$tier" = wave ]; then
	go list ./... >"$want" || exit 2
	step build go build ./...
	step vet go vet ./...
	test_step
	step tidy go mod tidy -diff
	step gofmt gofmt_clean
	step static static_build
	step cross cross
	step specdata specdata_fresh
	while read -r area slugs; do
		# shellcheck disable=SC2086
		step "reach $area $(echo $slugs | cut -d' ' -f1)" reach "$area" $slugs
	done < <(reach_groups)
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
		echo metais.org/metais/internal/finding
		echo metais.org/metais/internal/specdata
	} | sort -u >"$want"
	step build go build ./...
	step vet go vet ./...
	step gofmt gofmt_clean
	step tidy go mod tidy -diff
	step static static_build
	step cross cross
	test_step
	for t in tripwire-conformance tripwire-help tripwire-spec; do
		if grep -qxF "$(tripwire_pkg "$t")" "$want"; then
			echo "=== COVERED $t (its package is tested whole)"
			continue
		fi
		step "$t" tripwire "$t"
	done
	step specdata specdata_fresh
fi

rm -f "$want" "$state/$run".diff.* "$state/$run".reach.* "$dirs"

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
