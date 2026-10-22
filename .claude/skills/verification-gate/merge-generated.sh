#!/bin/sh
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# merge-generated.sh is the merge driver for the committed generated files
# (ADR-011 D3 item 7 as amended by A1). .gitattributes names the driver for
# each path, and merge-setup.sh installs it in a clone's configuration. Git
# runs it from the top of the work tree as:
#
#   merge-generated.sh whole  %O %A %B %P
#   merge-generated.sh region %O %A %B %P %L
#   merge-generated.sh values %O %A %B %P %L
#
# whole   The file is generated whole. Keep this side's copy (%A) and report
#         a clean merge: regenerate.sh rewrites it from the merged sources,
#         and its drift test fails until it does.
# region  The file is hand-written around generated regions. A region opens
#         on a line `<!-- ... generated ... Do not edit it by hand. -->` and
#         closes on a line `<!-- The end of the generated ... -->`. The text
#         outside the regions gets git's own three-way merge, conflicts
#         included; inside each region, this side's lines are kept. When the
#         markers are not one well-formed sequence of regions in all three
#         versions, or the merge disturbs them, the whole file gets git's
#         three-way merge instead, so hand-written text is never one side's.
# values  The file is hand-written and holds numbers a generator writes in
#         it: a conformance vector's offsets and sizes, which vectorgen's
#         TestValues writes. It gets git's three-way merge. A conflict of
#         as many lines on each side is then merged line by line: a line
#         keeps the side that changed it, and a line both sides changed
#         only in its digits is two sides regenerating one number, so it
#         keeps this side's, which regenerating rewrites. Every other
#         conflict is left to a person, so hand-written text is never one
#         side's.
#
# Every merge it resolves prints a line naming the path and saying that it
# needs regenerating, so a clean rebase is never mistaken for a finished one.

set -u

mode=${1:-}
base=${2:-}
ours=${3:-}
theirs=${4:-}
path=${5:-}
size=${6:-7}

case "$mode" in
whole | region | values) ;;
*)
	echo "merge-generated: unknown mode '$mode' (whole, region or values)" >&2
	exit 2
	;;
esac
[ -n "$path" ] || {
	echo "merge-generated: usage: merge-generated.sh whole|region|values %O %A %B %P [%L]" >&2
	exit 2
}

if [ "$mode" = whole ]; then
	echo "merge-generated: $path is generated; kept one side's copy. Run .claude/skills/verification-gate/regenerate.sh before you commit." >&2
	exit 0
fi

if [ "$mode" = values ]; then
	work=$(mktemp -d "${TMPDIR:-/tmp}/merge-generated.XXXXXX") || exit 2
	trap 'rm -rf "$work"' EXIT
	git merge-file -p --diff3 --marker-size="$size" -L ours -L base -L theirs "$ours" "$base" "$theirs" >"$work/merged"
	rc=$?
	[ "$rc" -ge 0 ] && [ "$rc" -lt 128 ] || exit 2
	# Each conflict that differs only in digits is resolved line by line;
	# any other is printed as git left it, and the exit status says so.
	awk -v n="$size" '
		function rep(c, k, s) { s = ""; while (k-- > 0) s = s c; return s }
		function digitless(s) { gsub(/[0-9]+/, "0", s); return s }
		# line is line i of the conflict merged alone: the side that
		# changed it, or this side when both changed only its digits.
		function line(i) {
			if (o[i] == b[i]) return t[i]
			if (t[i] == b[i] || digitless(o[i]) == digitless(b[i]) && digitless(t[i]) == digitless(b[i])) return o[i]
			return SUBSEP
		}
		function resolve(i, ok) {
			ok = no == nb && nb == nt
			for (i = 1; ok && i <= no; i++) ok = line(i) != SUBSEP
			if (ok) {
				for (i = 1; i <= no; i++) print line(i)
				return
			}
			left = 1
			print lt; for (i = 1; i <= no; i++) print o[i]
			print bar; for (i = 1; i <= nb; i++) print b[i]
			print eq; for (i = 1; i <= nt; i++) print t[i]
			print gt
		}
		BEGIN { lt = rep("<", n) " ours"; bar = rep("|", n) " base"; eq = rep("=", n); gt = rep(">", n) " theirs" }
		state == 0 && $0 == lt { state = 1; no = nb = nt = 0; next }
		state == 1 && $0 == bar { state = 2; next }
		state == 2 && $0 == eq { state = 3; next }
		state == 3 && $0 == gt { resolve(); state = 0; next }
		state == 1 { o[++no] = $0; next }
		state == 2 { b[++nb] = $0; next }
		state == 3 { t[++nt] = $0; next }
		{ print }
		END { exit left }
	' "$work/merged" >"$work/result"
	left=$?
	[ "$left" -le 1 ] || exit 2
	cat "$work/result" >"$ours" || exit 2
	if [ "$left" -ne 0 ]; then
		echo "merge-generated: $path: the hand-written text conflicts; resolve it, then run .claude/skills/verification-gate/regenerate.sh." >&2
		exit 1
	fi
	echo "merge-generated: $path: merged the hand-written text; its generated numbers may be either side's. Run .claude/skills/verification-gate/regenerate.sh before you commit." >&2
	exit 0
