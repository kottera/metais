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
#   test <pkg>          a package to test: touched, tested alone, or
#                       importing a touched one
#   blind <pkg>         a package to test whose tests build or list the module
#                       with the go command, so no import edge shows what
#                       they read, and the test cache cannot see it
#
# A path reaches packages by these rules. A package "touched" has its
# importers tested too; one "tested alone" does not, since none of its code
# changed.
#
#   - A non-test file inside a package's directory (its Go files, embedded
#     files, any other file outside testdata/) touches that package: the
#     package whose directory is the longest prefix of the path.
#   - A test file (_test.go), or a file under testdata/ the build does not
#     embed, has its own package tested alone: no other test binary
#     compiles a test file, and another package's test that reads the
#     testdata names it (the data rule).
#   - Any path is data to the tests that name it: a package is tested alone
#     when its test files name the path, or a directory above it, as a
#     string literal or as a run of literal arguments to a Join, read from
#     the package's directory and from the root
#     (gate-select-named.awk says how). "docs/guide/index.html" names that
#     path, not every file under docs/. A run that climbs out of a computed
#     base (Join(suite, "..", "README.md")) names any path that ends in it.
#     Inside a package's directory, a directory above the package's own
#     does not count; the walkers below are how the module is read whole.
#   - A Go file in no package's directory (build-ignored, under a directory
#     go list does not see) has tested alone every package whose test files
#     name the module root itself as a path ("../..", filepath.Join("..",
#     "..") from internal/x): those tests walk the module's tree, and its Go
#     files whole (licence headers, build directives), whatever go list
#     lists.
#   - Any path inside a package's directory has the module walkers tested
#     alone: the root readers whose test files also name Go files by a
#     pattern or suffix ("*.go", "_test.go"). They read every package's
#     files, test files and testdata included.
#   - A package is tested when it, or any package in its test binary's
#     closure (`go list -test`'s Deps), is touched.
#   - The module's build inputs, and go.mod and go.sum, have the blind
#     packages tested alone: their tests build the binary or list the
#     module, which no import edge records. A build input is a non-test
#     file in a package's directory, as above: what go build and go list
#     read. A test file or testdata in another package is not one.
#
# Unless given --paths, it first runs merge-setup.sh, on stderr, so a clone
# that is about to rebase has the generated files' merge drivers.
#
# Exit 0 with the facts on stdout, 2 when the change cannot be measured.

set -u

here=$(cd "$(dirname "$0")" && pwd) || exit 2
# shellcheck source=git-env.sh
. "$here/git-env.sh" || exit 2
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
# whether it has tests. Beside them, the files the build embeds, relative to
# the root.
go list -f '{{.ImportPath}}	{{.Dir}}	{{len .TestGoFiles}}{{len .XTestGoFiles}}{{range .EmbedFiles}}	{{.}}{{end}}' ./... >"$work/pkgs" || exit 2
awk -F'\t' -v r="$root/" '{d = $2; sub("^" r, "", d); if ($2 == substr(r, 1, length(r) - 1)) d = "."; print $1 "\t" d "\t" ($3 != "00")}' \
	"$work/pkgs" >"$work/dirs"
awk -F'\t' -v r="$root/" '{d = $2; sub("^" r, "", d); if ($2 == substr(r, 1, length(r) - 1)) d = "."; for (i = 4; i <= NF; i++) print (d == "." ? "" : d "/") $i}' \
	"$work/pkgs" >"$work/embeds"

