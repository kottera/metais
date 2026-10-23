// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/tools/release/buildline"
)

// fakeTool is a metais standing in for a release's: a script that answers
// version --json with the version and implements (a JSON array's
// members) it was written with.
const fakeTool = `#!/bin/sh
[ "$1 $2" = "version --json" ] || exit 2
echo '{"format":"metais.version/1","ok":true,"result":{"tool":{"name":"metais","version":"%s"},"specification":{"implements":[%s]}}}'
`

// writeAssets writes into dir the assets of the release version as a
// draft holds them: an archive per shipped target, each holding a fakeTool
// that reports reported and implements, and their SHA256SUMS.
func writeAssets(t *testing.T, dir, version, reported, implements string) {
	t.Helper()
	tool := strings.Replace(strings.Replace(fakeTool, "%s", reported, 1), "%s", implements, 1)
	var sums strings.Builder
	var names []string
	for _, tg := range buildline.Shipped {
		names = append(names, buildline.ArchiveName(version, tg))
	}
	slices.Sort(names)
	for _, name := range names {
		var b bytes.Buffer
		zw := gzip.NewWriter(&b)
		tw := tar.NewWriter(zw)
		for _, f := range []struct {
			name, data string
			mode       int64
		}{{"LICENSE", "the licence\n", 0o644}, {"metais", tool, 0o755}} {
			if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: f.mode, Size: int64(len(f.data)), Typeflag: tar.TypeReg}); err != nil {
				t.Fatal(err)
			}
			tw.Write([]byte(f.data))
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(b.Bytes())
		sums.WriteString(hex.EncodeToString(sum[:]) + "  " + name + "\n")
	}
	if err := os.WriteFile(filepath.Join(dir, sumsName), []byte(sums.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestVerify holds verify to ADR-013 D7 item 3's check of a draft's
// assets: exactly SHA256SUMS and the shipped targets' archives, each
// archive's SHA-256 its line's, and a tool that reports the release's
// version. Each refusal breaks one of these and nothing else.
func TestVerify(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake release's tool is a shell script")
	}
	const v = "1.0.0-rc.1"
	host := buildline.ArchiveName(v, buildline.Target{OS: runtime.GOOS, Arch: runtime.GOARCH})
	other := buildline.ArchiveName(v, buildline.Shipped[0])
	if other == host {
		other = buildline.ArchiveName(v, buildline.Shipped[1])
	}
	good := func(t *testing.T) string {
		dir := testenv.Dir(t)
		writeAssets(t, dir, v, v, `"1.0"`)
		return dir
	}
	got, err := verify(v, good(t))
	if err != nil || !slices.Equal(got, []string{"1.0"}) {
		t.Fatalf("verify of a release's assets returned %q, %v", got, err)
	}
	for _, c := range []struct {
		name    string
		version string
		change  func(t *testing.T, dir string)
		want    string
	}{
		{"another version", "1.0.0-rc.2", func(*testing.T, string) {}, "and a release of 1.0.0-rc.2 has exactly"},
		{"not a release version", "v1.0.0-rc.1", func(*testing.T, string) {}, "not a semantic version"},
		{"a file more", v, func(t *testing.T, dir string) { write(t, dir, "notes.txt", "x") }, `"notes.txt"`},
		{"an archive missing", v, func(t *testing.T, dir string) { os.Remove(filepath.Join(dir, other)) }, "and a release of 1.0.0-rc.1 has exactly"},
		{"no SHA256SUMS", v, func(t *testing.T, dir string) { os.Remove(filepath.Join(dir, sumsName)) }, "and a release of 1.0.0-rc.1 has exactly"},
		{"a directory", v, func(t *testing.T, dir string) { os.Mkdir(filepath.Join(dir, "sub"), 0o755) }, "sub is not a regular file"},
		{"an archive changed", v, func(t *testing.T, dir string) {
			p := filepath.Join(dir, other)
			b, _ := os.ReadFile(p)
			write(t, filepath.Dir(p), filepath.Base(p), string(append(b, 0)))
		}, other + "'s SHA-256 is"},
		{"a line changed", v, func(t *testing.T, dir string) {
			p := filepath.Join(dir, sumsName)
			b, _ := os.ReadFile(p)
			write(t, filepath.Dir(p), filepath.Base(p), "0"+string(b[1:]))
		}, "SHA-256 is"},
		{"SHA256SUMS not the builder's", v, func(t *testing.T, dir string) {
			p := filepath.Join(dir, sumsName)
			b, _ := os.ReadFile(p)
			write(t, filepath.Dir(p), filepath.Base(p), string(b)+"\n")
		}, "SHA256SUMS is not the release's"},
		{"a tool of another version", v, func(t *testing.T, dir string) { writeAssets(t, dir, v, "1.0.0-rc.2", `"1.0"`) }, `reports version "1.0.0-rc.2"`},
		{"a tool that implements nothing", v, func(t *testing.T, dir string) { writeAssets(t, dir, v, v, "") }, "not one or more revisions"},
		{"a tool that implements no revision", v, func(t *testing.T, dir string) { writeAssets(t, dir, v, v, `"draft"`) }, "not one or more revisions"},
	} {
		dir := good(t)
		c.change(t, dir)
		got, err := verify(c.version, dir)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: verify returned %q, %v; want a refusal saying %q", c.name, got, err, c.want)
		}
	}
}

func TestVerifyCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake release's tool is a shell script")
	}
	dir := testenv.Dir(t)
	writeAssets(t, dir, "1.0.0", "1.0.0", `"1.0","1.1"`)
	var stdout, stderr bytes.Buffer
	if code := cli([]string{"verify", "-version", "1.0.0", "-dir", dir}, &stdout, &stderr, nil); code != 0 {
		t.Fatalf("verify exited %d: %s", code, stderr.String())
	}
	if stdout.String() != "1.0\n1.1\n" {
		t.Errorf("verify wrote %q to stdout, want the revisions one per line", stdout.String())
	}
	for _, c := range []struct {
		args []string
		code int
	}{
		{[]string{"verify"}, 2},
		{[]string{"verify", "-version", "1.0.0"}, 2},
		{[]string{"verify", "-dir", dir}, 2},
		{[]string{"verify", "-version", "1.0.0", "-dir", dir, "extra"}, 2},
		{[]string{"verify", "-version", "1.0.1", "-dir", dir}, 1},
		{[]string{"verify", "-version", "1.0.0", "-dir", filepath.Join(dir, "absent")}, 1},
	} {
		var stdout, stderr bytes.Buffer
		if code := cli(c.args, &stdout, &stderr, nil); code != c.code {
			t.Errorf("cli(%q) exited %d, want %d", c.args, code, c.code)
		}
		if stdout.Len() > 0 {
			t.Errorf("cli(%q) wrote %q to stdout", c.args, stdout.String())
		}
		if stderr.Len() == 0 {
			t.Errorf("cli(%q) wrote nothing to stderr", c.args)
		}
	}
}
