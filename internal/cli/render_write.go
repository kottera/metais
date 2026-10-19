// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	pathpkg "path"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/tarw"
	"github.com/kottera/metais/internal/zipfmt"
)

// writeArchive writes plan, in path order, to w as ADR-005 D17's archive,
// built from the rendered bytes and never from disk (item 6). A failed
// write to w ends it unreported, as every write to standard output is
// (ADR-006 D12 item 5); any other error is a defect.
func writeArchive(w io.Writer, plan []*planned) {
	ew := &stickyWriter{w: w}
	tw := tarw.NewWriter(ew)
	for _, pf := range plan {
		size, _ := pf.measured()
		r, done := content(pf)
		err := tw.Add(pf.path, pf.decl.Executable, size, r)
		if derr := done(err); err == nil && derr != nil {
			panic("cli: a measured container does not write: " + derr.Error())
		}
		if err != nil {
			if ew.err != nil {
				return
			}
			panic("cli: " + err.Error())
		}
	}
	if err := tw.Close(); err != nil && ew.err == nil {
		panic("cli: " + err.Error())
	}
}

// writeContent writes pf's bytes alone to w (ADR-005 D18 item 1): an
// asset's as the package holds them, and a container's zip as it is
// assembled. A failed write ends it unreported.
func writeContent(w io.Writer, pf *planned) {
	ew := &stickyWriter{w: w}
	r, done := content(pf)
	_, err := io.Copy(ew, r)
	if derr := done(err); err == nil && derr != nil {
		panic("cli: a measured container does not write: " + derr.Error())
	}
	if err != nil && ew.err == nil {
		panic("cli: " + err.Error())
	}
}

// stickyWriter is w that keeps the first error a write returns.
type stickyWriter struct {
	w   io.Writer
	err error
}

func (sw *stickyWriter) Write(b []byte) (int, error) {
	if sw.err != nil {
		return 0, sw.err
	}
	n, err := sw.w.Write(b)
	sw.err = err
	return n, err
}

// writeContainer writes the zip of the container pf to w, assembling it as
// it is written (ADR-005 D8, D13 item 5): each entry's content is its
// rendered bytes, or the zip of a container entry, itself written as it
// is written. stored: true and a profile's mimetype are stored; every
// other entry is ADR-001 D4's auto (D8 item 7).
func writeContainer(w io.Writer, pf *planned) ([]zipfmt.Written, error) {
	parts := make([]zipfmt.Part, len(pf.entries))
	for i, e := range pf.entries {
		method := zipfmt.Auto
		if e.tool || e.decl.Stored {
			method = zipfmt.Store
		}
		parts[i] = zipfmt.Part{Name: e.path, Method: method, Content: func(w io.Writer) error {
			if e.isContainer() {
				_, err := writeContainer(w, e)
				return err
			}
			r, done := content(e)
			_, err := io.Copy(w, r)
			return done(err)
		}}
	}
	return zipfmt.WriteParts(w, parts)
}

// content is the reader of pf's bytes as they are written: an asset's are
// streamed from the package, its digest taken on the first whole read; a
// container's zip is written into a pipe as it is read; and done reports
// the error of its writing, once the reader is finished with.
func content(pf *planned) (r io.Reader, done func(read error) error) {
	if pf.asset != "" {
		rc, err := pf.tree.Open(pf.asset)
		if err != nil {
			panic("cli: opening " + pf.asset + ", which the plan found: " + err.Error())
		}
		h := sha256.New()
		return io.TeeReader(rc, h), func(read error) error {
			rc.Close()
			if read == nil && pf.digest == "" {
				pf.digest = "sha256:" + hex.EncodeToString(h.Sum(nil))
			}
			return nil
		}
	}
	if !pf.isContainer() {
		return bytes.NewReader(pf.bytes), func(error) error { return nil }
	}
	pr, pw := io.Pipe()
	errc := make(chan error, 1)
	go func() {
		_, err := writeContainer(pw, pf)
		pw.CloseWithError(err)
		errc <- err
	}()
	return pr, func(read error) error {
		// A reader that stopped early unblocks the writer.
		pr.CloseWithError(read)
		if err := <-errc; err != nil && read == nil {
			return err
		}
		return nil
	}
}

// commit writes plan, in path order, to out (ADR-005 D14): into dest,
// the existing directory phase 3 checked, by commitExisting; otherwise
// beneath a staging directory beside out, renamed to out (item 1). A
// failure removes what the render created and reports
// output.destination.nothing_written.
func commit(out string, dest *destination, plan []*planned, s *session) (finding.Finding, bool) {
	if dest != nil {
		return commitExisting(dest, plan)
	}
	o, err := hostfs.StageNew(out, hostfs.WithFaults(s.inv.Options.Faults))
	if err != nil {
		if hostfs.ReasonOf(err) == hostfs.NotDirectory {
			return finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil), false
		}
		return nothingWritten(hostfs.ReasonOf(err)), false
	}
	if f, ok := prepare(o, plan); !ok {
		return f, false
	}
	if err := o.Commit(); err != nil {
		return nothingWritten(hostfs.ReasonOf(err)), false
	}
	return finding.Finding{}, true
}

// commitExisting writes plan into dest under ADR-005 D14 item 2, as
// Amendment A1 amends it: every file is prepared beneath the one staging
// directory inside it (item 1), which a leftover refuses with
// output.destination.staging_exists, and is committed by unit (item 2),
// every check made again first. A failure before the first unit is renamed
// leaves the directory as it was, and is output.destination.nothing_written;
// one after is
// output.destination.commit_failed, listing the paths already committed
// (item 3).
func commitExisting(dest *destination, plan []*planned) (finding.Finding, bool) {
	defer dest.close()
	// --only writes the one file selected, by the same rules (ADR-005 D18
	// item 3); every planned path was checked all the same, so a selected
	// render is refused exactly when the whole one is (item 2).
	write := map[string]bool{}
	for _, pf := range plan {
		write[pf.path] = true
	}
	var checks []hostfs.PathCheck
	for _, c := range dest.checks {
		if write[c.Path] {
			checks = append(checks, c)
		}
	}
	o, err := dest.x.Stage(checks)
	if err != nil {
		return stageFailed(err), false
	}
	if f, ok := prepare(o, plan); !ok {
		return f, false
	}
	err = o.Commit()
	if err == nil {
		return finding.Finding{}, true
	}
	var ce *hostfs.CommitError
	if !errors.As(err, &ce) || len(ce.Committed) == 0 {
		return nothingWritten(hostfs.ReasonOf(err)), false
	}
	return finding.New(finding.CodeDestinationCommitFailed, invocation, nil, finding.Params{
		"reason":    finding.String(writeReason(hostfs.ReasonOf(err))),
		"committed": stringList(ce.Committed),
	}), false
}

// prepare writes every file of plan through o, in path order, with every
// directory it needs. A failure aborts o, which removes what it wrote.
func prepare(o *hostfs.Output, plan []*planned) (finding.Finding, bool) {
	for _, pf := range plan {
		err := o.MkdirAll(pathpkg.Dir(pf.path))
		if err == nil {
			r, done := content(pf)
			err = o.Create(pf.path, pf.decl.Executable, r)
			if derr := done(err); err == nil && derr != nil {
				panic("cli: a measured container does not write: " + derr.Error())
			}
		}
		if err != nil {
			o.Abort()
			return nothingWritten(hostfs.ReasonOf(err)), false
		}
	}
	return finding.Finding{}, true
}
