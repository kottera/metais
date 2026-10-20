#!/usr/bin/env bash
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# merge-check.sh shows the generated files' merge drivers working on a fresh
# clone of HEAD, from the repository root:
#
#   merge-check.sh           run every case, and remove the clone
#   merge-check.sh --keep    keep the clone, and print where it is
#
# Each case builds two branches from HEAD, rebases one onto the other, runs
# regenerate.sh and the drift tests, and checks what each step left:
#
#   control     with no drivers configured, the statements case stops on a
#               conflict in spec/statements.json, and merge-setup.sh --check
#               says the drivers are missing
#   statements  two new statements, in two sections of one chapter: the
#               rebase resolves spec/statements.json to one side, and
#               regenerating brings back the other's statement
#   catalog     two new finding codes whose catalog rows and registry entries
#               meet, and a prose edit on each side of the catalog in
#               spec/12-findings.md: the prose of both sides survives the
#               rebase, the catalog keeps one side, and regenerating brings
#               back the other's code
#   prose       one line of spec/12-findings.md's prose edited two ways: the
#               rebase stops on a conflict in it, so no side of hand-written
#               text is taken
#   collision   two statements that take the same identifier: the rebase
#               resolves spec/statements.json, and after regenerating
#               TestStatementsWellFormed fails on the duplicate (ADR-011 D3
#               item 7 as amended by A1)
#
# It clones HEAD, so commit first: uncommitted edits are not in the clone.
# Exit 0 when every case holds, 1 when one does not, 2 when it cannot run.

set -u

keep=""
case "${1:-}" in
"") ;;
--keep) keep=1 ;;
*)
	echo "merge-check: unknown argument $1 (usage: merge-check.sh [--keep])" >&2
	exit 2
	;;
esac

. "$(dirname "$0")/git-env.sh" || exit 2
root=$(git rev-parse --show-toplevel 2>/dev/null) || {
	echo "merge-check: not inside a git checkout" >&2
	exit 2
}
head=$(git -C "$root" rev-parse HEAD) || exit 2
[ -z "$(git -C "$root" status --porcelain --untracked-files=no)" ] ||
	echo "merge-check: NOTE: the checkout has uncommitted edits; the clone holds HEAD ($head) without them"
export GOTOOLCHAIN=auto

work=$(mktemp -d "${TMPDIR:-/tmp}/merge-check.XXXXXX") || exit 2
if [ -n "$keep" ]; then
	trap 'echo "merge-check: the clone is $work/clone"' EXIT
else
	trap 'rm -rf "$work"' EXIT
fi
git clone -q --no-local "$root" "$work/clone" || exit 2
cd "$work/clone" || exit 2
git checkout -q --detach "$head" || exit 2
git config user.name merge-check
git config user.email merge-check@invalid
skills=.claude/skills/verification-gate
echo "clone: $work/clone at $head"
echo "git:   $(git --version)"
echo "go:    $(go version)"

failed=""
ok() { echo "  ok: $*"; }
bad() {
	echo "  FAILED: $*"
	failed="$failed $case"
}
expect() { # expect <description> <command...>
	local what=$1
	shift
	if "$@" >/dev/null 2>&1; then ok "$what"; else bad "$what"; fi
}

# fresh puts the clone back at HEAD, with no branch but the detached head.
fresh() {
	git rebase --abort >/dev/null 2>&1
	git reset -q --hard && git clean -qfd && git checkout -q --detach "$head" || exit 2
	git for-each-ref --format='%(refname:short)' refs/heads | while read -r b; do git branch -qD "$b"; done
}

# branch <name> <edit-function> makes a branch from HEAD whose one commit is
# the edit and the regenerated files it changes.
branch() {
	git checkout -q -b "$1" "$head" || exit 2
	"$2" || exit 2
	go generate ./internal/finding ./internal/specdata ./internal/guarantees >"$work/gen.$1" 2>&1 || {
		cat "$work/gen.$1"
		exit 2
	}
	git add -u && git commit -qm "$1" || exit 2
}

# rebase <branch> <onto> rebases and prints git's output; its status is git's.
rebase() {
	local rc
	git checkout -q "$1" || exit 2
	git rebase "$2" >"$work/rebase.out" 2>&1
	rc=$?
	sed 's/^/    | /' "$work/rebase.out"
	return $rc
}

regen() {
	"$skills/regenerate.sh" >"$work/regen.out" 2>&1
	local rc=$?
	grep -E '^(--- |regenerate:|merge-setup:)| [MADR] ' "$work/regen.out" | sed 's/^/    | /'
	return $rc
}

drift() { # drift <go test -run pattern> <package>...
	local pat=$1
	shift
	go test -count=1 -run "$pat" "$@" >"$work/drift.out" 2>&1
	local rc=$?
	sed 's/^/    | /' "$work/drift.out" | grep -vE '^    \| (=== RUN|--- PASS)' | head -20
	return $rc
}

