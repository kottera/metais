#!/bin/sh
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# rehearse.sh runs one of the rehearsal's checks against a published release
# (ADR-013 D7 item 8, docs/dev/release.md "The rehearsal's checks"). Each
# step of .github/workflows/rehearse.yml runs one check, on a clean
# GitHub-hosted runner or in a container with nothing installed:
#
#   rehearse.sh install <version>        curl -fsSL https://metais.org/install.sh | METAIS_VERSION=<version> sh
#   rehearse.sh latest <version>         the same with no METAIS_VERSION: the refusal while
#                                        <version> is a prerelease, an install of <version> otherwise
#   rehearse.sh manual <version> <dir>   the manual route into <dir>: download this machine's
#                                        archive and SHA256SUMS, check it, unpack metais
#   rehearse.sh version <metais> <version>
#                                        metais --version reports <version>
#   rehearse.sh skill <metais> <version> <dir>
#                                        metais skill writes the skill of <version> into the new
#                                        project <dir>
#   rehearse.sh ids <tree>               every $id of the schemas in <tree> resolves as that JSON
#   rehearse.sh go-import                the module's go-import pages
#   rehearse.sh reference <metais>       a reference page for every command and topic <metais> has
#   rehearse.sh spec <tree>              the specification's draft, a page per chapter in <tree>
#   rehearse.sh pages <tree>             the landing page, the install page, and install.sh as
#                                        <tree> holds it
#   rehearse.sh kottera                  kottera.com
#
# <version> is the release's semantic version, with or without its v.
# <tree> is a checkout of the release's tag, so a schema, a command or a
# chapter added later is never skipped and one the release lacks is never
# asked for. install and latest install into ~/.local/bin, the golden
# path's directory.
#
# Every check prints each command it runs after "$ ", what it printed, and
# the version it reported, so its log is its evidence. A check that refuses
# says why on a line beginning FAIL; one that passes ends with a line
# beginning ok.
#
# METAIS_REHEARSE_SITE, METAIS_REHEARSE_DOWNLOAD and METAIS_REHEARSE_KOTTERA
# replace https://metais.org, https://github.com/kottera/metais/releases/download
# and https://kottera.com, for internal/guarantees' TestRehearseScript alone.
# A run of the rehearsal sets none of them.
#
# Exit 0 when the check passes, 1 when it does not, 2 on a usage error.
set -eu

site=${METAIS_REHEARSE_SITE:-https://metais.org}
download=${METAIS_REHEARSE_DOWNLOAD:-https://github.com/kottera/metais/releases/download}
kottera=${METAIS_REHEARSE_KOTTERA:-https://kottera.com}

usage() {
	sed -n 's/^#   rehearse.sh/usage: rehearse.sh/p' "$0" >&2
	exit 2
}

fail() {
	echo "FAIL: $*" >&2
	exit 1
}

# release <version> is <version> without its v, refused unless it is a
# version install.sh would accept.
release() {
	v=${1#v}
	case $v in
	'' | *[!0-9A-Za-z.+-]* | [!0-9]*) fail "\"$1\" is not a metais version, such as 1.0.0-rc.1" ;;
	esac
	echo "$v"
}

# prerelease <version> succeeds when <version> has a prerelease suffix.
prerelease() {
	case ${1%%+*} in
	*-*) return 0 ;;
	esac
	return 1
}

# platform is this machine's <os>_<arch>, as the archives name it.
platform() {
	case $(uname -s) in
	Darwin) os=darwin ;;
	Linux) os=linux ;;
	*) fail "the release has no archive for $(uname -s)" ;;
	esac
	case $(uname -m) in
	x86_64 | amd64) arch=amd64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*) fail "the release has no archive for $(uname -m)" ;;
	esac
	echo "${os}_$arch"
}

# show <command>... prints the command, then runs it.
show() {
	echo "\$ $*"
	"$@"
}

# check_version <metais> <version> runs <metais> --version and requires its
# first line to be "metais <version>".
check_version() {
	echo "\$ $1 --version"
	out=$("$1" --version 2>&1) || fail "$1 --version exited $?: $out"
	printf '%s\n' "$out"
	first=$(printf '%s\n' "$out" | head -n 1)
	[ "$first" = "metais $2" ] || fail "$1 --version reports \"$first\", want \"metais $2\""
	echo "ok: $1 reports $2"
}

# fingerprint <file> is the file's checksum, or absent.
fingerprint() {
	if [ -e "$1" ]; then cksum <"$1"; else echo absent; fi
}

