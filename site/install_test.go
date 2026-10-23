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
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
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