statement() { # statement <number> prints a statement of MTS-AUT-<number>
	printf '<a id="mts-aut-%s"></a>**MTS-AUT-%s** A merge check MUST find this statement %s.\n' "$1" "$1" "$2"
}
# end13 appends a statement to the end of chapter 13; in136 puts one first
# in its section 13.6.
end13() { { echo; statement "$1" "$2"; } >>spec/13-authoring.md; }
in136() {
	statement "$1" "$2" >"$work/stmt"
	awk -v f="$work/stmt" '{ print } /^## 13\.6 / { print ""; while ((getline l < f) > 0) print l }' \
		spec/13-authoring.md >"$work/ch" && cat "$work/ch" >spec/13-authoring.md
}
# The planted statements take the two numbers after the highest MTS-AUT
# number anywhere in HEAD's tree. Section 1.4 assigns an area's numbers in
# order of addition and never reuses one, so no statement HEAD holds can
# have either, however chapter 13 grows; a fixed pair is taken the day the
# specification reaches it.
top=$(git grep -ohE 'MTS-AUT-[0-9]+' "$head" -- . | sed 's/^MTS-AUT-0*//' | sort -n | tail -1)
case "$top" in
"" | *[!0-9]*)
	echo "merge-check: HEAD holds no MTS-AUT statement to number after" >&2
	exit 2
	;;
esac
n1=$(printf '%03d' $((top + 1)))
n2=$(printf '%03d' $((top + 2)))
echo "statements: MTS-AUT-$n1 and MTS-AUT-$n2, after HEAD's highest, MTS-AUT-$(printf '%03d' "$top")"
first_end() { end13 "$n1" "at the end of the chapter"; }
second_136() { in136 "$n2" "in section 13.6"; }
first_136() { in136 "$n1" "in section 13.6"; }

# The catalog case's edits: a code at the end of the collection family and
# one at the start of the csv family, whose catalog rows are adjacent, and a
# paragraph of prose on each side of the catalog.
entry() {
	cat <<EOF
	{
		Code:       "$1",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "The collection {input} holds a file the merge check added.",
		Hint:       "Nothing to do; this code exists only in the merge check's clone.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-015"},
	},
EOF
}
code_one() {
	entry collection.zz_merge_check_one >"$work/entry"
	awk -v f="$work/entry" '/^}$/ && !done { while ((getline l < f) > 0) print l; done = 1 } { print }' \
		internal/finding/registry_collection.go >"$work/go" && cat "$work/go" >internal/finding/registry_collection.go
	awk '{ print } /^## 12\.1 / { print ""; print "Prose added above the catalog by the first branch." }' \
		spec/12-findings.md >"$work/ch" && cat "$work/ch" >spec/12-findings.md
}
code_two() {
	entry csv.aa_merge_check_two >"$work/entry"
	awk -v f="$work/entry" '{ print } /^var csvCodes = \[\]Entry\{$/ { while ((getline l < f) > 0) print l }' \
		internal/finding/registry_csv.go >"$work/go" && cat "$work/go" >internal/finding/registry_csv.go
	awk '{ print } /^## 12\.2 / { print ""; print "Prose added below the catalog by the second branch." }' \
		spec/12-findings.md >"$work/ch" && cat "$work/ch" >spec/12-findings.md
}
prose_one() { sed -i.bak 's/^the one source of this table\./the one source of this table, as the first branch says./' spec/12-findings.md && rm -f spec/12-findings.md.bak; }
prose_two() { sed -i.bak 's/^the one source of this table\./the one source of this table, as the second branch says./' spec/12-findings.md && rm -f spec/12-findings.md.bak; }

has() { grep -qF -- "$2" "$1"; }
hasnt() { ! grep -qF -- "$2" "$1"; }
# in_catalog <code> succeeds when the code has a row between the markers.
in_catalog() {
	awk -v c="| \`$1\` |" '/^<!-- .*generated.*Do not edit/ { open = 1; next } /^<!-- The end of the generated/ { open = 0 } open && index($0, c) == 1 { found = 1 } END { exit !found }' spec/12-findings.md
}
not_in_catalog() { ! in_catalog "$1"; }
no_markers() { ! grep -qE '^(<<<<<<<|>>>>>>>)' "$1"; }

case=control
echo
echo "=== $case: no drivers configured"
"$skills/merge-setup.sh" --check >"$work/setup.out" 2>&1
rc=$?
sed 's/^/    | /' "$work/setup.out"
if [ "$rc" -eq 0 ]; then bad "merge-setup.sh --check passes in a clone without the drivers"; else ok "merge-setup.sh --check says the drivers are missing"; fi
fresh
branch one first_end
branch two second_136
if rebase two one; then
	bad "the rebase succeeded with no driver configured"
