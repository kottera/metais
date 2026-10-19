#!/usr/bin/env bash
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# gate-select.sh decides what a change reaches: the packages the scoped tier
# tests, and whether the change touches a shared seam that needs the wave
# tier. It is ADR-010 Amendment A5's definition of "touched", and gate.sh
# runs it. Run it yourself to see what a gate run will test.
#
#   gate-select.sh                  the change since the merge base with origin/main
#   gate-select.sh --base <ref>     the change since <ref>
#   gate-select.sh --paths <file>   the paths listed in <file>, one per line
#
# The change is every path that differs between the base and the working
# tree: committed, staged, unstaged and untracked (not ignored), deletions
# included. Each line of output is one fact:
#
#   base <sha>          the base the change was measured from
#   changed <path>      a changed path
#   seam <path>         a changed path on a shared seam: run the wave tier
#   code <path>         a changed path the build reads: in a package's
#                       directory, or go.mod or go.sum
#   touched <pkg> <why> a package the change touches directly
#   test <pkg>          a package to test: touched, or importing one
#   blind <pkg>         a package to test whose tests build or list the module
#                       with the go command, so no import edge shows what
#                       they read, and the test cache cannot see it
#
# A path reaches packages by these rules:
#
#   - A path inside a package's directory (its Go files, testdata, embedded
#     files) touches that package: the package whose directory is the
#     longest prefix of the path.
#   - A path in no package's directory (spec/, docs/, a file at the root)
#     touches every package whose Go files name its top-level entry as a path
#     literal ("spec/...", "../../spec", "NOTICE"), because those are the
#     tests that read it as data.
#   - A package is tested when it, or any package in its test binary's
#     closure (`go list -test`'s Deps), is touched.
#   - Any path inside a package's directory, and go.mod or go.sum, also
#     has the blind packages tested: their tests build the binary or list
#     the module, which no import edge records. Their importers are not
#     tested for it, since the blind package's own code did not change.
#
# Unless given --paths, it first runs merge-setup.sh, on stderr, so a clone
# that is about to rebase has the generated files' merge drivers.
#
# Exit 0 with the facts on stdout, 2 when the change cannot be measured.

set -u

root=$(git rev-parse --show-toplevel 2>/dev/null) || {
	echo "gate-select: not inside a git checkout" >&2
	exit 2
}
cd "$root" || exit 2

# The shared seams. A change to one of these runs the wave tier, the full
# list, because what it changes is not confined to the packages that import
# it (ADR-010 Amendment A5 item 4 says why for each).
seams='^(go\.mod|go\.sum|internal/fault/|internal/testenv/|\.claude/skills/verification-gate/)'

base=""
paths_file=""
while [ $# -gt 0 ]; do
	case "$1" in
	--base)
		[ -n "${2:-}" ] || { echo "gate-select: --base needs a ref" >&2; exit 2; }
		base=$2
		shift 2
		;;
	--paths)
		[ -n "${2:-}" ] || { echo "gate-select: --paths needs a file" >&2; exit 2; }
		paths_file=$2
		shift 2
		;;
	*)
		echo "gate-select: unknown argument $1" >&2
		exit 2
		;;
	esac
done
[ -n "$paths_file" ] || "$root/.claude/skills/verification-gate/merge-setup.sh" >&2

work=$(mktemp -d "${TMPDIR:-/tmp}/gate-select.XXXXXX") || exit 2
trap 'rm -rf "$work"' EXIT

if [ -n "$paths_file" ]; then
	grep -v '^$' "$paths_file" | sort -u >"$work/changed"
else
	if [ -z "$base" ]; then
		base=$(git merge-base HEAD origin/main 2>/dev/null) || {
			echo "gate-select: no merge base with origin/main; fetch it, or pass --base" >&2
			exit 2
		}
	fi
	base=$(git rev-parse --verify --quiet "$base^{commit}") || {
		echo "gate-select: --base names no commit" >&2
		exit 2
	}
	echo "base $base"
	{
		git diff --name-only --no-renames "$base" -- &&
			git ls-files --others --exclude-standard
	} | grep -v '^$' | sort -u >"$work/changed" || {
		echo "gate-select: git could not list the change" >&2
		exit 2
	}
