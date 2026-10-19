// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/cli"
	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/testenv"
)

// onePassVector is the SC8 vector: errors planted in several inputs,
// several fields of one input, several records of a CSV input and several
// files of a collection, with a parse error in one collection item and one
// CSV record refused alone (ADR-006's Conformance list, PRD SC8).
const onePassVector = "sup/every-finding-one-pass"

// planted is what identifies one planted finding: its code, its subject's
// name, and where it is.
type planted struct {
	Code    string `json:"code"`
	Subject struct {
		Name string `json:"name"`
	} `json:"subject"`
	Location struct {
		Origin   string `json:"origin"`
		File     string `json:"file"`
		Argument int    `json:"argument"`
		Pointer  string `json:"pointer"`
	} `json:"location"`
}

func (p planted) key() string {
	l := p.Location
	return strings.Join([]string{p.Code, p.Subject.Name, l.Origin, l.File, strconv.Itoa(l.Argument), l.Pointer}, "|")
}

// onePassStep reads the vector's one validate step: its arguments, as the
// cli adapter writes them, and the findings it expects.
func onePassStep(t *testing.T) (argv []string, want []planted) {
	t.Helper()
	dir := filepath.Join(moduleRootAbs(t), "spec", "conformance", "vectors", filepath.FromSlash(onePassVector))
	var v struct {
		Cases []struct {
			Steps []struct {
				Args struct {
					Package string   `json:"package"`
					Data    string   `json:"data"`
					Set     []string `json:"set"`
				} `json:"args"`
				Expect struct {
					Findings string `json:"findings"`
				} `json:"expect"`
			} `json:"steps"`
		} `json:"cases"`
	}
	b, err := os.ReadFile(filepath.Join(dir, "vector.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.Cases) != 1 || len(v.Cases[0].Steps) != 1 {
		t.Fatalf("%s has more than one step", onePassVector)
	}
	st := v.Cases[0].Steps[0]
	argv = []string{"validate", "--json", filepath.Join(dir, st.Args.Package), "--data", filepath.Join(dir, st.Args.Data)}
	for _, s := range st.Args.Set {
		argv = append(argv, "--set", s)
	}
	b, err = os.ReadFile(filepath.Join(dir, filepath.FromSlash(st.Expect.Findings)))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &want); err != nil {
		t.Fatal(err)
	}
	return argv, want
}

// detectOnePass is the detector: it runs validate in-process under faults
// and returns each planted finding the report does not hold.
func detectOnePass(t *testing.T, argv []string, want []planted, faults fault.Set) []string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cli.Run(context.Background(), cli.Invocation{
		Args: argv, Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
		Options: cli.Options{Faults: faults},
	})
	var doc struct {
		Findings []planted `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("%v: %q", err, stdout.String())
	}
	got := map[string]bool{}
	for _, f := range doc.Findings {
		got[f.key()] = true
	}
	var missing []string
	for _, f := range want {
		if !got[f.key()] {
			missing = append(missing, f.key())
		}
	}
	return missing
}

// TestOnePassDetector is SC8's planted-errors test (ADR-010 D13 item 4,
// W4-1.5): validate reports every planted finding of the SC8 vector in one
// run. Its control, in the same run, is the FirstFinding fault: with the
// report keeping only the first error, the detector must see planted
// findings missing, so a detector gone blind fails here.
func TestOnePassDetector(t *testing.T) {
	testenv.Home(t)
	argv, want := onePassStep(t)
	errors := 0
	for _, f := range want {
		if !strings.HasPrefix(f.Code, "input.constraints_not_evaluated") && !strings.HasPrefix(f.Code, "input.validation.partial") {
			errors++
		}
	}
	if errors < 8 {
		t.Fatalf("the SC8 vector plants %d errors, too few to prove one pass", errors)
	}
	if missing := detectOnePass(t, argv, want, 0); len(missing) > 0 {
		t.Errorf("validate did not report every planted finding: %q", missing)
	}
	if missing := detectOnePass(t, argv, want, fault.FirstFinding); len(missing) == 0 {
		t.Error("with FirstFinding, every planted finding was still reported: the detector cannot tell one pass from its absence")
	}
}
