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
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/kottera/metais/internal/testenv"
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

// TestRecordAndCompare writes the cross-platform record with
// -conformance.record from two runs of the fixture suite against the clean
// fake, and compares them with -conformance.compare on the host (ADR-011
// D15, ADR-010 D14 item 3). The records carry the tool the fake's version
// --json reports, agree, and hold no sandbox path; a record of another
// tool version or another suite is refused, and one differing in a field
// fails.
func TestRecordAndCompare(t *testing.T) {
	t.Parallel()
	suite := writeFixture(t, testenv.Dir(t))
	fake := fakeProgram(t, testenv.Dir(t), "clean")
	dir := testenv.Dir(t)
	conformance := func(args ...string) ([]byte, error) {
		cmd := exec.Command(testBinary(t), append([]string{"-test.run=^TestConformance$", "-test.count=1",
			"-conformance.suite=" + suite}, args...)...)
		return cmd.CombinedOutput()
	}
	recA, recB := filepath.Join(dir, "a.json"), filepath.Join(dir, "b.json")
	var wg sync.WaitGroup
	for _, p := range []string{recA, recB} {
		wg.Go(func() {
			if out, err := conformance("-conformance.tool="+fake, "-conformance.record="+p); err != nil {
				t.Errorf("the run recording %s: %v\n%s", filepath.Base(p), err, out)
			}
		})
	}
	wg.Wait()
	if t.Failed() {
		return
	}
	set, err := suiteSchemas(suite)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := readRecord(recA, set.validator(recordSchema))
	if err != nil {
		t.Fatal(err)
	}
	if rec.Tool != fakeTool {
		t.Errorf("the record's tool is %+v, want what version --json reports, %+v", rec.Tool, fakeTool)
	}
	if want := runtime.GOOS + "/" + runtime.GOARCH; rec.Target != want {
		t.Errorf("the record's target is %q, want %q", rec.Target, want)
	}
	if sum, _ := suiteDigest(suite); rec.Suite != sum {
		t.Errorf("the record's suite digest is %s, want %s", rec.Suite, sum)
	}
	envs := map[string]bool{}
	for _, s := range rec.Steps {
		envs[s.Env] = true
	}
	if len(rec.Steps) == 0 || !envs["A"] || !envs["B"] {
		t.Errorf("the record holds %d steps over the environments %v, want steps in A and B", len(rec.Steps), envs)
	}
	raw, err := os.ReadFile(recA)
	if err != nil {
		t.Fatal(err)
	}
	if src := sentinelAt(raw); src != "" || bytes.Contains(raw, []byte(dir)) {
		t.Errorf("the record holds a sandbox path")
	}

	if out, err := conformance("-conformance.compare=" + recA + "," + recB); err != nil {
		t.Errorf("two records of one build on one host do not compare equal: %v\n%s", err, out)
	}
	for _, c := range []struct {
		name, want string
		edit       func(map[string]any)
	}{
		{"another tool version", "records refused", func(m map[string]any) { m["tool"].(map[string]any)["version"] = "9.9.9" }},
		{"another suite", "records refused", func(m map[string]any) { m["suite_sha256"] = strings.Repeat("0", 64) }},
		{"a field", "differs", func(m map[string]any) {
			m["steps"].([]any)[0].(map[string]any)["stdout"].(map[string]any)["length"] = 1 << 20
		}},
	} {
		p := filepath.Join(dir, "edited.json")
		writeEdited(t, raw, p, c.edit)
		out, err := conformance("-conformance.compare=" + recA + "," + p)
		if err == nil || !strings.Contains(string(out), c.want) {
			t.Errorf("%s: the comparison gave %v, want a failure saying %q\n%s", c.name, err, c.want, out)
		}
	}
}

// writeEdited writes the record raw to p with edit applied.
func writeEdited(t *testing.T, raw []byte, p string, edit func(map[string]any)) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	edit(m)
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCompareRecordsEveryField proves the comparison fails on a
// difference in any field of a step, and on nothing but the target.
func TestCompareRecordsEveryField(t *testing.T) {
	b := blob{SHA256: strings.Repeat("a", 64), Length: 3}
	base := func() *record {
		return &record{Schema: recordFormat, Tool: fakeTool, Target: "darwin/arm64", Suite: strings.Repeat("b", 64),
			Steps: []recordRow{{Vector: "fix/v", Case: "c", Step: 1, Invocation: 1, Env: "A", OK: true,
				Stdout: b, Stderr: b, Stream: &b, Listing: &b,
				Files: []recordFile{{Path: "out/a.txt", Size: 3, SHA256: b.SHA256}}}}}
	}
	names := []string{"darwin/arm64", "linux/amd64"}
	other := base()
	other.Target = "linux/amd64"
	if errs := compareRecords(names, []*record{base(), other}); len(errs) > 0 {
		t.Errorf("records differing only in their target: %v", errs)
	}
	c := blob{SHA256: strings.Repeat("c", 64), Length: 4}
	for field, edit := range map[string]func(*recordRow){
		"vector":     func(r *recordRow) { r.Vector = "fix/w" },
		"case":       func(r *recordRow) { r.Case = "d" },
		"step":       func(r *recordRow) { r.Step = 2 },
		"invocation": func(r *recordRow) { r.Invocation = 2 },
		"env":        func(r *recordRow) { r.Env = "B" },
		"ok":         func(r *recordRow) { r.OK = false },
		"stdout":     func(r *recordRow) { r.Stdout = c },
		"stderr":     func(r *recordRow) { r.Stderr = c },
		"stream":     func(r *recordRow) { r.Stream = nil },
		"listing":    func(r *recordRow) { r.Listing = &c },
		"files":      func(r *recordRow) { r.Files[0].Executable = true },
	} {
		other := base()
		edit(&other.Steps[0])
		errs := compareRecords(names, []*record{base(), other})
		if len(errs) != 1 || !strings.Contains(errs[0].Error(), field+" differs") {
			t.Errorf("records differing in %s: %v", field, errs)
		}
	}
	extra := base()
	extra.Steps = append(extra.Steps, extra.Steps[0])
	if errs := compareRecords(names, []*record{base(), extra}); len(errs) == 0 {
		t.Error("records of different lengths compare equal")
	}
	for name, edit := range map[string]func(*record){
		"tool name":    func(r *record) { r.Tool.Name = "other" },
		"tool version": func(r *record) { r.Tool.Version = "1.0.0" },
		"suite":        func(r *record) { r.Suite = strings.Repeat("c", 64) },
	} {
		other := base()
		edit(other)
		errs := compareRecords(names, []*record{base(), other})
		if len(errs) == 0 || !errors.Is(errs[0], errRefused) {
			t.Errorf("records of another %s are not refused: %v", name, errs)
		}
	}
}
