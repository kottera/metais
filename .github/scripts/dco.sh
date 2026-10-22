#!/bin/sh
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# dco.sh <base> <head> is CI's DCO check (ADR-013 D3 item 2): every commit
# of the change carries a Signed-off-by: trailer naming a person and an
# email address, certifying the Developer Certificate of Origin
# (https://developercertificate.org/). The change is every commit
# reachable from <head> and not from <base>, merge commits included; when
# there is none, <head> is already on <base> and the change is <head>
# itself. There is no exemption list.
#
# Exit 0 when every commit is signed off, 1 when any is not, 2 when the
# change cannot be read.
set -u

if [ "$#" -ne 2 ]; then
	echo "usage: dco.sh <base> <head>" >&2
	exit 2
fi
base=$1
head=$2

commits=$(git rev-list "$base..$head") || {
	echo "dco: cannot list the commits of $base..$head" >&2
	exit 2
}
if [ -z "$commits" ]; then
	commits=$(git rev-parse --verify "$head^{commit}") || exit 2
fi

n=0
bad=0
for c in $commits; do
	n=$((n + 1))
	signoffs=$(git log -1 --format='%(trailers:key=Signed-off-by,valueonly,unfold)' "$c") || exit 2
	if printf '%s\n' "$signoffs" | grep -Eq '^[^<>]*[^<> ][^<>]* <[^<>@ ]+@[^<>@ ]+>$'; then
		continue
	fi
	bad=$((bad + 1))
	echo "dco: $(git log -1 --format='%h %s' "$c") carries no Signed-off-by: Name <email>"
done

if [ "$bad" -gt 0 ]; then
	echo "dco: $bad of $n commit(s) are not signed off. Sign each one off (git commit -s, or git rebase --signoff $base) to certify the Developer Certificate of Origin."
	exit 1
fi
echo "dco: all $n commit(s) are signed off"
