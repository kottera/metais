// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package site

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/tools/release/buildline"
)

// installScript is the install script (ADR-013 D6), from this directory, as
// the site serves it at /install.sh.
const installScript = "public/install.sh"

// The tests run the script as published, with no variable or flag of its own
// pointing it elsewhere: the script names github.com and api.github.com and
// nothing else, so no setting of a user's can change what it downloads or how
// it verifies it. What the tests change is what PATH finds for curl and uname,
// as fakeCurl and fakeUname, so the script reads a release laid out in a
// temporary directory and sees the platform a test names, without the network.
// sha256sum is this test binary, by TestMain, so the hash the script compares
// does not rest on which of sha256sum and shasum a host has, or on shasum's
// perl being able to run; the install jobs of ci.yml run the script with each
// host's own curl and hash tool.

// sha256sum prints each file's SHA-256 as sha256sum does.
func sha256sum(files []string) int {
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, "sha256sum:", err)
			return 1
		}
		sum := sha256.Sum256(b)
		fmt.Printf("%s  %s\n", hex.EncodeToString(sum[:]), f)
	}
	return 0
}

// fakeCurl answers a URL https://<host>/<path> from the file $FAKE_WEB/<host>/<path>,
// for the curl options install.sh passes: -o, -w '%{http_code}', and -f, under
// which a missing file is curl's exit status 22 rather than a 404 body. It
// appends every URL it is asked for to $FAKE_WEB.log.
const fakeCurl = `#!/bin/sh
out= fmt= f= url=
while [ "$#" -gt 0 ]; do
	case $1 in
	-o) out=$2; shift ;;
	-w) fmt=$2; shift ;;
	-*f*) f=1 ;;
	-*) ;;
	*) url=$1 ;;
	esac
	shift
done
echo "$url" >>"$FAKE_WEB.log"
file=$FAKE_WEB/${url#https://}
if [ -f "$file" ]; then
	code=200
else
	[ -n "$f" ] && exit 22
	code=404 file=/dev/null
fi
if [ -n "$out" ]; then cp "$file" "$out"; else cat "$file"; fi
[ -n "$fmt" ] && printf '%s' "$code"
exit 0
`

// fakeUname prints $FAKE_UNAME_S for -s and $FAKE_UNAME_M for -m.
const fakeUname = `#!/bin/sh
case $1 in
-s) echo "$FAKE_UNAME_S" ;;
-m) echo "$FAKE_UNAME_M" ;;
esac
`

// release is the release a test lays out: version, without its v, and the
// metais binary each archive holds, which is a script printing its version.
const (
	releaseVersion = "1.2.3"
	releaseTag     = "v" + releaseVersion
	releaseBin     = "#!/bin/sh\necho metais " + releaseVersion + "\n"
)

const (
	downloadPrefix = "github.com/kottera/metais/releases/download/"
	latestPath     = "api.github.com/repos/kottera/metais/releases/latest"
)

// installHost is one run's host: a fake web, a bin directory holding
// fakeCurl, fakeUname and sha256sum, a HOME and the directory to install into.
type installHost struct {
	web, bin, home, dir string
}

// newInstallHost lays out a release of releaseVersion on a fake web, with
// one archive per shipped target and the SHA256SUMS that lists them, and no
// latest release. sums, when not nil, rewrites SHA256SUMS before it is
// written.
func newInstallHost(t *testing.T, sums func(string) string) installHost {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is a POSIX shell script for macOS and Linux")
	}
	root := testenv.Dir(t)
	h := installHost{
		web:  filepath.Join(root, "web"),
		bin:  filepath.Join(root, "bin"),
		home: testenv.Home(t),
		dir:  filepath.Join(root, "install"),
	}
	writeFile(t, filepath.Join(h.bin, "curl"), fakeCurl, 0o755)
	writeFile(t, filepath.Join(h.bin, "uname"), fakeUname, 0o755)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(exe, filepath.Join(h.bin, "sha256sum")); err != nil {
		t.Fatal(err)
	}

	rel := filepath.Join(h.web, filepath.FromSlash(downloadPrefix+releaseTag))
	var list strings.Builder
	for _, target := range []string{"darwin_amd64", "darwin_arm64", "linux_amd64", "linux_arm64"} {
		name := "metais_" + releaseVersion + "_" + target + ".tar.gz"
		b := tarGz(t, map[string]string{"metais": releaseBin, "LICENSE": "licence\n"})
		writeFile(t, filepath.Join(rel, name), string(b), 0o644)
		sum := sha256.Sum256(b)
		list.WriteString(hex.EncodeToString(sum[:]) + "  " + name + "\n")
	}
	s := list.String()
	if sums != nil {
		s = sums(s)
	}
	writeFile(t, filepath.Join(rel, "SHA256SUMS"), s, 0o644)
	return h
}

