#!/bin/sh
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# install.sh installs metais, the reference tool of the METAIS package
# format, from its GitHub release (ADR-013 D6):
#
#   curl -fsSL https://metais.org/install.sh | sh
#
# It downloads this machine's archive and the release's SHA256SUMS from
# github.com, refuses unless the archive's SHA-256 is the one SHA256SUMS
# lists for it, and copies metais into ~/.local/bin. It never runs sudo and
# never edits a shell's startup files.
#
#   METAIS_VERSION      the version to install, as 1.0.0 or v1.0.0; without
#                       it, the latest release that is not a prerelease
#   METAIS_INSTALL_DIR  the directory to install into, ~/.local/bin without it
set -eu

repo=kottera/metais
download=https://github.com/$repo/releases/download
latest=https://github.com/$repo/releases/latest
routes="go install metais.org/metais/cmd/metais@latest, which builds it from source, or the archives at https://github.com/$repo/releases"

fail() {
	echo "install.sh: $*" >&2
	exit 1
}

case $(uname -s) in
Darwin) os=darwin ;;
Linux) os=linux ;;
*) fail "metais is released for macOS and Linux, and this is $(uname -s). Other routes: $routes" ;;
esac
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) fail "metais is released for amd64 and arm64, and this is $(uname -m). Other routes: $routes" ;;
esac
command -v curl >/dev/null 2>&1 || fail "install.sh downloads with curl, which is not on PATH"

dir=${METAIS_INSTALL_DIR:-${HOME:?install.sh: set HOME or METAIS_INSTALL_DIR}/.local/bin}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
trap 'exit 1' HUP INT TERM

version=${METAIS_VERSION:-}
if [ -z "$version" ]; then
	# github.com answers its latest release, the newest that is not a
	# prerelease, with a redirect to that release's tag, and sends a
	# repository with only prereleases to its list of releases instead.
	# The REST API at api.github.com would say the same, but limits an
	# anonymous caller to a few requests an hour for its whole network.
	answer=$(curl -sS -o /dev/null -w '%{http_code} %{redirect_url}' "$latest") ||
		fail "could not ask GitHub for the latest release at $latest"
	code=${answer%% *}
	to=${answer#* }
	case $code:$to in
	30[1278]:https://github.com/"$repo"/releases/tag/?*) version=${to##*/} ;;
	30[1278]:https://github.com/"$repo"/releases | 30[1278]:https://github.com/"$repo"/releases/ | 404:*)
		fail "metais has no stable release yet, only prereleases. Pick a version from https://github.com/$repo/releases and set METAIS_VERSION to it: curl -fsSL https://metais.org/install.sh | METAIS_VERSION=<version> sh" ;;
	403:* | 429:*) fail "GitHub refused to name the latest release, HTTP $code from $latest, as it does when this network has asked it too often. Try again later, or set METAIS_VERSION to install a version by name" ;;
	30[1278]:*) fail "GitHub sent $latest to $to, which is not a release. Set METAIS_VERSION to install a version by name" ;;
	*) fail "GitHub answered HTTP $code for the latest release at $latest. Set METAIS_VERSION to install a version by name" ;;
	esac
fi
version=${version#v}
case $version in
'' | *[!0-9A-Za-z.+-]*) fail "\"$version\" is not a metais version, such as 1.0.0" ;;
esac

archive=metais_${version}_${os}_${arch}.tar.gz
echo "Downloading metais $version for $os/$arch"
curl -fsSL -o "$tmp/$archive" "$download/v$version/$archive" ||
	fail "could not download $download/v$version/$archive"
curl -fsSL -o "$tmp/SHA256SUMS" "$download/v$version/SHA256SUMS" ||
	fail "could not download $download/v$version/SHA256SUMS"

want=$(awk -v f="$archive" '$2 == f { n++; h = $1 } END { if (n == 1) print h }' "$tmp/SHA256SUMS")
[ -n "$want" ] || fail "the release's SHA256SUMS has no line for $archive, so it cannot be verified. Nothing was installed"
if command -v sha256sum >/dev/null 2>&1; then
	got=$(sha256sum "$tmp/$archive")
elif command -v shasum >/dev/null 2>&1; then
	got=$(shasum -a 256 "$tmp/$archive")
else
	fail "install.sh verifies the download with sha256sum or shasum, and neither is on PATH"
fi
got=${got%% *}
[ "$got" = "$want" ] ||
	fail "$archive has SHA-256 $got, but the release's SHA256SUMS lists $want. Nothing was installed"

mkdir "$tmp/x"
tar -xzf "$tmp/$archive" -C "$tmp/x" metais
mkdir -p "$dir"
cp "$tmp/x/metais" "$dir/.metais.$$"
chmod 755 "$dir/.metais.$$"
mv -f "$dir/.metais.$$" "$dir/metais"
echo "Installed metais $version in $dir/metais"

case :$PATH: in
*:"$dir":*) ;;
*)
	echo
	echo "$dir is not on your PATH. Add this line to your shell's startup file:"
	echo
	echo "  export PATH=\"$dir:\$PATH\""
	;;
esac
echo
echo "Next, run metais skill at the top of a project, to give its coding agent the skill that teaches it metais."
