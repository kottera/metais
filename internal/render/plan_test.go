// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/testenv"
	"github.com/kottera/metais/internal/zipfmt"
)

// W5-6.2, MTS-TRN-007: a container is planned with at most
// zipfmt.MaxEntries entries, 65,534, since a zip without zip64 holds no
// more: exactly that many are accepted, and one more is refused with
// output.container.too_many_entries. Both are planned only, never
// rendered, so the boundary is proved without rendering 65,534 templates;
// trn/container-too-many-entries holds the refusal end to end. The
// planning is bounded work, held to a CPU time.
func TestContainerEntryBoundary(t *testing.T) {
	const cpuBound = 20 * time.Second // about half a second, two under the race detector
	for _, n := range []int{zipfmt.MaxEntries, zipfmt.MaxEntries + 1} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			root := testenv.Dir(t)
			dir := writeSourcePackage(t, root, "  - id: box\n    path: box.zip\n    container:\n      entries:\n"+
				"        - template: e.txt.liquid\n          for_each: (1.."+strconv.Itoa(n)+")\n          path: \"{{ item }}\"\n",
				map[string]string{"templates/e.txt.liquid": ""})
			pkg, fs := loader.Open(dir, loader.Options{Limits: limits.Defaults()}, 0)
			if pkg == nil {
				t.Fatalf("the package does not open: %v", fs)
			}
			defer pkg.Close()
			lim := limits.Defaults()
			ts := template.NewTemplates(pkg.Tree, lim, pkg.Manifest.Inputs)
			b, err := template.NewBindings(template.Package{
				Tree: pkg.Tree, Manifest: pkg.Manifest, ManifestValue: pkg.ManifestValue,
				Digest: pkg.Digest, Combined: pkg.CombinedValue,
			}, nil, "", lim)
			if err != nil {
				t.Fatal(err)
			}
			r := template.NewRenderer(ts, b, lim, 0)
			before, measured := selfCPU()
			plan, fs := PlanOutputs(pkg, lim, 0, r)
			if after, _ := selfCPU(); measured {
				t.Logf("planning %d entries took %v of CPU", n, after-before)
				if after-before > cpuBound {
					t.Errorf("planning %d entries took %v of CPU, more than %v", n, after-before, cpuBound)
				}
			}
			if len(plan) != 1 || len(plan[0].entries) != n {
				t.Fatalf("planned %d outputs, want one container of %d entries", len(plan), n)
			}
			codes := make([]finding.Code, len(fs))
			for i, f := range fs {
				codes[i] = f.Code()
			}
			var want []finding.Code
			if n > zipfmt.MaxEntries {
				want = []finding.Code{finding.CodeContainerTooManyEntries}
			}
			if !slices.Equal(codes, want) {
				t.Fatalf("planning %d entries: findings %v, want %v", n, codes, want)
			}
		})
	}
}
