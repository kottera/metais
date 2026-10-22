// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/tools/ux/scrub"
)

// scrubMax is the largest file of a session's directory a scrubbed copy
// keeps. A larger one, and any file that is not UTF-8 text such as a
// packed package, is left out: the record still lists it, with its size
// and digest.
const scrubMax = 256 << 10

// scrubSet writes a scrubbed copy of the set in c.out to dest, which must
// not exist yet: set.json, and for each run its record, transcript and
// standard error, and the text files of its work/ and tmp/, every one put
// through package scrub. The harness's own files under tmp/, the
// session's HOME, its work/.claude/, and every fixture of work/ the
// session left unchanged are left out. It is the step a scorer's fixtures are
// made by, and the one a scorecard's transcripts go through before they
// are kept.
func (c *config) scrubSet(dest string) error {
	if !filepath.IsAbs(dest) {
		return errors.New("-scrub must be an absolute path")
	}
	if _, err := os.Lstat(dest); err == nil {
		return fmt.Errorf("-scrub: %s exists; give a new directory", dest)
	}
	set := scrub.New("", c.out)
	b, err := os.ReadFile(filepath.Join(c.out, setFile))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dest, setFile), set.Bytes(b), 0o644); err != nil {
		return err
	}
	runs := os.DirFS(filepath.Join(c.out, "runs"))
	n := 0
	err = fs.WalkDir(runs, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.Name() != recordFile || d.IsDir() {
			return err
		}
		n++
		return scrubRun(filepath.Join(c.out, "runs", filepath.Dir(p)), c.out, filepath.Join(dest, "runs", filepath.Dir(p)))
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(c.stdout, "scrubbed %d runs into %s\n", n, dest)
	return nil
}

// scrubRun writes a scrubbed copy of the run in dir, of the set in
// setDir, to dest.
func scrubRun(dir, setDir, dest string) error {
	s := scrub.New(dir, setDir)
	run := os.DirFS(dir)
	// A fixture the session left as it started is the commit's or the
	// runner's, not the session's: the record lists it, and the copy
	// leaves it out, so a tree such as spec/ is not copied into every run.
	var rec runRecord
	if err := readJSON(filepath.Join(dir, recordFile), &rec); err != nil {
		return err
	}
	unchanged := map[string]bool{}
	for _, f := range rec.Files {
		if f.Change == "unchanged" {
			unchanged[workDir+"/"+f.Path] = true
		}
	}
	return fs.WalkDir(run, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		top, _, _ := strings.Cut(p, "/")
		switch {
		case d.IsDir() && (top == "home" || strings.HasPrefix(p, "tmp/claude-") || p == workDir+"/.claude"):
			// The session's HOME, the harness's own temporary files, and
			// the skill -with-skill installed, which a checkout holding the
			// copy would offer its own sessions as a skill.
			return fs.SkipDir
		case d.IsDir() || !d.Type().IsRegular():
			return nil
		case top != workDir && top != "tmp" && p != recordFile && p != transcriptFile && p != stderrFile, unchanged[p]:
			return nil
		}
		b, err := fs.ReadFile(run, p)
		if err != nil {
			return err
		}
		if (top == workDir || top == "tmp") && (len(b) > scrubMax || !utf8.Valid(b)) {
			return nil
		}
		out := filepath.Join(dest, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, s.Bytes(b), 0o644)
	})
}
