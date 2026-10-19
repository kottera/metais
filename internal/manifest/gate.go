// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/version"
)

// Spec is a specification version, MAJOR.MINOR (ADR-002 D7 item 1).
type Spec struct {
	Major, Minor int
}

// String returns the version as the manifest writes it, "1.0".
func (s Spec) String() string { return strconv.Itoa(s.Major) + "." + strconv.Itoa(s.Minor) }

// ParseSpec returns the version text names, which must match D7 item 1's
// grammar: two parts, each 0 or a decimal of at most four digits without a
// leading zero.
func ParseSpec(text string) (Spec, bool) {
	if !specRE.MatchString(text) {
		return Spec{}, false
	}
	maj, min, _ := strings.Cut(text, ".")
	// The grammar bounds each part to four digits, so neither overflows.
	a, _ := strconv.Atoi(maj)
	b, _ := strconv.Atoi(min)
	return Spec{a, b}, true
}

// implemented returns the specification versions this tool implements
// (ADR-002 D8), ascending.
func implemented() []Spec {
	var out []Spec
	for _, v := range version.Implements() {
		s, ok := ParseSpec(v)
		if !ok {
			panic(fmt.Sprintf("manifest: the implemented version %q is not a specification version", v))
		}
		out = append(out, s)
	}
	return out
}

// Gate is ADR-002 D11 step 4: it judges the root's `metais` member by D7
// items 1, 7, 8 and 11, and returns the version declared when this tool
// implements it. Otherwise it returns the one finding that refuses the
// package, of class refused, and nothing after it may run: a manifest that
// names another set of rules cannot be judged by this tool's (D7 item 8).
//
// Gate reads only `metais`, so it judges a root of any shape: a root that is
// not an object has no `metais` member.
func Gate(root value.Value, src Source) (Spec, []finding.Finding) {
	impl := implemented()
	implList := make([]finding.Value, len(impl))
	for i, s := range impl {
		implList[i] = finding.String(s.String())
	}
	refuse := func(code finding.Code, ptr string, p finding.Params) (Spec, []finding.Finding) {
		p["implemented"] = finding.List(implList...)
		return Spec{}, []finding.Finding{src.finding(code, finding.SubjectPackage, ptr, p)}
	}

	obj, _ := root.(value.Object)
	v, ok := obj.Get("metais")
	if !ok {
		return refuse(CodeVersionMissing, "", finding.Params{})
	}
	ptr := finding.Pointer("metais")
	s, ok := v.(value.String)
	if !ok {
		p := finding.Params{"actual": finding.String(v.Kind().String())}
		if n, isNum := v.(value.Number); isNum {
			// D7 item 1: the finding quotes the number as it was written,
			// since its value (1.10 is 1.1) is not what the author meant.
			p["written"] = finding.String(n.Literal())
		}
		return refuse(CodeVersionNotString, ptr, p)
	}
	declared, ok := ParseSpec(s.Text())
	if !ok {
		return refuse(CodeVersionMalformed, ptr, finding.Params{})
	}
	highest, majorKnown := Spec{}, false
	for _, i := range impl {
		if i.Major == declared.Major {
			highest, majorKnown = i, true
		}
	}
	p := finding.Params{"declared": finding.String(declared.String())}
	switch {
	case !majorKnown:
		return refuse(CodeVersionMajor, ptr, p)
	case declared.Minor > highest.Minor:
		return refuse(CodeVersionMinor, ptr, p)
	}
	return declared, nil
}