else
	expect "the rebase stopped on a conflict in spec/statements.json" sh -c 'git diff --name-only --diff-filter=U | grep -qx spec/statements.json'
fi

echo
echo "=== setup"
"$skills/merge-setup.sh" | sed 's/^/    | /'
expect "merge-setup.sh --check passes" "$skills/merge-setup.sh" --check

case=statements
echo
echo "=== $case: two new statements, in two sections of one chapter"
fresh
branch one first_end
branch two second_136
if rebase two one; then ok "the rebase resolved"; else bad "the rebase stopped"; fi
expect "spec/statements.json holds one side: MTS-AUT-$n1" has spec/statements.json "\"MTS-AUT-$n1\""
expect "spec/statements.json lacks the other side's MTS-AUT-$n2 before regenerating" hasnt spec/statements.json "\"MTS-AUT-$n2\""
if regen; then ok "regenerate.sh ran"; else bad "regenerate.sh failed"; fi
expect "spec/statements.json holds MTS-AUT-$n1 after regenerating" has spec/statements.json "\"MTS-AUT-$n1\""
expect "spec/statements.json holds MTS-AUT-$n2 after regenerating" has spec/statements.json "\"MTS-AUT-$n2\""
if drift '^(TestStatementIndex|TestStatementsWellFormed|TestFindingCatalog)$' ./internal/guarantees; then ok "the drift tests pass"; else bad "the drift tests fail"; fi

case=catalog
echo
echo "=== $case: two finding codes whose catalog rows meet, and prose on each side of the catalog"
fresh
branch one code_one
branch two code_two
if rebase two one; then ok "the rebase resolved"; else bad "the rebase stopped"; fi
expect "spec/12-findings.md holds no conflict markers" no_markers spec/12-findings.md
expect "the first branch's prose survived" has spec/12-findings.md 'Prose added above the catalog by the first branch.'
expect "the second branch's prose survived" has spec/12-findings.md 'Prose added below the catalog by the second branch.'
expect "the catalog holds one side: collection.zz_merge_check_one" in_catalog collection.zz_merge_check_one
expect "the catalog lacks the other side's csv.aa_merge_check_two before regenerating" not_in_catalog csv.aa_merge_check_two
expect "both registry table entries merged as Go source" sh -c 'grep -q zz_merge_check_one internal/finding/registry_collection.go && grep -q aa_merge_check_two internal/finding/registry_csv.go'
if regen; then ok "regenerate.sh ran"; else bad "regenerate.sh failed"; fi
expect "the catalog holds collection.zz_merge_check_one after regenerating" in_catalog collection.zz_merge_check_one
expect "the catalog holds csv.aa_merge_check_two after regenerating" in_catalog csv.aa_merge_check_two
expect "spec/findings/registry.json holds both codes after regenerating" sh -c 'grep -q zz_merge_check_one spec/findings/registry.json && grep -q aa_merge_check_two spec/findings/registry.json'
expect "both sides' prose is still there" sh -c "grep -q 'by the first branch' spec/12-findings.md && grep -q 'by the second branch' spec/12-findings.md"
if drift '^(TestFindingCatalog|TestFindingCatalogForm|TestStatementIndex)$' ./internal/guarantees &&
	drift '^TestRegistryFileGenerated$' ./internal/finding &&
	drift '' ./internal/specdata; then
	ok "the drift tests pass"
else
	bad "the drift tests fail"
fi

case=prose
echo
echo "=== $case: one line of spec/12-findings.md's prose, edited two ways"
fresh
branch one prose_one
branch two prose_two
if rebase two one; then
	bad "the rebase resolved, taking one side of hand-written text"
else
	expect "the rebase stopped on a conflict in spec/12-findings.md" sh -c 'git diff --name-only --diff-filter=U | grep -qx spec/12-findings.md'
	expect "the conflict markers hold both sides' prose" sh -c "grep -q '^<<<<<<<' spec/12-findings.md && grep -q 'as the first branch says' spec/12-findings.md && grep -q 'as the second branch says' spec/12-findings.md"
	expect "the catalog is intact" in_catalog yaml.tag
fi

case=collision
echo
echo "=== $case: two statements that take the same identifier"
fresh
branch one first_end
branch two first_136
if rebase two one; then ok "the rebase resolved"; else bad "the rebase stopped"; fi
if regen; then ok "regenerate.sh ran"; else bad "regenerate.sh failed"; fi
if drift '^TestStatementsWellFormed$' ./internal/guarantees; then
	bad "TestStatementsWellFormed passed with MTS-AUT-$n1 used twice"
else
	expect "TestStatementsWellFormed names the duplicate" grep -q "MTS-AUT-$n1 is already used" "$work/drift.out"
fi

fresh
echo
if [ -n "$failed" ]; then
	echo "MERGE-CHECK FAIL:$failed"
	exit 1
fi
echo "MERGE-CHECK PASS"
