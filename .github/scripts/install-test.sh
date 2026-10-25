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
# them on 127.0.0.1 with python3's http.server. That server answers the
# repository's latest release, as github.com does, with a redirect to the URL
# the file <dir>/web/latest holds. install.sh then runs as published, with its
# host's own tar, sha256sum or shasum, and curl: the only thing changed is
# that the curl PATH finds first is a wrapper that sends https://github.com/
# to that server.
#
# It installs the release pinned as 0.0.0-install into a named directory,
# and as v0.0.0-install into ~/.local/bin under a fresh HOME, and requires
# each to be the archive's own binary, reporting the version. With no
# METAIS_VERSION it requires the refusal of a repository with no stable
# release while the latest release redirects to the list of releases, as
# github.com's does when every release is a prerelease, and an install of
# the release once it redirects to the release's tag. It removes the tag on
# exit.
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

trap '[ -z "$server" ] || kill "$server" 2>/dev/null; git tag -d "v$version" >/dev/null 2>&1' EXIT

fail() {
	echo "install-test.sh: $*" >&2
	exit 1
}

git tag "v$version"
go run ./internal/tools/release -version "$version"
mkdir -p "$rel"
mv dist/* "$rel/"
rmdir dist

echo https://github.com/kottera/metais/releases >"$web/latest"
python3 -c '
import http.server, os, sys
web = sys.argv[1]
class Handler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path != "/github.com/kottera/metais/releases/latest":
            return super().do_GET()
        with open(os.path.join(web, "latest")) as f:
            to = f.read().strip()
        self.send_response(302)
        self.send_header("Location", to)
        self.send_header("Content-Length", "0")
        self.end_headers()
http.server.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[2])),
    lambda *a: Handler(*a, directory=web)).serve_forever()
' "$web" "$port" >"$work/http.log" 2>&1 &
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
	https://github.com/*)
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

# try_install <name> <metais> <latest> <env>... runs install.sh with env, and
# requires it to have installed the archive's metais at <metais>, fetching
# the archive and SHA256SUMS and nothing else, after the latest release when
# <latest> is yes.
try_install() {
	name=$1 bin=$2 latest=$3
	shift 3
	rm -f "$work/requested"
	env PATH="$work/bin:$PATH" "$@" sh site/public/install.sh >"$work/$name.out" 2>&1 ||
		fail "$name: install.sh failed: $(cat "$work/$name.out")"
	cat "$work/$name.out"
	cmp "$bin" "$work/want/metais" || fail "$name: $bin is not the archive's metais"
	got=$("$bin" version | head -n 1)
	[ "$got" = "metais $version" ] || fail "$name: $bin version says \"$got\", want \"metais $version\""
	: >"$work/want-requested"
	[ "$latest" = no ] || echo https://github.com/kottera/metais/releases/latest >>"$work/want-requested"
	printf '%s\n' "https://github.com/kottera/metais/releases/download/v$version/$archive" \
		"https://github.com/kottera/metais/releases/download/v$version/SHA256SUMS" >>"$work/want-requested"
	cmp "$work/requested" "$work/want-requested" || fail "$name: install.sh fetched $(cat "$work/requested")"
	grep -q 'metais skill' "$work/$name.out" || fail "$name: install.sh does not suggest metais skill"
}

try_install pinned "$work/dir/metais" no METAIS_VERSION="$version" METAIS_INSTALL_DIR="$work/dir"
grep -q "$work/dir is not on your PATH" "$work/pinned.out" || fail "pinned: install.sh does not say $work/dir is not on PATH"
mkdir "$work/home"
try_install default "$work/home/.local/bin/metais" no HOME="$work/home" METAIS_VERSION="v$version"

if env PATH="$work/bin:$PATH" HOME="$work/home" METAIS_INSTALL_DIR="$work/latest" sh site/public/install.sh >"$work/latest.out" 2>&1; then
	fail "latest: install.sh succeeded with no stable release: $(cat "$work/latest.out")"
fi
cat "$work/latest.out"
for want in 'no stable release' METAIS_VERSION; do
	grep -q "$want" "$work/latest.out" || fail "latest: install.sh's refusal does not say \"$want\""
done
[ ! -e "$work/latest" ] || fail "latest: the refused install wrote $work/latest"

echo "https://github.com/kottera/metais/releases/tag/v$version" >"$web/latest"
try_install stable "$work/stable/metais" yes METAIS_INSTALL_DIR="$work/stable"
echo "install-test.sh: every install and the refusal are as required"
