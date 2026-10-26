#!/bin/sh
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# squash-body.sh <file> checks a squash merge's body before the merge
# (AGENTS.md, Git and pull requests): the file passed to gh pr merge
# --body-file. CI's DCO and co-author checks read a pull request's own
# commits and its title and body, so neither can see the body a merger
# types at the merge, and a body that loses the trailer lands on main
# unsigned. The body must not be empty; its last line that is not blank
# must be a Signed-off-by: line naming a person and an email address, with
# nothing after it; git must parse that line as a trailer of the merged
# commit's message, the subject, a blank line and the body, as dco.sh
# reads it; and no line may open with Co-authored-by.
#
# Exit 0 when the body is fit to merge with, 1 when it is not, 2 when it
# cannot be read.
set -u

if [ "$#" -ne 1 ]; then
	echo "usage: squash-body.sh <file>" >&2
	exit 2
fi
file=$1

if [ ! -f "$file" ] || [ ! -r "$file" ]; then
	echo "squash-body: cannot read $file" >&2
	exit 2
fi

# signoff is dco.sh's form of a sign-off's value: a name and an address.
signoff='[^<>]*[^<> ][^<>]* <[^<>@ ]+@[^<>@ ]+>'

if ! grep -q '[^[:space:]]' "$file"; then
	echo "squash-body: $file is empty, so the merged commit would carry no Signed-off-by"
	exit 1
fi

bad=0
last=$(grep -v '^[[:space:]]*$' "$file" | tail -n 1)
if ! printf '%s\n' "$last" | grep -Eq "^Signed-off-by: $signoff\$"; then
	bad=1
	echo "squash-body: $file does not end with a Signed-off-by: Name <email> line; its last line is: $last"
fi
trailers=$(printf 'squash\n\n%s\n' "$(cat "$file")" | git interpret-trailers --parse) || exit 2
if ! printf '%s\n' "$trailers" | grep -Eq "^Signed-off-by: $signoff\$"; then
	bad=1
	echo "squash-body: git parses no Signed-off-by: Name <email> trailer from $file; put the line in the body's last paragraph"
fi
if grep -Eiq '^[[:space:]]*co-authored-by[[:space:]]*:' "$file"; then
	bad=1
	echo "squash-body: $file carries a Co-authored-by line"
fi

if [ "$bad" -ne 0 ]; then
	exit 1
fi
echo "squash-body: $file ends with its Signed-off-by trailer"
