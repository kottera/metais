// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/kottera/metais/internal/cli"
	"github.com/kottera/metais/internal/fault"
)

// sampleFiles are the files metais sample writes for the SMP vectors'
// package with --items 2, the paths the detector seeds every other one of.
var sampleFiles = []string{"events/001.yaml", "events/002.yaml", "lines.csv", "summary.txt", "vendor.json"}

func init() {
	existingWrites = append(existingWrites, existingWrite{
		name:  "metais sample --to --overwrite",
		files: sampleFiles,
		run:   runSampleOverwrite,
	})
}

// runSampleOverwrite runs metais sample --to out --overwrite in-process,
// under faults, over the SMP vectors' package (W5-10.2).
func runSampleOverwrite(t *testing.T, out string, faults fault.Set) error {
	t.Helper()
	pkg := filepath.Join(moduleRootAbs(t), "spec", "conformance", "vectors", "smp", "sample-data-directory", "package")
	var stdout, stderr bytes.Buffer
	status := cli.Run(context.Background(), cli.Invocation{
		Args:    []string{"sample", pkg, "--to", out, "--overwrite", "--items", "2", "--json"},
		Stdout:  &stdout,
		Stderr:  &stderr,
		Options: cli.Options{Faults: faults},
	})
	if status != 0 {
		return fmt.Errorf("metais sample exited %d: %s%s", status, stdout.String(), stderr.String())
	}
	return nil
}
