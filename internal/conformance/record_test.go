// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
)

// recordFormat names the record's format, the $id of its schema.
const recordFormat = "https://metais.org/schemas/conformance/1/record.schema.json"

// record is the cross-platform record of ADR-011 D15 item 3: the tool, the
// target, the suite's digest, and what every step of every case gave in
// each environment. It holds no sandbox path.
type record struct {
	Schema string      `json:"$schema"`
	Tool   toolID      `json:"tool"`
	Target string      `json:"target"`
	Suite  string      `json:"suite_sha256"`
	Steps  []recordRow `json:"steps"`
}

// recordRow is one invocation of one step in one environment.
type recordRow struct {
	Vector     string       `json:"vector"`
	Case       string       `json:"case"`
	Step       int          `json:"step"`
	Invocation int          `json:"invocation"`
	Env        string       `json:"env"`
	OK         bool         `json:"ok"`
	Stdout     blob         `json:"stdout"`
	Stderr     blob         `json:"stderr"`
	Stream     *blob        `json:"stream"`
	Listing    *blob        `json:"listing"`
	Files      []recordFile `json:"files"`
}

// blob is the SHA-256 and length of some bytes.
type blob struct {
	SHA256 string `json:"sha256"`
	Length int    `json:"length"`
}

// recordFile is one file of a declared tree.
type recordFile struct {
	Path       string `json:"path"`
	Size       int    `json:"size"`
	SHA256     string `json:"sha256"`
	Executable bool   `json:"executable"`
}

func blobOf(b []byte) blob { return blob{SHA256: digest(b), Length: len(b)} }

// recorder gathers the rows of a run, from parallel vectors, and writes
// the record when the run ends.
type recorder struct {
	path string
	mu   sync.Mutex
	rows []recordRow
}

// add records every step of a case's runs.
func (rc *recorder) add(v *vector, c *vcase, res *caseResult) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	for _, e := range []env{envA, envB} {
		run := res.runs[e]
		if run == nil {
			continue
		}
		for si, sr := range run.steps {
			var files []recordFile
			for _, root := range sortedKeys(sr.trees) {
				t := sr.trees[root]
				for _, n := range sortedKeys(t.files) {
					f := t.files[n]
					files = append(files, recordFile{Path: root + "/" + n, Size: len(f.content), SHA256: digest(f.content), Executable: f.exec})
				}
			}
			for ii, inv := range sr.invs {
				row := recordRow{
					Vector: v.ID, Case: c.Name, Step: si + 1, Invocation: ii + 1, Env: string(e),
					OK: inv.ok, Stdout: blobOf(inv.stdout), Stderr: blobOf(inv.stderr), Files: files,
				}
				if inv.stream != nil {
					b := blobOf(inv.stream)
					row.Stream = &b
				}
				if inv.listing != nil {
					b := blobOf(marshal(inv.listing))
					row.Listing = &b
				}
				if row.Files == nil {
					row.Files = []recordFile{}
				}
				rc.rows = append(rc.rows, row)
			}
		}
	}
}

// write writes the record for r's run.
func (rc *recorder) write(r *runner) error {
	id, err := r.identity()
	if err != nil {
		return fmt.Errorf("the tool's identity: %w", err)
	}
	sum, err := suiteDigest(r.suite)
	if err != nil {
		return err
	}
	rc.mu.Lock()
	rows := slices.Clone(rc.rows)
	rc.mu.Unlock()
	slices.SortFunc(rows, func(a, b recordRow) int {
		return cmp.Or(cmp.Compare(a.Vector, b.Vector), cmp.Compare(a.Case, b.Case),
			cmp.Compare(a.Step, b.Step), cmp.Compare(a.Invocation, b.Invocation), cmp.Compare(a.Env, b.Env))
	})
	if rows == nil {
		rows = []recordRow{}
	}
	rec := record{Schema: recordFormat, Tool: id, Target: runtime.GOOS + "/" + runtime.GOARCH, Suite: sum, Steps: rows}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(rc.path, append(b, '\n'), 0o644)
}

// suiteDigest is the SHA-256 of the suite tree: of the lines
// "<path> <sha256>\n" of every regular file below suite, by path, with
// paths relative to suite and slash-separated.
func suiteDigest(suite string) (string, error) {
	var lines []string
	err := filepath.WalkDir(suite, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(suite, p)
		if err != nil {
			return err
		}
		sum, err := fileDigest(p)
		if err != nil {
			return err
		}
		lines = append(lines, filepath.ToSlash(rel)+" "+sum+"\n")
		return nil
	})
	if err != nil {
		return "", err
	}
	slices.Sort(lines)
	var b bytes.Buffer
	for _, l := range lines {
		b.WriteString(l)
	}
	return digest(b.Bytes()), nil
}

// errRefused marks records compareRecords will not compare.
var errRefused = errors.New("records refused")

// compareRecords compares records pairwise (ADR-011 D15 item 4). Records
// of different tools or tool versions, or of different suites, are
// refused; otherwise every difference in any field but the target is a
// failure, named by the first record that differs from the first.
func compareRecords(names []string, recs []*record) []error {
	var errs []error
	first := recs[0]
	for i, r := range recs[1:] {
		if r.Tool != first.Tool {
			errs = append(errs, fmt.Errorf("%w: %s is of %s %s, %s of %s %s", errRefused,
				names[0], first.Tool.Name, first.Tool.Version, names[i+1], r.Tool.Name, r.Tool.Version))
		}
		if r.Suite != first.Suite {
			errs = append(errs, fmt.Errorf("%w: %s and %s ran different suites (%s, %s)", errRefused,
				names[0], names[i+1], first.Suite, r.Suite))
		}
	}
	if len(errs) > 0 {
		return errs
	}
	for i := range recs {
		for j := i + 1; j < len(recs); j++ {
			if d := recordDifference(recs[i], recs[j]); d != "" {
				errs = append(errs, fmt.Errorf("%s (%s) and %s (%s): %s", names[i], recs[i].Target, names[j], recs[j].Target, d))
			}
		}
	}
	return errs
}

// recordDifference names the first field in which a and b differ, the
// target aside, or "".
func recordDifference(a, b *record) string {
	if a.Schema != b.Schema {
		return fmt.Sprintf("$schema: %q and %q", a.Schema, b.Schema)
	}
	if len(a.Steps) != len(b.Steps) {
		return fmt.Sprintf("%d and %d steps", len(a.Steps), len(b.Steps))
	}
	for i := range a.Steps {
		x, y := a.Steps[i], b.Steps[i]
		xb, yb := marshal(x), marshal(y)
		if bytes.Equal(xb, yb) {
			continue
		}
		var xm, ym map[string]json.RawMessage
		_ = json.Unmarshal(xb, &xm)
		_ = json.Unmarshal(yb, &ym)
		for _, k := range sortedKeys(xm) {
			if !bytes.Equal(xm[k], ym[k]) {
				return fmt.Sprintf("%s/%s step %d invocation %d env %s: %s differs: %s and %s",
					x.Vector, x.Case, x.Step, x.Invocation, x.Env, k, xm[k], ym[k])
			}
		}
	}
	return ""
}

// readRecord reads and validates one record.
func readRecord(path string, validate func([]byte) error) (*record, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := validate(b); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var rec record
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rec); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &rec, nil
}
