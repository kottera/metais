#!/bin/sh
# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# install-test.sh is ci.yml's real install with site/public/install.sh
# (ADR-013 D6 item 8). From the root of a clean clone:
#
#   install-test.sh <dir>
#
# It tags HEAD v0.0.0-install in this clone alone, and builds that release
# with internal/tools/release, as a prerelease (ADR-011 A8 item 5). It lays
# the release's archives and SHA256SUMS out under <dir>, which must not
# exist, at the paths github.com serves a release's assets from, and serves
# them on 127.0.0.1 with python3's http.server. install.sh then runs as
# published, with its host's own tar, sha256sum or shasum, and curl: the only
# thing changed is that the curl PATH finds first is a wrapper that sends
# https://github.com/ and https://api.github.com/ to that server.
#
# It installs the release pinned as 0.0.0-install into a named directory,
# and as v0.0.0-install into ~/.local/bin under a fresh HOME, and requires
# each to be the archive's own binary, reporting the version. With no
# METAIS_VERSION it requires the refusal of a repository with no stable
# release, which is every repository the server serves, since it answers
# 404 for the latest one. It removes the tag on exit.
#
# Exit 0 when every install and the refusal are as required, 1 otherwise.
set -eu

if [ "$#" -ne 1 ] || [ -e "$1" ]; then
	echo "usage: install-test.sh <dir>, a directory that does not exist" >&2
	exit 1
fi
mkdir -p "$1"
work=$(cd "$1" && pwd)
version=0.0.0-install
port=8417
web=$work/web
rel=$web/github.com/kottera/metais/releases/download/v$version
server=

cleanup() {
	[ -z "$server" ] || kill "$server" 2>/dev/null || true
	git tag -d "v$version" >/dev/null 2>&1 || true
}
trap cleanup EXIT

fail() {
	echo "install-test.sh: $*" >&2
	exit 1
}

git tag "v$version"
go run ./internal/tools/release -version "$version"
mkdir -p "$rel"
mv dist/* "$rel/"
rmdir dist

python3 -m http.server --bind 127.0.0.1 --directory "$web" "$port" >"$work/http.log" 2>&1 &
server=$!
n=0
until curl -fsS -o /dev/null "http://127.0.0.1:$port/github.com/kottera/metais/releases/download/v$version/SHA256SUMS"; do
	n=$((n + 1))
	[ "$n" -lt 30 ] || fail "the release server did not start: $(cat "$work/http.log")"
	sleep 1
done

mkdir "$work/bin"
cat >"$work/bin/curl" <<EOF
#!/bin/sh
for a do
	case \$a in
	https://github.com/* | https://api.github.com/*)
		echo "\$a" >>"$work/requested"
		a=http://127.0.0.1:$port/\${a#https://}
		;;
	esac
	set -- "\$@" "\$a"
	shift
done
exec "$(command -v curl)" "\$@"
EOF
chmod 755 "$work/bin/curl"

case $(uname -s) in Darwin) os=darwin ;; *) os=linux ;; esac
case $(uname -m) in x86_64) arch=amd64 ;; *) arch=arm64 ;; esac
archive=metais_${version}_${os}_${arch}.tar.gz
mkdir "$work/want"
tar -xzf "$rel/$archive" -C "$work/want" metais

# try_install <name> <metais> <env>... runs install.sh with env, and requires it
# to have installed the archive's metais at <metais>, fetching the archive
# and SHA256SUMS and nothing else.
try_install() {
	name=$1 bin=$2
	shift 2
	rm -f "$work/requested"
	env PATH="$work/bin:$PATH" "$@" sh site/public/install.sh >"$work/$name.out" 2>&1 ||
		fail "$name: install.sh failed: $(cat "$work/$name.out")"
	cat "$work/$name.out"
	cmp "$bin" "$work/want/metais" || fail "$name: $bin is not the archive's metais"
	got=$("$bin" version | head -n 1)
	[ "$got" = "metais $version" ] || fail "$name: $bin version says \"$got\", want \"metais $version\""
	printf '%s\n' "https://github.com/kottera/metais/releases/download/v$version/$archive" \
		"https://github.com/kottera/metais/releases/download/v$version/SHA256SUMS" >"$work/want-requested"
	cmp "$work/requested" "$work/want-requested" || fail "$name: install.sh fetched $(cat "$work/requested")"
	grep -q 'metais skill' "$work/$name.out" || fail "$name: install.sh does not suggest metais skill"
}

try_install pinned "$work/dir/metais" METAIS_VERSION="$version" METAIS_INSTALL_DIR="$work/dir"
grep -q "$work/dir is not on your PATH" "$work/pinned.out" || fail "pinned: install.sh does not say $work/dir is not on PATH"
mkdir "$work/home"
try_install default "$work/home/.local/bin/metais" HOME="$work/home" METAIS_VERSION="v$version"

if env PATH="$work/bin:$PATH" HOME="$work/home" METAIS_INSTALL_DIR="$work/latest" sh site/public/install.sh >"$work/latest.out" 2>&1; then
	fail "latest: install.sh succeeded with no stable release: $(cat "$work/latest.out")"
fi
cat "$work/latest.out"
for want in 'no stable release' METAIS_VERSION; do
	grep -q "$want" "$work/latest.out" || fail "latest: install.sh's refusal does not say \"$want\""
done
[ ! -e "$work/latest" ] || fail "latest: the refused install wrote $work/latest"
echo "install-test.sh: every install and the refusal are as required"
