// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package filters

import (
	"strings"

	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
)

// noArgs refuses an application of name that has an argument, naming the
// first by its keyword, or by its position, 1, when it has none.
func noArgs(name, keyword string) *liquid.Error {
	if keyword == "" {
		keyword = "1"
	}
	return &liquid.Error{Code: liquid.CodeFilterArgument, Args: map[string]string{"filter": name, "argument": keyword},
		Detail: name + " takes no argument"}
}

// checkNoArgs is noArgs of an application at parse.
func checkNoArgs(name string, f *ast.Filter) *liquid.Error {
	if len(f.Args) == 0 {
		return nil
	}
	return noArgs(name, f.Args[0].Name)
}

// applyNoArgs is noArgs of an application at render.
func applyNoArgs(name string, args []liquid.Arg) *liquid.Error {
	if len(args) == 0 {
		return nil
	}
	return noArgs(name, args[0].Name)
}

// safeFilter is safe (ADR-004 D9 item 3): its input, unchanged and marked
// safe from escaping. It may be only the last filter of an output
// expression, which the parser holds it to. In an output whose escape is
// none nothing is escaped, so it does nothing there.
type safeFilter struct{}

func (safeFilter) Name() string                      { return "safe" }
func (safeFilter) Check(f *ast.Filter) *liquid.Error { return checkNoArgs("safe", f) }
func (safeFilter) MarksSafe() bool                   { return true }
func (safeFilter) OnlyLast() bool                    { return true }

// Apply charges the shallow size of the input, which is the result too
// (ADR-009 D11 item 2).
func (safeFilter) Apply(c liquid.FilterContext, in value.Value, args []liquid.Arg) (value.Value, *liquid.Error) {
	if err := applyNoArgs("safe", args); err != nil {
		return nil, err
	}
	if err := c.Meter().Charge(liquid.WorkUnits, 2*liquid.Shallow(in)); err != nil {
		return nil, err
	}
	return in, nil
}

// jsonScriptFilter is json_script (ADR-004 D10 item 5): the json filter's
// compact RFC 8785 text, with "<", ">", "&", U+2028 and U+2029 written as
// \u003c, \u003e, \u0026, \u2028 and \u2029, so that the text can neither
// close a <script> element nor open an HTML comment. Its result is safe
// (D9 item 2).
type jsonScriptFilter struct{}

func (jsonScriptFilter) Name() string                      { return "json_script" }
func (jsonScriptFilter) Check(f *ast.Filter) *liquid.Error { return checkNoArgs("json_script", f) }
func (jsonScriptFilter) MarksSafe() bool                   { return true }

// Apply charges the deep size of the input, which encoding visits whole,
// before encoding it, then the result's bytes as a string it builds, as
// work units (ADR-009 D11 item 2).
func (jsonScriptFilter) Apply(c liquid.FilterContext, in value.Value, args []liquid.Arg) (value.Value, *liquid.Error) {
	if err := applyNoArgs("json_script", args); err != nil {
		return nil, err
	}
	m := c.Meter()
	if err := m.Charge(liquid.WorkUnits, liquid.DeepSize(in)); err != nil {
		return nil, err
	}
	b, err := jsonx.Encode(in, 0)
	if err != nil {
		return nil, &liquid.Error{Code: liquid.CodeInternal, Detail: "json_script: " + err.Error()}
	}
	s := scriptSafe.Replace(string(b))
	n := int64(len(s))
	if err := m.Charge(liquid.WorkUnits, n); err != nil {
		return nil, err
	}
	v, verr := value.NewString(s)
	if verr != nil {
		return nil, &liquid.Error{Code: liquid.CodeInternal, Detail: "json_script: " + verr.Error()}
	}
	return v, nil
}

// scriptSafe is json_script's further escaping. Each character it replaces
// can stand in RFC 8785's text only inside a string, where \u is its
// escape.
var scriptSafe = strings.NewReplacer(
	"<", `\u003c`, ">", `\u003e`, "&", `\u0026`, "\u2028", `\u2028`, "\u2029", `\u2029`,
)