// setLatest makes tag the latest release that is not a prerelease, as
// GitHub's API answers it.
func (h installHost) setLatest(t *testing.T, tag string) {
	t.Helper()
	writeFile(t, filepath.Join(h.web, filepath.FromSlash(latestPath)),
		"{\n  \"url\": \"https://api.github.com/repos/kottera/metais/releases/1\",\n  \"tag_name\": \""+tag+"\",\n  \"prerelease\": false\n}\n", 0o644)
}

// run runs install.sh as curl ... | sh runs it, on a host whose uname says
// sysname and machine, with env added to a bare environment. It returns the
// script's output and whether it succeeded.
func (h installHost) run(t *testing.T, sysname, machine string, env ...string) (string, bool) {
	t.Helper()
	script, err := os.ReadFile(installScript)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh")
	cmd.Stdin = bytes.NewReader(script)
	cmd.Dir = testenv.Dir(t)
	cmd.Env = append([]string{
		"PATH=" + h.bin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + h.home,
		"TMPDIR=" + testenv.Dir(t),
		"LC_ALL=C",
		"FAKE_WEB=" + h.web,
		"FAKE_UNAME_S=" + sysname,
		"FAKE_UNAME_M=" + machine,
	}, env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatal(err)
		}
	}
	return string(out), err == nil
}

// requested is every URL the fake curl was asked for.
func (h installHost) requested(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(h.web + ".log")
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(b))
}

// requireNothingInstalled requires dir to hold no metais, and no file at all.
func requireNothingInstalled(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("a refused install wrote into %s: %v", dir, entries)
	}
}

// requireInstalled requires dir/metais to be the release's binary,
// executable.
func requireInstalled(t *testing.T, dir string) {
	t.Helper()
	p := filepath.Join(dir, "metais")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != releaseBin {
		t.Errorf("%s holds %q, want the release's %q", p, b, releaseBin)
	}
	if fi, err := os.Stat(p); err != nil {
		t.Fatal(err)
	} else if fi.Mode().Perm() != 0o755 {
		t.Errorf("%s has mode %v, want 0755", p, fi.Mode().Perm())
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("%s holds %v, want metais alone", dir, entries)
	}
}

func requireContains(t *testing.T, out string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("install.sh's output does not say %q:\n%s", w, out)
		}
	}
}

// TestInstall installs a pinned version, spelled with and without its v, on
// each platform the script supports, and requires the platform's archive and
// SHA256SUMS to be fetched from the GitHub release and nothing else, the
// binary installed in METAIS_INSTALL_DIR, the line that adds the directory to
// PATH printed, and metais skill suggested (ADR-013 D6 items 4 to 7 and 9).
func TestInstall(t *testing.T) {
	for _, c := range []struct{ sysname, machine, target string }{
		{"Linux", "x86_64", "linux_amd64"},
		{"Linux", "aarch64", "linux_arm64"},
		{"Darwin", "x86_64", "darwin_amd64"},
		{"Darwin", "arm64", "darwin_arm64"},
	} {
		for _, pin := range []string{releaseVersion, releaseTag} {
			t.Run(c.target+"/"+pin, func(t *testing.T) {
				h := newInstallHost(t, nil)
				out, ok := h.run(t, c.sysname, c.machine, "METAIS_VERSION="+pin, "METAIS_INSTALL_DIR="+h.dir)
				if !ok {
					t.Fatalf("install.sh failed:\n%s", out)
				}
				requireInstalled(t, h.dir)
				base := "https://" + downloadPrefix + releaseTag + "/"
				want := []string{base + "metais_" + releaseVersion + "_" + c.target + ".tar.gz", base + "SHA256SUMS"}
				if got := h.requested(t); !slices.Equal(got, want) {
					t.Errorf("install.sh fetched %q, want %q", got, want)
				}
				requireContains(t, out,
					h.dir+" is not on your PATH",
					`export PATH="`+h.dir+`:$PATH"`,
					"metais skill")
			})
		}
	}
}

