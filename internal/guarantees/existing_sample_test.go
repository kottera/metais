// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"metais.org/metais/internal/cli"
	"metais.org/metais/internal/fault"
)

// sampleFiles are the files metais sample writes for the SMP vectors'
// package with --items 2, the paths the detector seeds every other one of.
var sampleFiles = []string{"events/001.yaml", "events/002.yaml", "lines.csv", "summary.txt", "vendor.json"}

func init() {
	existingWrites = append(existingWrites, existingWrite{
		name:  "metais sample --out --overwrite",
		files: sampleFiles,
		run:   func(t *testing.T, out string, faults fault.Set) error { return runSampleOverwrite(t, out, faults, "2") },
	}, existingWrite{
		name:  "metais sample --out --overwrite --items 0",
		files: sampleFiles[2:],
		dirs:  []string{"events"},
		run:   func(t *testing.T, out string, faults fault.Set) error { return runSampleOverwrite(t, out, faults, "0") },
	})
}

// runSampleOverwrite runs metais sample --out out --overwrite --items items
// in-process, under faults, over the SMP vectors' package (W5-10.2); with
// --items 0 its collection is an empty directory (W5-V2.A3).
func runSampleOverwrite(t *testing.T, out string, faults fault.Set, items string) error {
	t.Helper()
	pkg := filepath.Join(moduleRootAbs(t), "spec", "conformance", "vectors", "smp", "sample-data-directory", "package")
	var stdout, stderr bytes.Buffer
	status := cli.Run(context.Background(), cli.Invocation{
		Args:    []string{"sample", pkg, "--out", out, "--overwrite", "--items", items, "--json"},
		Stdout:  &stdout,
		Stderr:  &stderr,
		Options: cli.Options{Faults: faults},
	})
	if status != 0 {
		return fmt.Errorf("metais sample exited %d: %s%s", status, stdout.String(), stderr.String())
	}
	return nil
}
