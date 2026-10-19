// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/testenv"
)

// packedVectors are the suite's valid packed packages, each packed from a
// source directory beside it.
var packedVectors = []string{
	"pkg/opens-packed/package.metais",
	"dat/combined-schema-packed/package.metais",
	"man/canonical-form-packed/package.metais",
	"man/large-integer-packed/package.metais",
}

// inspectResult runs inspect --json on p and returns its result, without
// the package's source, which is the one member a packed package and its
// source directory differ in (ADR-008 D6 item 1).
func inspectOf(t *testing.T, p string) (map[string]any, string) {
	t.Helper()
	r := runTool(t, "inspect", "--json", p)
	if !r.doc.OK || r.status != 0 {
		t.Fatalf("metais inspect %s: %s", p, r.stdout)
	}
	var res map[string]any
	if err := json.Unmarshal(r.doc.Result, &res); err != nil {
		t.Fatal(err)
	}
	pk := res["package"].(map[string]any)
	source, _ := pk["source"].(string)
	delete(pk, "source")
	return res, source
}

// TestUnpackRoundTrip is W4-5.2, ADR-007 D8 item 5's half that needs no
// pack: unpacking a committed package gives a source directory that
// inspect reads with the same result apart from source, its package
// digest included, and the files unpack reports are the package's
// entries but .compiled/'s, in byte order.
func TestUnpackRoundTrip(t *testing.T) {
	testenv.Home(t)
	for _, v := range packedVectors {
		t.Run(v, func(t *testing.T) {
			pkg := filepath.Join(vectorsDir, filepath.FromSlash(v))
			to := filepath.Join(testenv.Dir(t), "src")
			r := runTool(t, "unpack", "--json", pkg, "--to", to)
			if !r.doc.OK || r.status != 0 {
				t.Fatalf("metais unpack: %s", r.stdout)
			}
			var res unpackResult
			if err := json.Unmarshal(r.doc.Result, &res); err != nil {
				t.Fatal(err)
			}
			var onDisk []string
			filepath.WalkDir(to, func(p string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					rel, _ := filepath.Rel(to, p)
					onDisk = append(onDisk, filepath.ToSlash(rel))
				}
				return err
			})
			slices.Sort(onDisk)
			if !slices.Equal(res.Files, onDisk) || slices.ContainsFunc(onDisk, func(f string) bool { return strings.HasPrefix(f, ".compiled/") }) {
				t.Errorf("unpack reported %q and wrote %q", res.Files, onDisk)
			}
			want, ws := inspectOf(t, pkg)
			got, gs := inspectOf(t, to)
			if ws != "packed" || gs != "directory" {
				t.Errorf("sources %s and %s", ws, gs)
			}
			a, _ := json.Marshal(want)
			b, _ := json.Marshal(got)
			if !bytes.Equal(a, b) {
				t.Errorf("inspect of the package and of its unpacked source differ apart from source:\n%s\n%s", a, b)
			}
			if want["package"].(map[string]any)["digest"] == nil {
				t.Error("inspect reported no digest")
			}
		})
	}
}

// TestUnpackOutput is unpack's result in each mode: the files written,
// and check as the next step once metais has it.
func TestUnpackOutput(t *testing.T) {
	testenv.Home(t)
	pkg := filepath.Join(vectorsDir, "pkg/opens-packed/package.metais")
	r := runTool(t, "unpack", pkg, "--to", filepath.Join(testenv.Dir(t), "src"))
	if r.status != 0 || !strings.HasPrefix(r.stdout, "Wrote 13 files:\n  assets/static/logo.txt\n") {
		t.Errorf("metais unpack: status %d, stdout %q, stderr %q", r.status, r.stdout, r.stderr)
	}
	wantNext := lookup(table, "check") != nil
	r = runTool(t, "unpack", "--json", pkg, "--to", filepath.Join(testenv.Dir(t), "src"))
	if got := len(r.doc.Next) == 1 && r.doc.Next[0].Run == "metais check <dir>"; got != wantNext {
		t.Errorf("next %v, with check declared %v", r.doc.Next, wantNext)
	}
}