# get <url> <out> fetches <url> into <out>, following redirects, prints
# "GET <url>: <code> <content type>", and leaves the code and the type in
# $code and $type. A failure to connect leaves code 000, and curl's own
# message above the line.
get() {
	got=$(curl -sSL -o "$2" -w '%{http_code} %{content_type}' "$1") || true
	code=${got%% *}
	type=${got#* }
	echo "GET $1: $got"
}

# page <url> requires <url> to serve HTML, and counts a refusal in $bad.
page() {
	get "$1" "$tmp/page"
	case $code:$type in
	200:text/html*) ;;
	*)
		echo "FAIL: $1 answered $code $type, not 200 text/html" >&2
		bad=$((bad + 1))
		;;
	esac
}

# finish <what> ends a check that counted its refusals in $bad.
finish() {
	[ "$bad" -eq 0 ] || fail "$bad of $n $1: refused"
	echo "ok: $n of $n $1"
}

[ "$#" -ge 1 ] || usage
check=$1
shift
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
bad=0
n=0

case $check in
install)
	[ "$#" -eq 1 ] || usage
	version=$(release "$1")
	bin=${HOME:?rehearse.sh: HOME is not set}/.local/bin/metais
	before=$(fingerprint "$bin")
	echo "\$ curl -fsSL $site/install.sh | METAIS_VERSION=$version sh"
	curl -fsSL "$site/install.sh" | METAIS_VERSION=$version sh || fail "install.sh exited $?"
	[ -x "$bin" ] || fail "install.sh did not install $bin"
	[ "$(fingerprint "$bin")" != "$before" ] || fail "install.sh left $bin as it was"
	check_version "$bin" "$version"
	;;

latest)
	[ "$#" -eq 1 ] || usage
	version=$(release "$1")
	bin=${HOME:?rehearse.sh: HOME is not set}/.local/bin/metais
	before=$(fingerprint "$bin")
	echo "\$ curl -fsSL $site/install.sh | sh"
	rc=0
	curl -fsSL "$site/install.sh" | sh >"$tmp/out" 2>&1 || rc=$?
	cat "$tmp/out"
	echo "(exit $rc)"
	if prerelease "$version"; then
		echo "$version is a prerelease, so no stable release exists and install.sh must refuse"
		[ "$rc" -ne 0 ] || fail "install.sh succeeded with no stable release"
		for want in 'no stable release' METAIS_VERSION; do
			grep -q "$want" "$tmp/out" || fail "install.sh's refusal does not say \"$want\""
		done
		[ "$(fingerprint "$bin")" = "$before" ] || fail "the refused install changed $bin"
		echo "ok: install.sh refuses, naming METAIS_VERSION, and installs nothing"
	else
		echo "$version is a release, so install.sh must install it as the latest"
		[ "$rc" -eq 0 ] || fail "install.sh exited $rc"
		check_version "$bin" "$version"
	fi
	;;

manual)
	[ "$#" -eq 2 ] || usage
	version=$(release "$1")
	[ ! -e "$2" ] || fail "$2 exists"
	archive=metais_${version}_$(platform).tar.gz
	mkdir -p "$2"
	cd "$2"
	show curl -fsSLO "$download/v$version/$archive" || fail "could not download $archive"
	show curl -fsSLO "$download/v$version/SHA256SUMS" || fail "could not download SHA256SUMS"
	show cat SHA256SUMS
	# The install page's commands, one for each platform.
	if [ "$(uname -s)" = Darwin ]; then
		set -- shasum -a 256 --check --ignore-missing SHA256SUMS
	else
		set -- sha256sum --check --ignore-missing SHA256SUMS
	fi
	echo "\$ $*"
	out=$("$@" 2>&1) || fail "$1 refused the archive: $out"
	printf '%s\n' "$out"
	printf '%s\n' "$out" | grep -qx "$archive: OK" || fail "$1 did not check $archive"
	show tar -xzf "$archive" metais || fail "could not unpack metais from $archive"
	check_version "$PWD/metais" "$version"
	;;

version)
	[ "$#" -eq 2 ] || usage
	check_version "$1" "$(release "$2")"
	;;

skill)
	[ "$#" -eq 3 ] || usage
	version=$(release "$2")
	[ ! -e "$3" ] || fail "$3 exists"
	mkdir -p "$3"
	cd "$3"
	show "$1" skill --harness claude || fail "metais skill exited $?"
	file=.claude/skills/metais/SKILL.md
	[ -f "$file" ] || fail "metais skill did not write $file"
	show head -n 5 "$file"
	echo "..."
	stamp=$(tail -n 1 "$file")
	printf '%s\n' "$stamp"
	case $stamp in
	"<!-- metais skill: version $version, sha256 "*) ;;
	*) fail "$file does not end naming version $version" ;;
	esac
	echo "ok: metais skill wrote $file, of version $version"
	;;