# Blind packages: a test file that runs the go command, which it gives
# testenv.GoEnv's environment, or names directly.
: >"$work/blind"
while IFS=$'\t' read -r pkg dir _; do
	if grep -lE 'testenv\.GoEnv\(|exec\.Command(Context)?\([^"]*"go"' "$dir"/*_test.go >/dev/null 2>&1; then
		echo "$pkg" >>"$work/blind"
	fi
done <"$work/dirs"

# Root readers: a test file that names the module root itself, by as many
# ".." as its package's directory is deep, as a literal or a Join.
: >"$work/rootreaders"
while IFS=$'\t' read -r pkg dir _; do
	[ "$dir" = . ] && continue
	ups=$(printf '%s\n' "$dir" | awk -F/ '{ s = ""; for (i = 1; i < NF; i++) s = s "\\.\\./"; print s }')
	joins=$(printf '%s\n' "$dir" | awk -F/ '{ s = ""; for (i = 1; i < NF; i++) s = s "\"\\.\\.\", *"; print s }')
	if grep -lE "\"$ups\\.\\./?\"|Join\($joins\"\\.\\.\"\)" "$dir"/*_test.go >/dev/null 2>&1; then
		echo "$pkg" >>"$work/rootreaders"
	fi
done <"$work/dirs"

# Module walkers: root readers whose test files name Go files by a pattern
# or suffix ("*.go", "_test.go"). They read the module's files whole, test
# files and testdata included, wherever they are.
: >"$work/walkers"
while IFS= read -r pkg; do
	dir=$(awk -F'\t' -v p="$pkg" '$1 == p { print $2 }' "$work/dirs")
	if grep -lE '\.go"' "$dir"/*_test.go >/dev/null 2>&1; then
		echo "$pkg" >>"$work/walkers"
	fi
done <"$work/rootreaders"

# Named paths: "<pkg> <kind> <path>" for every path a package's test files
# name, by a string literal or a run of literal arguments to a Join, as
# gate-select-named.awk reads them. filepath.Join(root, "docs", "adr") names
# docs/adr, not every file under docs/.
: >"$work/testfiles"
while IFS=$'\t' read -r _ dir _; do
	for f in "$dir"/*_test.go; do
		[ -e "$f" ] && echo "$f" >>"$work/testfiles"
	done
done <"$work/dirs"
tr '\n' '\0' <"$work/testfiles" | LC_ALL=C xargs -0 awk -f "$here/gate-select-named.awk" |
	awk -F'\t' 'NR == FNR { pkg[$2] = $1; next } { print pkg[$1] "\t" $2 "\t" $3 }' "$work/dirs" - |
	sort -u >"$work/named"

# Touched packages, with the path that touched each. "touched" packages have
# their importers tested too; "direct" ones are tested alone.
: >"$work/touched"
: >"$work/code"
: >"$work/direct"
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
	# Data: every package whose tests name the path, or a directory above
	# it. In a package's directory, a directory above the package's own is
	# not enough: naming internal/ is how a test walks the module, and the
	# walkers are reached below.
	awk -F'\t' -v p="$p" -v d="$bestdir" '
		d != "" && ($2 == "E" || $2 == "P") && $3 != d && index($3, d "/") != 1 { next }
		($2 == "E" && (p == $3 || index(p, $3 "/") == 1)) ||
		($2 == "P" && index(p, $3) == 1) || ($2 == "F" && p == $3) ||
		($2 == "S" && index("/" p "/", "/" $3 "/") > 0) {
			print $1 "\t" p " (named as data)"
		}' "$work/named" | sort -u >>"$work/direct"
	if [ -n "$best" ]; then
		echo "code $p" >>"$work/code"
		sed "s|\$|	$p (in a package; its tests walk the module)|" "$work/walkers" >>"$work/direct"
		# A test file, or testdata the build does not embed, is read by its
		# own package's tests alone: no other test binary compiles a test
		# file, and another package's test that reads the testdata names it.
		case "$p" in
		*_test.go)
			printf '%s\t%s (a test file)\n' "$best" "$p" >>"$work/direct"
			continue
			;;
		esac
		case "/${p#"$bestdir"/}" in
		*/testdata/*)
			if ! grep -qxF "$p" "$work/embeds"; then
				printf '%s\t%s (testdata)\n' "$best" "$p" >>"$work/direct"
				continue
			fi
			;;
		esac
		printf '%s\t%s\n' "$best" "$p" >>"$work/touched"
		sed "s|\$|	$p (the module's code; its tests are blind to it)|" "$work/blind" >>"$work/direct"
		continue
	fi
	case "$p" in
	go.mod | go.sum)
		echo "code $p" >>"$work/code"
		sed "s|\$|	$p|" "$work/blind" >>"$work/direct"
		cut -f1 "$work/dirs" | sed "s|\$|	$p|" >>"$work/touched"
		continue
		;;
	esac
	# A Go file outside every package: the module's tree walkers read it.
	case "$p" in
	*.go) sed "s|\$|	$p (a Go file in no package; its tests walk the module)|" "$work/rootreaders" >>"$work/direct" ;;
	esac
done <"$work/changed"
cat "$work/code"
sort -u -t$'\t' -k1,1 "$work/touched" "$work/direct" | sed 's/^/touched /; s/\t/ /'
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
	cat - "$work/direct" | cut -f1 | sort -u >"$work/test" || exit 2
# A test binary's Deps name test variants as "pkg [x.test]"; the field split
# above sees "pkg" and "[x.test]" as two words, which is what is wanted.

while IFS= read -r pkg; do
	if grep -qxF "$pkg" "$work/blind"; then
		echo "blind $pkg"
	else
		echo "test $pkg"
	fi
done <"$work/test"