// TestUnpackRefusals is every refusal unpack's own: R2-105 and R2-136, a
// source directory, and standard input. Each writes nothing.
func TestUnpackRefusals(t *testing.T) {
	testenv.Home(t)
	pkg := filepath.Join(vectorsDir, "pkg/opens-packed/package.metais")
	root := testenv.Dir(t)
	full := filepath.Join(root, "full")
	if err := os.MkdirAll(filepath.Join(full, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(full, "notes.txt"), []byte("theirs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("theirs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name   string
		args   []string
		code   finding.Code
		status int
		params string
	}{
		{"no --to", []string{"unpack", "--json", pkg}, finding.CodeUsageMissingOperand, finding.ExitUsage,
			`{"command":"unpack","operand":"--to <dir>"}`},
		{"an empty --to", []string{"unpack", "--json", pkg, "--to="}, finding.CodeUsageBadValue, finding.ExitUsage,
			`{"argument":4,"flag":"--to","reason":"value_not_allowed"}`},
		{"standard input", []string{"unpack", "--json", "-", "--to", "x"}, finding.CodeUsagePackageFromStdin, finding.ExitUsage,
			`{"argument":3}`},
		{"a source directory", []string{"unpack", "--json", filepath.Join(vectorsDir, "pkg/opens-source/package"), "--to", filepath.Join(root, "x")},
			finding.CodeUsageNotPackedPackage, finding.ExitUsage, `{"argument":3}`},
		{"a non-empty directory", []string{"unpack", "--json", pkg, "--to", full}, finding.CodeDestinationNotEmpty, finding.ExitDestination,
			`{"entries":["notes.txt"],"more":false}`},
		{"a file", []string{"unpack", "--json", pkg, "--to", file}, finding.CodeDestinationNotDirectory, finding.ExitDestination, `{}`},
		{"beneath a file", []string{"--json", "unpack", pkg, "--to", filepath.Join(file, "src")}, finding.CodeDestinationNotDirectory, finding.ExitDestination, `{}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := snapshot(t, root)
			r := runTool(t, c.args...)
			if r.status != c.status || !slices.Equal(r.codes(), []string{string(c.code)}) {
				t.Fatalf("status %d, %s", r.status, r.stdout)
			}
			var want any
			if err := json.Unmarshal([]byte(c.params), &want); err != nil {
				t.Fatal(err)
			}
			params, _ := json.Marshal(r.doc.Findings[0].Params)
			canon, _ := json.Marshal(want)
			if !bytes.Equal(params, canon) {
				t.Errorf("params %s, want %s", params, c.params)
			}
			if after := snapshot(t, root); !slices.Equal(after, before) {
				t.Errorf("the refusal wrote: %q, was %q", after, before)
			}
		})
	}
}

// TestUnpackRefusesAPackageAsTheOpenDoes is W4-5.5 for a source directory:
// unpack opens a package before it asks which form it is, so a source
// directory every opening command refuses is refused here with the same
// findings, not as a source directory, and nothing is written. The exit
// status is the authoring role's (ADR-006 D12 item 2): a package's own
// content error is content under unpack, as under check.
func TestUnpackRefusesAPackageAsTheOpenDoes(t *testing.T) {
	testenv.Home(t)
	for _, v := range []string{"pkg/source-compiled-in-source/package", "dat/schema-ref-escapes/package", "pkg/digest-mismatch/package.metais"} {
		p := filepath.Join(vectorsDir, filepath.FromSlash(v))
		to := filepath.Join(testenv.Dir(t), "src")
		want := runTool(t, "inspect", "--json", p)
		got := runTool(t, "unpack", "--json", p, "--to", to)
		if got.doc.OK || !slices.Equal(got.codes(), want.codes()) || got.status == 0 {
			t.Errorf("%s: unpack refused %q with %d; inspect %q with %d", v, got.codes(), got.status, want.codes(), want.status)
		}
		if _, err := os.Lstat(to); !os.IsNotExist(err) {
			t.Errorf("%s: unpack wrote its target", v)
		}
	}
}

// snapshot is every path beneath root with its size, in byte order.
func snapshot(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, _ := d.Info()
		rel, _ := filepath.Rel(root, p)
		out = append(out, filepath.ToSlash(rel)+" "+info.Mode().String())
		return nil
	})
	return out
}

// TestUnpackRefusalsTriggered is ADR-008 D16 item 2 for unpack: every
// refusal it declares is triggered by one of unpack's tests here, by
// inspect's over the suite, which unpack shares through the open, or by a
// line refusal, or is a code the unwired list says no vector produces
// yet. The line's and the open's are proved by inspect's test.
func TestUnpackRefusalsTriggered(t *testing.T) {
	own := map[finding.Code]bool{
		finding.CodeUsageMissingOperand: true, finding.CodeUsagePackageFromStdin: true, finding.CodeUsageBadValue: true,
		finding.CodeUsageNotPackedPackage: true, finding.CodeDestinationNotEmpty: true,
		finding.CodeDestinationNotDirectory: true, finding.CodeDestinationWriteFailed: true,
	}
	inspect := lookup(table, "inspect").Refusals
	for _, rf := range lookup(table, "unpack").Refusals {
		if !own[rf.Code] && !slices.ContainsFunc(inspect, func(i Refusal) bool { return i.Code == rf.Code }) {
			t.Errorf("unpack declares %s, which neither its own tests nor inspect's trigger", rf.Code)
		}
	}
}