fi

begin='^<!-- .*generated.*Do not edit it by hand[.] -->$'
end='^<!-- The end of the generated .*-->$'

# regions prints how many regions a file holds, or "bad" when its markers
# do not open and close in turn.
regions() {
	awk -v b="$begin" -v e="$end" '
		$0 ~ b { if (open) bad = 1; open = 1; n++; next }
		$0 ~ e { if (!open) bad = 1; open = 0; next }
		END { if (bad || open) print "bad"; else print n + 0 }
	' "$1"
}

# strip prints a file with every region's lines removed, its markers kept.
strip() {
	awk -v b="$begin" -v e="$end" '
		$0 ~ b { print; open = 1; next }
		$0 ~ e { print; open = 0; next }
		!open { print }
	' "$1"
}

# whole_text gives the whole file git's three-way merge, and its verdict.
whole_text() {
	echo "merge-generated: $path: $1; merging the whole file as text." >&2
	git merge-file --marker-size="$size" -L ours -L base -L theirs "$ours" "$base" "$theirs" || exit 1
	exit 0
}

n=$(regions "$ours")
if [ "$n" = bad ] || [ "$n" = 0 ]; then
	whole_text "this side's generated-region markers are not well formed"
fi
if [ "$(regions "$base")" != "$n" ] || [ "$(regions "$theirs")" != "$n" ]; then
	whole_text "the three versions do not hold the same generated regions"
fi

work=$(mktemp -d "${TMPDIR:-/tmp}/merge-generated.XXXXXX") || exit 2
trap 'rm -rf "$work"' EXIT
strip "$base" >"$work/base" &&
	strip "$ours" >"$work/ours" &&
	strip "$theirs" >"$work/theirs" || exit 2
# Each region's lines on this side, one file per region.
awk -v b="$begin" -v e="$end" -v dir="$work" '
	$0 ~ b { open = 1; n++; printf "" > (dir "/region." n); next }
	$0 ~ e { open = 0; next }
	open { print > (dir "/region." n) }
' "$ours" || exit 2

git merge-file --marker-size="$size" -L ours -L base -L theirs "$work/ours" "$work/base" "$work/theirs"
rc=$?
[ "$rc" -ge 0 ] && [ "$rc" -lt 128 ] || exit 2
[ "$(regions "$work/ours")" = "$n" ] || whole_text "the merge of the text around the generated regions disturbed their markers"

awk -v b="$begin" -v dir="$work" '
	{ print }
	$0 ~ b { n++; f = dir "/region." n; while ((getline l < f) > 0) print l; close(f) }
' "$work/ours" >"$work/result" || exit 2
cat "$work/result" >"$ours" || exit 2

if [ "$rc" -ne 0 ]; then
	echo "merge-generated: $path: the hand-written text conflicts; resolve it, then run .claude/skills/verification-gate/regenerate.sh." >&2
	exit 1
fi
echo "merge-generated: $path: merged the hand-written text and kept one side's generated region. Run .claude/skills/verification-gate/regenerate.sh before you commit." >&2
exit 0
