#!/bin/sh
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# build.sh <out> writes the metais.org site into <out>, which must not exist:
# the pages and CNAME of site/public/, every JSON Schema of spec/schemas/
# under schemas/, and every CLI output schema of docs/cli/schemas/ under
# cli/schemas/, so that each one is served at the path its $id names
# (https://metais.org/schemas/<path> is spec/schemas/<path>, and
# https://metais.org/cli/schemas/<path> is docs/cli/schemas/<path>). The
# schemas are copied from the commit being built, never committed twice;
# site_test.go holds the $id-to-path mapping and runs this script.
set -eu

if [ "$#" -ne 1 ]; then
	echo "usage: site/build.sh <out>" >&2
	exit 2
fi
out=$1
if [ -e "$out" ] || [ -L "$out" ]; then
	echo "site/build.sh: $out exists; the site is built into a new directory" >&2
	exit 1
fi
root=$(cd "$(dirname "$0")/.." && pwd)

if [ -e "$root/site/public/schemas" ]; then
	echo "site/build.sh: site/public/schemas exists; schemas/ is spec/schemas/ alone" >&2
	exit 1
fi
if [ -e "$root/site/public/cli" ]; then
	echo "site/build.sh: site/public/cli exists; cli/ is docs/cli/schemas/ alone" >&2
	exit 1
fi

mkdir -p "$out"
cp -R "$root/site/public/." "$out/"
mkdir "$out/schemas"
cp -R "$root/spec/schemas/." "$out/schemas/"
mkdir -p "$out/cli/schemas"
cp -R "$root/docs/cli/schemas/." "$out/cli/schemas/"
