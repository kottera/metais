#!/bin/sh
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# coauthor.sh <base> <head> [<message>] is CI's co-author check: no commit
# of the change, and not the pull request's title and body in the file
# <message>, carries a Co-authored-by line, which AGENTS.md forbids (the
# ruling at PLAN-2026-10-metais-v1's T-A4). A line opening with the key
# fails the change in any case and wherever it sits, since GitHub's squash
# merge carries a commit's body into the squash message; the key inside a
# line of prose does not. The change is the DCO check's: every commit
# reachable from <head> and not from <base>, merge commits included, or
# <head> itself when it is already on <base>.
#
# Exit 0 when nothing carries the line, 1 when anything does, 2 when the
# change or the message cannot be read.
set -u

if [ "$#" -ne 2 ] && [ "$#" -ne 3 ]; then
	echo "usage: coauthor.sh <base> <head> [<message>]" >&2
	exit 2
fi
base=$1
head=$2

# coauthored reads a message on its standard input and succeeds when a
# line of it opens with the key.
coauthored() {
	grep -Eiq '^[[:space:]]*co-authored-by[[:space:]]*:'
}

commits=$(git rev-list "$base..$head") || {
	echo "coauthor: cannot list the commits of $base..$head" >&2
	exit 2
}
if [ -z "$commits" ]; then
	commits=$(git rev-parse --verify "$head^{commit}") || exit 2
fi

n=0
bad=0
for c in $commits; do
	n=$((n + 1))
	msg=$(git log -1 --format=%B "$c") || exit 2
	if printf '%s\n' "$msg" | coauthored; then
		bad=$((bad + 1))
		echo "coauthor: $(git log -1 --format='%h %s' "$c") carries a Co-authored-by line"
	fi
done

what="$n commit(s)"
if [ "$#" -eq 3 ]; then
	what="$what and the pull request's title and body"
	if [ ! -r "$3" ]; then
		echo "coauthor: cannot read the pull request's title and body from $3" >&2
		exit 2
	fi
	if coauthored <"$3"; then
		bad=$((bad + 1))
		echo "coauthor: the pull request's title and body carry a Co-authored-by line"
	fi
fi

if [ "$bad" -gt 0 ]; then
	echo "coauthor: $bad of $what carry a Co-authored-by line. Remove each (git rebase -i $base, or edit the pull request's body), and merge with an explicit squash body, since GitHub's default one adds the line."
	exit 1
fi
echo "coauthor: no Co-authored-by line in $what"
