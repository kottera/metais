// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command site builds the metais.org site (ADR-013 D8):
//
//	go run ./internal/tools/site -root <checkout> -out <dir> [-release <semver> -sums <SHA256SUMS>] [-data <dir>]
//
// site/build.sh is the one command that runs it. It builds metais from the
// checkout at -root, exports the site package's inputs as JSON, renders
// the package at site/package with that metais, and writes the site into
// -out, which must not exist, beside the static files of site/public and
// the schemas at their $ids. With -data, the exported inputs are also
// kept in that directory, which must not exist, so a page can be rendered
// from them with metais render --data while it is written. Every path is
// absolute.
//
// The inputs, one file each in the data directory the render reads:
//
//   - help.json: the result of metais help --json, the metais.help/1
//     document of every command and every topic;
//   - skill.json: the skill metais skill --print prints, as its text and as
//     an HTML fragment of the text after its frontmatter;
//   - spec.json: the specification's Markdown files as HTML fragments, the
//     checkout's own spec/ as the draft and the files of every spec-<M>.<m>
//     tag as that revision (ADR-011 A8 items 1 and 2), whether the draft is
//     served, which it is unless spec/ is the latest revision's, and every
//     page of those revisions in one list, which one output iterates;
//   - release.json: the release the site is built for, its version and the
//     archives SHA256SUMS lists, or published false for a build with no
//     release;
//   - tool.json: the result of the built metais's version --json.
//
// With -release and -sums, metais is built as that release is (ADR-010
// D12 item 1), and must report its version. Without them, it is built from
// the checkout with the version Go stamps from it, and release.json says
// no release is published.
//
// The tool is a developer's tool, never linked into metais. It is the one
// package ADR-009 Amendment A11 lets run commands, read the environment
// whole and read and write files for the site.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	os.Exit(cli(os.Args[1:], os.Stdout, os.Stderr, os.Environ()))
}

// cli parses args and builds the site, writing its report to stdout and a
// refusal to stderr. It returns the exit status: 0 when the site is
// written, 1 on a refusal, 2 on a usage error.
func cli(args []string, stdout, stderr io.Writer, environ []string) int {
	fs := flag.NewFlagSet("site", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "the checkout to build the site from, an absolute path")
	out := fs.String("out", "", "the directory to write the site into, an absolute path that does not exist")
	release := fs.String("release", "", "the release's semantic version, without a leading v")
	sums := fs.String("sums", "", "the release's SHA256SUMS file")
	data := fs.String("data", "", "a directory that does not exist, to keep the exported inputs in, an absolute path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	usage := fs.NArg() > 0 || *root == "" || *out == "" ||
		!filepath.IsAbs(*root) || !filepath.IsAbs(*out) || (*data != "" && !filepath.IsAbs(*data)) || (*release == "") != (*sums == "")
	if usage {
		fmt.Fprintln(stderr, "usage: go run ./internal/tools/site -root <checkout> -out <dir> [-release <semver> -sums <SHA256SUMS>] [-data <dir>], with absolute paths")
		return 2
	}
	b := builder{root: *root, out: *out, data: *data, version: *release, sums: *sums, environ: environ, report: stdout}
	if err := b.run(); err != nil {
		fmt.Fprintln(stderr, "site:", err)
		return 1
	}
	return 0
}
