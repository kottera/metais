// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"metais.org/metais/internal/tools/release/buildline"
)

// errSums is the formula generator's refusal of a SHA256SUMS that is not
// exactly the one the builder writes for the version.
var errSums = errors.New("SHA256SUMS is not the release's")

// errBrewVersion refuses a version Homebrew would read otherwise from the
// archive's URL.
var errBrewVersion = errors.New("Homebrew does not read this version from an archive's URL; a formula's version is MAJOR.MINOR.PATCH, optionally with a prerelease suffix -alpha, -beta, -pre or -rc and an optional .<number>")

// releaseURL is where the GitHub release of v<version> serves its assets
// (ADR-013 D6 item 5, D7 item 3).
const releaseURL = "https://github.com/kottera/metais/releases/download/v"

// The Homebrew block each shipped target's archive is selected in: the
// operating system's, then the architecture's.
var (
	brewOS   = map[string]string{"darwin": "on_macos", "linux": "on_linux"}
	brewArch = map[string]string{"amd64": "on_intel", "arm64": "on_arm"}
)

// formulaCLI is the release command's formula subcommand: it reads the
// SHA256SUMS -sums names and writes the formula for -version to stdout,
// or a refusal to stderr. It returns the exit status: 0 when the formula
// is written, 1 on a refusal, 2 on a usage error.
func formulaCLI(args []string, stdout, stderr io.Writer) int {
	const usage = "usage: go run ./internal/tools/release formula -version <semver> -sums <SHA256SUMS>"
	fs := flag.NewFlagSet("release formula", flag.ContinueOnError)
	fs.SetOutput(stderr)
	version := fs.String("version", "", "the release's semantic version, without a leading v")
	sums := fs.String("sums", "", "the release's SHA256SUMS, as the builder wrote it to dist/")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 || *version == "" || *sums == "" {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	b, err := os.ReadFile(*sums)
	if err == nil {
		b, err = formula(*version, b)
	}
	if err == nil {
		_, err = stdout.Write(b)
	}
	if err != nil {
		fmt.Fprintln(stderr, "release formula:", err)
		return 1
	}
	return 0
}

// parseSums is the SHA-256 of each shipped target's archive that sums,
// the SHA256SUMS of the release version, lists. sums must be exactly what
// the builder writes for version: one "<sha256>  <archive>" line per
// shipped target, in byte order of the name, and nothing else.
func parseSums(version string, sums []byte) (map[buildline.Target]string, error) {
	want := make([]string, len(buildline.Shipped))
	targets := map[string]buildline.Target{}
	for i, t := range buildline.Shipped {
		want[i] = buildline.ArchiveName(version, t)
		targets[want[i]] = t
	}
	slices.Sort(want)
	text := string(sums)
	if !strings.HasSuffix(text, "\n") {
		return nil, fmt.Errorf("%w: it does not end with a newline", errSums)
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(lines) != len(want) {
		return nil, fmt.Errorf("%w: it has %d lines, and a release has one for each of %d archives", errSums, len(lines), len(want))
	}
	hashes := map[buildline.Target]string{}
	for i, line := range lines {
		sum, name, ok := strings.Cut(line, "  ")
		if !ok || len(sum) != 64 || strings.Trim(sum, "0123456789abcdef") != "" {
			return nil, fmt.Errorf("%w: line %d, %q, is not \"<sha256>  <archive>\"", errSums, i+1, line)
		}
		if name != want[i] {
			return nil, fmt.Errorf("%w: line %d names %q, and a release of %s has %q there", errSums, i+1, name, version, want[i])
		}
		hashes[targets[name]] = sum
	}
	return hashes, nil
}

// formula is the Homebrew formula metais of the tap kottera/tap (ADR-013
// D1 item 3, D7 item 5) for the release version, whose SHA256SUMS is
// sums. Each shipped target's archive is fetched from the release, and
// its SHA-256 is its line in sums, so neither can drift from what the
// release published. sums must be exactly what the builder writes for
// version (parseSums). The version is not stated, because
// brew audit refuses a version Homebrew can read from the URL, so version
// must be one it reads there exactly (brewVersion).
func formula(version string, sums []byte) ([]byte, error) {
	if !releaseVersion(version) {
		return nil, fmt.Errorf("%w: %q", errVersion, version)
	}
	if !brewVersion(version) {
		return nil, fmt.Errorf("%w: %q", errBrewVersion, version)
	}
	hashes, err := parseSums(version, sums)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	fmt.Fprintf(&b, `# Generated from the SHA256SUMS of metais v%s by
# go run ./internal/tools/release formula. Do not edit it by hand.
class Metais < Formula
  desc "Reference tool for METAIS packages, which turn input data into files"
  homepage "https://metais.org"
  license "Apache-2.0"
`, version)
	for i, t := range buildline.Shipped {
		osBlock, arch := brewOS[t.OS], brewArch[t.Arch]
		if osBlock == "" || arch == "" {
			return nil, fmt.Errorf("the formula has no Homebrew block for the shipped target %s", t)
		}
		first := i == 0 || buildline.Shipped[i-1].OS != t.OS
		if first {
			fmt.Fprintf(&b, "\n  %s do\n", osBlock)
		} else {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "    %s do\n      url \"%s%s/%s\"\n      sha256 \"%s\"\n    end\n",
			arch, releaseURL, version, buildline.ArchiveName(version, t), hashes[t])
		if i == len(buildline.Shipped)-1 || buildline.Shipped[i+1].OS != t.OS {
			b.WriteString("  end\n")
		}
	}
	b.WriteString(`
  def install
    bin.install "metais"
  end

  def caveats
    <<~EOS
      To teach a coding agent to use metais, install its agent skill
      into a project by running, at the project's top level:
        metais skill
    EOS
  end

  test do
    assert_match "metais #{version}", shell_output("#{bin}/metais version")
  end
end
`)
	return []byte(b.String()), nil
}

// brewVersion reports whether Homebrew reads the release version v, as
// releaseVersion accepts it, back exactly from an archive's URL: a version
// with no prerelease suffix, or with one of the words Homebrew's version
// scanner knows, alone or followed by .<number> (1.0.0-rc.1, 1.0.0-beta).
// It reads 1.0.0-0, 1.0.0-dev and 1.0.0-preview.4 as other versions.
func brewVersion(v string) bool {
	_, pre, ok := strings.Cut(v, "-")
	if !ok {
		return true
	}
	word, n, numbered := strings.Cut(pre, ".")
	return slices.Contains([]string{"alpha", "beta", "pre", "rc"}, word) && (!numbered || numeral(n))
}
