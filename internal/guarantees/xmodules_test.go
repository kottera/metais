// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"slices"
	"strings"
	"testing"
)

// TestWindowsLinksNoXModules holds ADR-010 Amendment A1 (proposed), which
// the wave 2 brief carries as the stricter rule: built for windows, no
// package of the module links golang.org/x/sys or golang.org/x/term,
// because golang.org/x/sys/windows imports net and ADR-009 D16 item 3
// keeps net out of the binary on every target. internal/hostfs's
// no-replace rename is syscall.MoveFile there, and internal/hostenv's
// terminal check syscall.GetConsoleMode.
//
// It reads every package of the module, not only cmd/metais's closure,
// so a package is held to the rule before it is wired into the binary.
func TestWindowsLinksNoXModules(t *testing.T) {
	root := moduleRootAbs(t)
	banned := func(dep string) bool {
		return dep == "golang.org/x/sys" || strings.HasPrefix(dep, "golang.org/x/sys/") ||
			dep == "golang.org/x/term" || strings.HasPrefix(dep, "golang.org/x/term/")
	}
	for _, tg := range targets {
		pkgs := cachedList(t, root, tg, "./...")
		if tg.goos != "windows" {
			// The control: on unix hostfs links golang.org/x/sys/unix for
			// its rename, so a scan that reads no dependencies fails here.
			i := slices.IndexFunc(pkgs, func(p *listedPackage) bool {
				return p.ImportPath == modulePath+"/internal/hostfs"
			})
			if i < 0 || !slices.Contains(pkgs[i].Deps, "golang.org/x/sys/unix") {
				t.Errorf("%s: internal/hostfs does not link golang.org/x/sys/unix; the scan proves nothing", tg)
			}
			continue
		}
		for _, p := range pkgs {
			if !p.inModule(modulePath) {
				continue
			}
			for _, d := range p.Deps {
				if banned(d) {
					t.Errorf("%s: %s links %s", tg, p.ImportPath, d)
				}
			}
		}
	}
}
