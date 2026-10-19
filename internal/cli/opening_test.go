// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"slices"
	"testing"
)

// TestOpeningSetAsTheRunnerReadsIt is where the command line meets the
// conformance runner (W2-1, W2-2). The runner never links this package: it
// reads the commands that open a package from metais help --json, as each
// result.commands[] member whose opens_package is true, in the document's
// order, and runs a hostile vector's open step through each
// (internal/conformance's cliAdapter.readOpening; ADR-011 D16 item 3).
// That reading gives exactly the set the declarations mark. It is empty
// this wave.
func TestOpeningSetAsTheRunnerReadsIt(t *testing.T) {
	r := runTool(t, "help", "--json")
	var res struct {
		Commands []struct {
			Name         string `json:"name"`
			OpensPackage bool   `json:"opens_package"`
		} `json:"commands"`
	}
	if r.doc == nil {
		t.Fatalf("help --json printed no document: %q", r.stdout)
	}
	if err := json.Unmarshal(r.doc.Result, &res); err != nil {
		t.Fatal(err)
	}
	var read []string
	for _, c := range res.Commands {
		if c.OpensPackage {
			read = append(read, c.Name)
		}
	}
	if want := OpeningCommands(); !slices.Equal(read, want) {
		t.Errorf("the runner reads the opening set %q from help --json; the declarations give %q", read, want)
	}
	if len(res.Commands) != len(table) {
		t.Errorf("help --json lists %d commands; the table declares %d", len(res.Commands), len(table))
	}
}