fi
sed 's/^/changed /' "$work/changed"
grep -E "$seams" "$work/changed" | sed 's/^/seam /'

# The module's packages: import path, directory relative to the root, and
# whether it has tests.
go list -f '{{.ImportPath}}	{{.Dir}}	{{len .TestGoFiles}}{{len .XTestGoFiles}}' ./... >"$work/pkgs" || exit 2
awk -F'\t' -v r="$root/" '{d = $2; sub("^" r, "", d); if ($2 == substr(r, 1, length(r) - 1)) d = "."; print $1 "\t" d "\t" ($3 != "00")}' \
	"$work/pkgs" >"$work/dirs"

# Blind packages: a test file that runs the go command.
: >"$work/blind"
while IFS=$'\t' read -r pkg dir _; do
	if grep -lE 'exec\.Command(Context)?\([^"]*"go"' "$dir"/*_test.go >/dev/null 2>&1; then
		echo "$pkg" >>"$work/blind"
	fi
done <"$work/dirs"

# Touched packages, with the path that touched each.
: >"$work/touched"
: >"$work/code"
: >"$work/blindhit"
while IFS= read -r p; do
	best=""
	bestdir=""
	while IFS=$'\t' read -r pkg dir _; do
		case "$p" in
		"$dir"/*)
			if [ ${#dir} -gt ${#bestdir} ]; then
				best=$pkg
				bestdir=$dir
			fi
			;;
		esac
	done <"$work/dirs"
	if [ -n "$best" ]; then
		printf '%s\t%s\n' "$best" "$p" >>"$work/touched"
		echo "code $p" >>"$work/code"
		sed "s|\$|	$p (the module's code; its tests are blind to it)|" "$work/blind" >>"$work/blindhit"
		continue
	fi
	case "$p" in
	go.mod | go.sum)
		echo "code $p" >>"$work/code"
		sed "s|\$|	$p|" "$work/blind" >>"$work/blindhit"
		cut -f1 "$work/dirs" | sed "s|\$|	$p|" >>"$work/touched"
		continue
		;;
	esac
	# Data outside every package: its readers name its top-level entry.
	top=${p%%/*}
	q=$(printf '%s' "$top" | sed 's/[.[\*^$+?(){}|]/\\&/g')
	while IFS=$'\t' read -r pkg dir _; do
		if grep -lE "\"(\\.\\./)*$q([\"/]|\$)" "$dir"/*_test.go >/dev/null 2>&1; then
			printf '%s\t%s (named as data)\n' "$pkg" "$p" >>"$work/touched"
		fi
	done <"$work/dirs"
done <"$work/changed"
cat "$work/code"
sort -u -t$'\t' -k1,1 "$work/touched" "$work/blindhit" | sed 's/^/touched /; s/\t/ /'
cut -f1 "$work/touched" | sort -u >"$work/touched.set"

# Tested packages: those with tests whose test closure meets a touched one.
go list -test -f '{{.ImportPath}} {{join .Deps " "}}' ./... |
	awk -v suffix=".test" '
		FNR == NR { touched[$1] = 1; next }
		substr($1, length($1) - length(suffix) + 1) != suffix { next }
		{
			pkg = substr($1, 1, length($1) - length(suffix))
			hit = (pkg in touched)
			for (i = 2; i <= NF && !hit; i++) {
				d = $i
				sub(/ .*/, "", d)
				if (d in touched) hit = 1
			}
			if (hit) print pkg
		}' "$work/touched.set" - |
	cat - "$work/blindhit" | cut -f1 | sort -u >"$work/test" || exit 2
# A test binary's Deps name test variants as "pkg [x.test]"; the field split
# above sees "pkg" and "[x.test]" as two words, which is what is wanted.

while IFS= read -r pkg; do
	if grep -qxF "$pkg" "$work/blind"; then
		echo "blind $pkg"
	else
		echo "test $pkg"
	fi
done <"$work/test"