ids)
	[ "$#" -eq 1 ] || usage
	[ -d "$1/spec/schemas" ] || fail "$1 holds no spec/schemas"
	find "$1/spec/schemas" "$1/docs/cli/schemas" -name '*.json' -type f | LC_ALL=C sort >"$tmp/schemas"
	while IFS= read -r f; do
		n=$((n + 1))
		id=$(jq -r '."$id" // empty' "$f") || id=
		path=${id#https://metais.org/}
		if [ -z "$id" ] || [ "$path" = "$id" ]; then
			echo "FAIL: ${f#"$1"/} has no \$id under https://metais.org/: \"$id\"" >&2
			bad=$((bad + 1))
			continue
		fi
		get "$site/$path" "$tmp/body"
		served=$(jq -r '."$id" // empty' "$tmp/body" 2>/dev/null) || served="(not JSON)"
		case $code:$type in
		200:application/json*) ;;
		*)
			echo "FAIL: $id answered $code $type, not 200 application/json" >&2
			bad=$((bad + 1))
			continue
			;;
		esac
		if [ "$served" != "$id" ]; then
			echo "FAIL: $id serves JSON whose \$id is \"$served\"" >&2
			bad=$((bad + 1))
		fi
	done <"$tmp/schemas"
	[ "$n" -gt 0 ] || fail "$1 holds no schema"
	finish "schemas' \$ids resolve as JSON with their own \$id"
	;;

go-import)
	[ "$#" -eq 0 ] || usage
	want='<meta name="go-import" content="metais.org/metais git https://github.com/kottera/metais">'
	echo "each must hold: $want"
	# The go command's own request for each package path a user can name
	# (ADR-013 D1 item 2).
	for p in metais metais/cmd/metais; do
		n=$((n + 1))
		get "$site/$p?go-get=1" "$tmp/page"
		grep -o '<meta name="go-[a-z]*" content="[^"]*">' "$tmp/page" || true
		if [ "$code" != 200 ] || ! grep -qF "$want" "$tmp/page"; then
			echo "FAIL: metais.org/$p answered $code without the go-import tag" >&2
			bad=$((bad + 1))
		fi
	done
	finish "go-import pages serve the module's go-import tag"
	;;

reference)
	[ "$#" -eq 1 ] || usage
	echo "\$ $1 help --json"
	"$1" help --json >"$tmp/help.json" || fail "metais help --json exited $?"
	{
		echo reference/
		echo reference/skill/
		jq -r '.result.commands[].name | "reference/commands/\(.)/"' "$tmp/help.json"
		jq -r '.result.topics[].name | "reference/topics/\(.)/"' "$tmp/help.json"
	} >"$tmp/pages" || fail "metais help --json does not list commands and topics"
	[ "$(wc -l <"$tmp/pages")" -gt 2 ] || fail "metais help --json lists no command or topic"
	while IFS= read -r p; do
		n=$((n + 1))
		page "$site/$p"
	done <"$tmp/pages"
	finish "reference pages serve HTML"
	;;

spec)
	[ "$#" -eq 1 ] || usage
	[ -f "$1/spec/README.md" ] || fail "$1 holds no spec/README.md"
	for f in "$1"/spec/*.md; do
		name=$(basename "$f" .md)
		n=$((n + 1))
		if [ "$name" = README ]; then p=spec/draft/; else p=spec/draft/$name/; fi
		page "$site/$p"
		if [ "$code" = 200 ] && ! grep -qF '<meta name="robots" content="noindex">' "$tmp/page"; then
			echo "FAIL: $site/$p does not ask not to be indexed, as a draft does" >&2
			bad=$((bad + 1))
		fi
	done
	finish "chapters of the draft specification serve HTML, unindexed"
	;;

pages)
	[ "$#" -eq 1 ] || usage
	[ -f "$1/site/public/install.sh" ] || fail "$1 holds no site/public/install.sh"
	for p in "" install/; do
		n=$((n + 1))
		page "$site/$p"
	done
	n=$((n + 1))
	get "$site/install.sh" "$tmp/install.sh"
	if [ "$code" != 200 ] || ! cmp "$tmp/install.sh" "$1/site/public/install.sh"; then
		echo "FAIL: $site/install.sh answered $code, not the release's site/public/install.sh" >&2
		bad=$((bad + 1))
	fi
	finish "pages serve, install.sh as the release holds it"
	;;

kottera)
	[ "$#" -eq 0 ] || usage
	n=1
	page "$kottera/"
	sed -n 's:.*<title>\(.*\)</title>.*:title\: \1:p' "$tmp/page"
	finish "kottera.com home page serves HTML"
	;;

*) usage ;;
esac
