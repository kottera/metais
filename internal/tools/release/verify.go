// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"metais.org/metais/internal/tools/release/buildline"
)

// errAssets is verify's refusal of a release's assets.
var errAssets = errors.New("the release's assets are not the release's")

// sumsName is the name of the file the builder writes beside the archives
// and a release publishes with them.
const sumsName = "SHA256SUMS"

// verifyCLI is the release command's verify subcommand: it checks the
// assets of a release downloaded into -dir (verify), writes the
// specification revisions the release's tool implements to stdout, one
// per line, and its report or a refusal to stderr. It returns the exit
// status: 0 when the assets are the release's, 1 on a refusal, 2 on a
// usage error.
func verifyCLI(args []string, stdout, stderr io.Writer) int {
	const usage = "usage: go run ./internal/tools/release verify -version <semver> -dir <assets>"
	fs := flag.NewFlagSet("release verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	version := fs.String("version", "", "the release's semantic version, without a leading v")
	dir := fs.String("dir", "", "a directory holding the release's assets and nothing else")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 || *version == "" || *dir == "" {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	implements, err := verify(*version, *dir)
	if err != nil {
		fmt.Fprintln(stderr, "release verify:", err)
		return 1
	}
	fmt.Fprintf(stderr, "release verify: %s and its %d archives are the release %s's, and its tool reports %s and implements %s\n",
		sumsName, len(buildline.Shipped), *version, *version, strings.Join(implements, ", "))
	for _, rev := range implements {
		fmt.Fprintln(stdout, rev)
	}
	return 0
}

// verify checks that dir holds the assets of the release version, as
// ADR-013 D7 item 3 has release.yml check a draft's before it publishes
// anything: SHA256SUMS and one archive per shipped target (ADR-010 D12
// item 2), and nothing else; SHA256SUMS exactly what the builder writes
// for version (parseSums); and each archive's SHA-256 its line's. It then
// runs the metais of the archive built for this host, which must report
// version, and returns the specification revisions it implements, so the
// caller can hold a release to ADR-011 D5 item 6. Running the shipped
// binary builds nothing: the bytes are the release procedure's (ADR-013
// D7 item 2).
func verify(version, dir string) ([]string, error) {
	if !releaseVersion(version) {
		return nil, fmt.Errorf("%w: %q", errVersion, version)
	}
	want := []string{sumsName}
	for _, t := range buildline.Shipped {
		want = append(want, buildline.ArchiveName(version, t))
	}
	slices.Sort(want)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var got []string
	for _, e := range entries {
		if !e.Type().IsRegular() {
			return nil, fmt.Errorf("%w: %s is not a regular file", errAssets, e.Name())
		}
		got = append(got, e.Name())
	}
	if !slices.Equal(got, want) {
		return nil, fmt.Errorf("%w: they are %q, and a release of %s has exactly %q", errAssets, got, version, want)
	}
	sums, err := os.ReadFile(filepath.Join(dir, sumsName))
	if err != nil {
		return nil, err
	}
	hashes, err := parseSums(version, sums)
	if err != nil {
		return nil, err
	}
	for _, t := range buildline.Shipped {
		name := buildline.ArchiveName(version, t)
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(b)
		if got := hex.EncodeToString(sum[:]); got != hashes[t] {
			return nil, fmt.Errorf("%w: %s's SHA-256 is %s, and its line in %s says %s", errAssets, name, got, sumsName, hashes[t])
		}
	}
	host := buildline.Target{OS: runtime.GOOS, Arch: runtime.GOARCH}
	if !slices.Contains(buildline.Shipped, host) {
		return nil, fmt.Errorf("this host, %s, is no shipped target, so it cannot run the release's tool", host)
	}
	return reportedVersion(filepath.Join(dir, buildline.ArchiveName(version, host)), version)
}

// reportedVersion runs metais version --json with the metais of archive,
// and returns the revisions it implements, refusing unless it reports
// version and implements at least one revision M.m.
func reportedVersion(archive, version string) ([]string, error) {
	bin, err := os.MkdirTemp("", "metais-verify-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(bin)
	exe := filepath.Join(bin, "metais")
	if err := extract(archive, "metais", exe); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "version", "--json")
	cmd.Env = []string{}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s's metais version --json: %v: %s", filepath.Base(archive), err, stderr.Bytes())
	}
	var doc struct {
		OK     bool `json:"ok"`
		Result struct {
			Tool struct {
				Version string `json:"version"`
			} `json:"tool"`
			Specification struct {
				Implements []string `json:"implements"`
			} `json:"specification"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &doc); err != nil || !doc.OK {
		return nil, fmt.Errorf("%s's metais version --json wrote no version: %q", filepath.Base(archive), out)
	}
	if doc.Result.Tool.Version != version {
		return nil, fmt.Errorf("%w: %s's metais reports version %q, not %s", errAssets, filepath.Base(archive), doc.Result.Tool.Version, version)
	}
	implements := doc.Result.Specification.Implements
	if len(implements) == 0 || slices.ContainsFunc(implements, func(r string) bool { return !revision(r) }) {
		return nil, fmt.Errorf("%w: %s's metais implements %q, not one or more revisions M.m", errAssets, filepath.Base(archive), implements)
	}
	return implements, nil
}

// maxBinary bounds the binary extract writes: a shipped metais is a
// fraction of it.
const maxBinary = 256 << 20

// extract writes the regular file name of the gzipped tar archive to out,
// executable, refusing one larger than maxBinary.
func extract(archive, name, out string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(archive), err)
	}
	tr := tar.NewReader(zr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("%w: %s holds no %s", errAssets, filepath.Base(archive), name)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(archive), err)
		}
		if h.Name != name || h.Typeflag != tar.TypeReg {
			continue
		}
		w, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
		if err != nil {
			return err
		}
		n, err := io.Copy(w, io.LimitReader(tr, maxBinary+1))
		if err == nil && n > maxBinary {
			err = fmt.Errorf("%w: %s's %s is larger than %d bytes", errAssets, filepath.Base(archive), name, maxBinary)
		}
		if err != nil {
			w.Close()
			return err
		}
		return w.Close()
	}
}