// TestInstallLatest installs with no METAIS_VERSION, and requires the latest
// release that is not a prerelease, as GitHub's API names it, to be the one
// installed, into ~/.local/bin by default, with no PATH advice when that
// directory is on PATH already.
func TestInstallLatest(t *testing.T) {
	h := newInstallHost(t, nil)
	h.setLatest(t, releaseTag)
	dir := filepath.Join(h.home, ".local", "bin")
	out, ok := h.run(t, "Linux", "x86_64", "PATH="+h.bin+string(os.PathListSeparator)+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if !ok {
		t.Fatalf("install.sh failed:\n%s", out)
	}
	requireInstalled(t, dir)
	if got := h.requested(t); len(got) == 0 || got[0] != "https://"+latestPath {
		t.Errorf("install.sh fetched %q, want https://%s first", got, latestPath)
	}
	if strings.Contains(out, "not on your PATH") {
		t.Errorf("install.sh says a directory on PATH is not:\n%s", out)
	}
}

// TestInstallRefuses holds each of install.sh's refusals (ADR-013 D6 items 4,
// 5 and 7, and the brief's ruling on no stable release): each fails, says
// why, and installs nothing.
func TestInstallRefuses(t *testing.T) {
	archive := "metais_" + releaseVersion + "_linux_amd64.tar.gz"
	for _, c := range []struct {
		name             string
		sums             func(string) string
		sysname, machine string
		noVersion        bool
		want             []string
	}{
		{
			name: "a hash mismatch",
			sums: func(s string) string {
				lines := strings.Split(s, "\n")
				for i, l := range lines {
					if strings.HasSuffix(l, "  "+archive) {
						lines[i] = strings.Repeat("0", 64) + "  " + archive
					}
				}
				return strings.Join(lines, "\n")
			},
			sysname: "Linux", machine: "x86_64",
			want: []string{archive + " has SHA-256", "Nothing was installed"},
		},
		{
			name: "a missing line",
			sums: func(s string) string {
				var keep []string
				for l := range strings.SplitSeq(s, "\n") {
					if !strings.HasSuffix(l, "  "+archive) {
						keep = append(keep, l)
					}
				}
				return strings.Join(keep, "\n")
			},
			sysname: "Linux", machine: "x86_64",
			want: []string{"SHA256SUMS has no line for " + archive, "Nothing was installed"},
		},
		{
			name: "a repeated line",
			sums: func(s string) string {
				for l := range strings.SplitSeq(s, "\n") {
					if strings.HasSuffix(l, "  "+archive) {
						return s + l + "\n"
					}
				}
				return s
			},
			sysname: "Linux", machine: "x86_64",
			want: []string{"SHA256SUMS has no line for " + archive, "Nothing was installed"},
		},
		{
			name:    "a look-alike name in a directory",
			sums:    rename(archive, "dist/"+archive),
			sysname: "Linux", machine: "x86_64",
			want: []string{"SHA256SUMS has no line for " + archive, "Nothing was installed"},
		},
		{
			name:    "a look-alike name with a prefix",
			sums:    rename(archive, "x"+archive),
			sysname: "Linux", machine: "x86_64",
			want: []string{"SHA256SUMS has no line for " + archive, "Nothing was installed"},
		},
		{
			name:    "a look-alike name with a suffix",
			sums:    rename(archive, archive+".sig"),
			sysname: "Linux", machine: "x86_64",
			want: []string{"SHA256SUMS has no line for " + archive, "Nothing was installed"},
		},
		{
			name:    "an unknown system",
			sysname: "FreeBSD", machine: "amd64",
			want: []string{"this is FreeBSD", "go install metais.org/metais/cmd/metais@latest", "https://github.com/kottera/metais/releases"},
		},
		{
			name:    "an unknown machine",
			sysname: "Linux", machine: "riscv64",
			want: []string{"this is riscv64", "go install metais.org/metais/cmd/metais@latest", "https://github.com/kottera/metais/releases"},
		},
		{
			name:    "a 32-bit ARM machine",
			sysname: "Linux", machine: "armv7l",
			want: []string{"this is armv7l", "go install metais.org/metais/cmd/metais@latest"},
		},
		{
			name:    "a 32-bit x86 machine",
			sysname: "Linux", machine: "i686",
			want: []string{"this is i686", "go install metais.org/metais/cmd/metais@latest"},
		},
		{
			name:    "no stable release",
			sysname: "Linux", machine: "x86_64",
			noVersion: true,
			want:      []string{"no stable release", "METAIS_VERSION"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newInstallHost(t, c.sums)
			env := []string{"METAIS_INSTALL_DIR=" + h.dir}
			if !c.noVersion {
				env = append(env, "METAIS_VERSION="+releaseVersion)
			}
			out, ok := h.run(t, c.sysname, c.machine, env...)
			if ok {
				t.Fatalf("install.sh succeeded:\n%s", out)
			}
			requireContains(t, out, c.want...)
			requireNothingInstalled(t, h.dir)
		})
	}
}

// rename is a rewrite of SHA256SUMS that gives the line of archive, with
// its own SHA-256, the name to instead: a file whose name holds the
// archive's, which install.sh must not take for the archive.
func rename(archive, to string) func(string) string {
	return func(s string) string {
		return strings.Replace(s, "  "+archive+"\n", "  "+to+"\n", 1)
	}
}

// TestInstallNoHashTool runs install.sh on a host with neither sha256sum
// nor shasum, and requires it to refuse, since it cannot verify the
// archive, and to install nothing. The control is the same host with
// sha256sum, where it installs: what the host lacks is the hash tool
// alone.
func TestInstallNoHashTool(t *testing.T) {
	h := newInstallHost(t, nil)
	tools := testenv.Dir(t)
	for _, name := range []string{"awk", "cat", "chmod", "cp", "gzip", "head", "mkdir", "mktemp", "mv", "rm", "sed", "tar"} {
		p, err := exec.LookPath(name)
		if err != nil {
			t.Skipf("the host has no %s: %v", name, err)
		}
		if err := os.Symlink(p, filepath.Join(tools, name)); err != nil {
			t.Fatal(err)
		}
	}
	// h.bin holds curl and uname, and the test binary as sha256sum.
	bare := testenv.Dir(t)
	for _, name := range []string{"curl", "uname"} {
		if err := os.Symlink(filepath.Join(h.bin, name), filepath.Join(bare, name)); err != nil {
			t.Fatal(err)
		}
	}
	path := func(dirs ...string) string { return "PATH=" + strings.Join(dirs, string(os.PathListSeparator)) }
	env := []string{"METAIS_VERSION=" + releaseVersion, "METAIS_INSTALL_DIR=" + h.dir}

	out, ok := h.run(t, "Linux", "x86_64", append(env, path(bare, tools))...)
	if ok {
		t.Fatalf("install.sh succeeded with no hash tool:\n%s", out)
	}
	requireContains(t, out, "neither is on PATH")
	requireNothingInstalled(t, h.dir)

	out, ok = h.run(t, "Linux", "x86_64", append(env, path(h.bin, tools))...)
	if !ok {
		t.Fatalf("install.sh failed with sha256sum on PATH, so the host lacks more than a hash tool:\n%s", out)
	}
	requireInstalled(t, h.dir)
}

// shellRef matches a reference to a shell variable, $name or ${name...},
// and shellAssign the assignment of one, at the start of a line or of a
// case's branch, up to the end of the line or the branch.
var (
	shellRef    = regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)`)
	shellAssign = regexp.MustCompile(`(?m)(?:^\s*|\)\s+)([A-Za-z_][A-Za-z0-9_]*)=([^;\n]*)`)
)

// scriptEnv are the variables install.sh may read from its environment:
// its two settings, and the HOME and PATH of the user running it.
var scriptEnv = []string{"HOME", "METAIS_INSTALL_DIR", "METAIS_VERSION", "PATH"}

// TestInstallURLsConstant requires the addresses install.sh downloads
// from to be constants that no environment variable overrides (ADR-013
// D6): repo, download and latest are each assigned once, to the address
// written here, and every variable the script reads that it does not
// assign is one of scriptEnv. A run with those three names and look-alike
// settings in its environment, each pointing elsewhere, fetches from
// github.com alone.
func TestInstallURLsConstant(t *testing.T) {
	b, err := os.ReadFile(installScript)
	if err != nil {
		t.Fatal(err)
	}
	script := string(b)
	assigned := map[string][]string{}
	for _, m := range shellAssign.FindAllStringSubmatch(script, -1) {
		assigned[m[1]] = append(assigned[m[1]], strings.TrimSpace(m[2]))
	}
	for name, want := range map[string]string{
		"repo":     "kottera/metais",
		"download": "https://github.com/$repo/releases/download",
		"latest":   "https://api.github.com/repos/$repo/releases/latest",
	} {
		if got := assigned[name]; !slices.Equal(got, []string{want}) {
			t.Errorf("install.sh assigns %s %q, want once, %q", name, got, want)
		}
	}
	read := map[string]bool{}
	for _, m := range shellRef.FindAllStringSubmatch(script, -1) {
		read[m[1]] = true
	}
	if len(read) < len(scriptEnv) {
		t.Fatalf("install.sh reads %d variables; the check is blind", len(read))
	}
	for _, name := range slices.Sorted(maps.Keys(read)) {
		if _, ok := assigned[name]; !ok && !slices.Contains(scriptEnv, name) {
			t.Errorf("install.sh reads $%s, which it does not assign: a variable of its environment it does not document", name)
		}
	}

	h := newInstallHost(t, nil)
	elsewhere := "https://example.org/kottera/metais"
	env := []string{"METAIS_VERSION=" + releaseVersion, "METAIS_INSTALL_DIR=" + h.dir}
	for _, name := range []string{"repo", "download", "latest", "METAIS_REPO", "METAIS_DOWNLOAD", "METAIS_DOWNLOAD_URL", "METAIS_BASE_URL", "METAIS_RELEASES_URL", "GITHUB_SERVER_URL", "GITHUB_API_URL"} {
		env = append(env, name+"="+elsewhere)
	}
	out, ok := h.run(t, "Linux", "x86_64", env...)
	if !ok {
		t.Fatalf("install.sh failed:\n%s", out)
	}
	requireInstalled(t, h.dir)
	base := "https://" + downloadPrefix + releaseTag + "/"
	if got, want := h.requested(t), []string{base + "metais_" + releaseVersion + "_linux_amd64.tar.gz", base + "SHA256SUMS"}; !slices.Equal(got, want) {
		t.Errorf("install.sh fetched %q, want %q", got, want)
	}
}

// TestInstallServed requires the built site to serve install.sh at
// /install.sh byte for byte (ADR-013 D1 item 5).
func TestInstallServed(t *testing.T) {
	want, err := os.ReadFile(installScript)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(builtSite(t), "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("the built site's install.sh differs from %s", installScript)
	}
}

func writeFile(t *testing.T, p, data string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), mode); err != nil {
		t.Fatal(err)
	}
}

// tarGz is a .tar.gz of files, by name.
func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for _, name := range slices.Sorted(maps.Keys(files)) {
		mode := int64(0o644)
		if name == "metais" {
			mode = 0o755
		}
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(files[name])), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(files[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestInstallPage holds the install page to ADR-013 D6: item 1's four
// routes, each command as D6 writes it; item 6's install directory, its
// variable and the PATH line it prints; item 7's METAIS_VERSION; item 9's
// metais skill; item 10's xattr for a binary downloaded with a browser;
// and a link to the script, which item 3 asks a reader to read.
func TestInstallPage(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(builtSite(t), "install", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := html.UnescapeString(string(b))
	for _, want := range []string{
		"<code>brew install kottera/tap/metais</code>",
		"<code>curl -fsSL https://metais.org/install.sh | sh</code>",
		"<code>go install metais.org/metais/cmd/metais@latest</code>",
		"<code>SHA256SUMS</code>",
		"<code>~/.local/bin</code>",
		"<code>METAIS_INSTALL_DIR</code>",
		"<code>PATH</code>",
		"<code>curl -fsSL https://metais.org/install.sh | METAIS_VERSION=1.0.0 sh</code>",
		"<code>metais skill</code>",
		"<code>xattr -d com.apple.quarantine ~/.local/bin/metais</code>",
		`<a href="/install.sh">`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the install page does not hold %s", want)
		}
	}
}

// TestInstallPageRelease builds the site as pages.yml builds it for a
// release, with build.sh --release and the release's SHA256SUMS, from a
// whole clone whose HEAD is the release's tag, and requires the install
// page to list each archive of SHA256SUMS, in its order, linked to its
// download on the release, with its SHA-256, beside the release's tag and
// its SHA256SUMS. The clone is a repository made of this checkout's
// files, committed and tagged, since the checkout itself may be shallow
// and is not at a release's tag.
func TestInstallPageRelease(t *testing.T) {
	const version = "1.2.3"
	repo := releaseRepo(t, "v"+version)

	var names []string
	for _, tg := range buildline.Shipped {
		names = append(names, buildline.ArchiveName(version, tg))
	}
	slices.Sort(names)
	var sums strings.Builder
	digests := map[string]string{}
	for i, n := range names {
		digests[n] = strings.Repeat(string("0123456789abcdef"[i]), 64)
		sums.WriteString(digests[n] + "  " + n + "\n")
	}
	sumsFile := filepath.Join(testenv.Dir(t), "SHA256SUMS")
	writeFile(t, sumsFile, sums.String(), 0o644)

	out := filepath.Join(testenv.Dir(t), "site")
	cmd := buildCmd(t, testenv.Dir(t), "--release", version, sumsFile, out)
	cmd.Args[1] = filepath.Join(repo, "site", "build.sh")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sh build.sh --release %s: %v\n%s", version, err, b)
	}
	b, err := os.ReadFile(filepath.Join(out, "install", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)

	const releases = "https://github.com/kottera/metais/releases/"
	for _, want := range []string{
		"<p>metais " + version + ", from the release",
		`<a href="` + releases + `tag/v` + version + `">v` + version + `</a>`,
		`<a href="` + releases + `download/v` + version + `/SHA256SUMS"><code>SHA256SUMS</code></a>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the release's install page does not hold %s", want)
		}
	}
	row := regexp.MustCompile(`<tr><td><a href="([^"]*)"><code>([^<]*)</code></a></td><td><code>([^<]*)</code></td></tr>`)
	var got []string
	for _, m := range row.FindAllStringSubmatch(page, -1) {
		got = append(got, m[1]+" "+m[2]+" "+m[3])
	}
	var want []string
	for _, n := range names {
		want = append(want, releases+"download/v"+version+"/"+n+" "+n+" "+digests[n])
	}
	if !slices.Equal(got, want) {
		t.Errorf("the release's install page lists the archives\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if strings.Contains(page, "/releases/latest") {
		t.Errorf("the release's install page sends the reader to the latest release, as a build with no release does")
	}
}

