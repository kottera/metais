#!/bin/sh
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# merge-setup.sh installs the generated files' merge drivers in this clone's
# configuration (.git/config), from the repository root:
#
#   merge-setup.sh          install them, and say whether they were missing
#   merge-setup.sh --check  install nothing; exit 1 when one is missing
#
# .gitattributes names a driver for each committed generated file, and git
# reads drivers only from configuration, never from the repository. A clone
# without them merges those files as text, without a word. So gate.sh,
# gate-select.sh, gate-why.sh and regenerate.sh all run this first, and a
# fresh clone runs it once before its first rebase. Every merge=metais-* name
# in .gitattributes must be one this script installs, or it fails.
#
# Exit 0 when every driver is installed, 1 when one is missing (--check) or
# unknown, 2 when it cannot run.

set -u

root=$(git rev-parse --show-toplevel 2>/dev/null) || {
	echo "merge-setup: not inside a git checkout" >&2
	exit 2
}
cd "$root" || exit 2
check=""
case "${1:-}" in
"") ;;
--check) check=1 ;;
*)
	echo "merge-setup: unknown argument $1 (usage: merge-setup.sh [--check])" >&2
	exit 2
	;;
esac

script=.claude/skills/verification-gate/merge-generated.sh

# driver prints the command git runs for one of the drivers.
driver() {
	case "$1" in
	metais-generated) echo "sh $script whole %O %A %B %P" ;;
	metais-generated-region) echo "sh $script region %O %A %B %P %L" ;;
	esac
}

label() {
	case "$1" in
	metais-generated) echo "metais: a generated file; keep one side, then regenerate" ;;
	metais-generated-region) echo "metais: generated regions in hand-written text; merge the text, keep one side of each region, then regenerate" ;;
	esac
}

names=$(grep -o 'merge=metais-[A-Za-z0-9-]*' .gitattributes 2>/dev/null | cut -d= -f2 | sort -u)
[ -n "$names" ] || {
	echo "merge-setup: .gitattributes names no merge=metais-* driver" >&2
	exit 1
}

rc=0
missing=""
for n in $names; do
	want=$(driver "$n")
	if [ -z "$want" ]; then
		echo "merge-setup: .gitattributes names the driver $n, which this script does not know" >&2
		rc=1
		continue
	fi
	[ "$(git config --get "merge.$n.driver")" = "$want" ] && continue
	missing="$missing $n"
	[ -n "$check" ] && continue
	git config "merge.$n.name" "$(label "$n")" &&
		git config "merge.$n.driver" "$want" || exit 2
done

if [ -z "$missing" ]; then
	[ "$rc" -eq 0 ] && echo "merge-setup: the generated files' merge drivers are installed"
	exit $rc
fi
if [ -n "$check" ]; then
	echo "merge-setup: this clone lacks the merge driver(s)$missing; run .claude/skills/verification-gate/merge-setup.sh" >&2
	exit 1
fi
echo "merge-setup: NOTE: installed the merge driver(s)$missing, which this clone lacked. A rebase or merge before now merged generated files as text: run .claude/skills/verification-gate/regenerate.sh."
exit $rc
