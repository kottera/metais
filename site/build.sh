#!/bin/sh
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# build.sh [--release <version> <SHA256SUMS>] <out> writes the metais.org
# site into <out>, which must not exist, from the commit checked out here
# (ADR-013 D8). It is the one command that builds the site, and it runs
# internal/tools/site, which:
#
#   1. builds metais from this checkout;
#   2. exports the site package's inputs as JSON, from that metais (its
#      help --json, skill --print and version --json), from spec/ and the
#      spec-<M>.<m> tags, and from the release named below;
#   3. renders the package at site/package with that metais, with no hook,
#      script or network;
#   4. adds every file of site/public at its own path, byte for byte, and
#      every JSON Schema of spec/schemas/ under schemas/ and of
#      docs/cli/schemas/ under cli/schemas/, so each is served at the path
#      its $id names; a file whose path a page already has is refused;
#   5. moves the whole site into <out> in one rename.
#
# The release reaches it as arguments: --release <version> <SHA256SUMS>, the
# release's semantic version without its v, and the SHA256SUMS file the
# release builder wrote into dist/ and the GitHub release publishes beside
# its archives. It is refused unless HEAD is the tag v<version>, and unless
# the checkout holds the whole history, tags included (actions/checkout with
# fetch-depth: 0), because the specification's released revisions are read
# from their spec-<M>.<m> tags: a shallow clone is refused. metais is then
# built as the release is, and must report that version, and the release
# input lists each archive with its checksum and its download URL on the
# release.
#
# With no --release, as CI and a local build run it, metais is built with
# the version Go stamps from the commit, and the release input says no
# release is published: the pages are those of a build with no release. The
# specification's released revisions are the spec-<M>.<m> tags the
# checkout holds, which in a shallow clone may be none.
#
# The same commit and arguments give the same site, byte for byte;
# site_test.go builds it twice to hold that, and holds what it serves.
set -eu

usage() {
	echo "usage: site/build.sh [--release <version> <SHA256SUMS>] <out>" >&2
	exit 2
}

release=""
sums=""
if [ "$#" -ge 1 ] && [ "$1" = "--release" ]; then
	[ "$#" -eq 4 ] || usage
	release=$2
	sums=$3
	shift 3
fi
[ "$#" -eq 1 ] || usage
out=$1
case $out in
"") usage ;;
/*) ;;
*) out=$(pwd)/$out ;;
esac
if [ -n "$sums" ]; then
	case $sums in
	/*) ;;
	*) sums=$(pwd)/$sums ;;
	esac
fi
if [ -e "$out" ] || [ -L "$out" ]; then
	echo "site/build.sh: $out exists; the site is built into a new directory" >&2
	exit 1
fi
root=$(cd "$(dirname "$0")/.." && pwd)

cd "$root"
if [ -n "$release" ]; then
	exec go run ./internal/tools/site -root "$root" -out "$out" -release "$release" -sums "$sums"
fi
exec go run ./internal/tools/site -root "$root" -out "$out"