// releaseRepo is a whole repository holding every file of this checkout
// that git does not ignore, as it is in the working tree, in one commit
// tagged tag, and checked out. The commit is written by git fast-import,
// as one pack: git add would write each of the checkout's thousands of
// files as an object of its own.
func releaseRepo(t *testing.T, tag string) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	home := testenv.Dir(t)
	git := func(dir string, stdin io.Reader, args ...string) []byte {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = testenv.GitEnv(append(slices.Clone(hostEnviron), "HOME="+home))
		cmd.Stdin = stdin
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, stderr.Bytes())
		}
		return out
	}
	repo := testenv.Dir(t)
	var stream bytes.Buffer
	stream.WriteString("commit refs/heads/main\ncommitter Site Test <site@example.org> 0 +0000\ndata 7\nrelease\n")
	for _, name := range strings.Split(strings.TrimSuffix(string(git(root, nil, "ls-files", "-z", "--cached", "--others", "--exclude-standard")), "\x00"), "\x00") {
		src := filepath.Join(root, filepath.FromSlash(name))
		fi, err := os.Lstat(src)
		if os.IsNotExist(err) {
			continue // deleted in the working tree
		}
		if err != nil {
			t.Fatal(err)
		}
		if !fi.Mode().IsRegular() {
			t.Fatalf("%s is not a regular file", name)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(repo, filepath.FromSlash(name)), string(data), fi.Mode().Perm())
		mode := "100644"
		if fi.Mode().Perm()&0o111 != 0 {
			mode = "100755"
		}
		fmt.Fprintf(&stream, "M %s inline %s\ndata %d\n", mode, name, len(data))
		stream.Write(data)
		stream.WriteString("\n")
	}
	fmt.Fprintf(&stream, "reset refs/tags/%s\nfrom refs/heads/main\n", tag)
	git(repo, nil, "init", "-q", "-b", "main")
	git(repo, &stream, "fast-import", "--quiet")
	git(repo, nil, "reset", "-q")
	if st := git(repo, nil, "status", "--porcelain"); len(st) != 0 {
		t.Fatalf("the release's clone differs from its commit:\n%s", st)
	}
	return repo
}
