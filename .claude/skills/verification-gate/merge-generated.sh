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
whole | region) ;;
*)
	echo "merge-generated: unknown mode '$mode' (whole or region)" >&2
	exit 2
	;;
esac
[ -n "$path" ] || {
	echo "merge-generated: usage: merge-generated.sh whole|region %O %A %B %P [%L]" >&2
	exit 2
}

if [ "$mode" = whole ]; then
	echo "merge-generated: $path is generated; kept one side's copy. Run .claude/skills/verification-gate/regenerate.sh before you commit." >&2
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
	git merge-file --marker-size="$size" -L ours -L base -L theirs "$ours" "$base" "$theirs"
	[ $? -eq 0 ] || exit 1
	exit 0
}

n=$(regions "$ours")
[ "$n" != bad ] && [ "$n" != 0 ] || whole_text "this side's generated-region markers are not well formed"
[ "$(regions "$base")" = "$n" ] && [ "$(regions "$theirs")" = "$n" ] ||
	whole_text "the three versions do not hold the same generated regions"

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
