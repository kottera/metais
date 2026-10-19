// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/inputs"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/loader"
)

// The supply flags of validate and render (ADR-008 D7).
var (
	dataFlag = args.Flag{Name: "data", Value: "<dir>",
		Description: "The data directory: one file per input named <input>.<ext>, and one directory per collection input. metais inspect lists each input's file names."}
	setFlag = args.Flag{Name: "set", Value: "<pointer>=<text>", Repeatable: true,
		Description: "Set one value at an inputs pointer, such as /vendor/name=Acme; the text is coerced to the type the schema declares there. Applied after --data and --inputs-json, in command-line order with --set-json."}
	setJSONFlag = args.Flag{Name: "set-json", Value: "<pointer>=<json>", Repeatable: true,
		Description: "Set one value at an inputs pointer as JSON text, the way to set null, an array or an object. Applied in command-line order with --set."}
	inputsJSONFlag = args.Flag{Name: "inputs-json", Value: "<json>", Repeatable: true,
		Description: "A JSON object whose members are inputs, each replacing that input's whole value; - reads it from standard input, once. Applied after --data, in command-line order."}
)

// supplyFlags are the supply flags, in the order help lists them.
var supplyFlags = []args.Flag{dataFlag, setFlag, setJSONFlag, inputsJSONFlag}

// stdinOperand is the --inputs-json value that reads standard input.
const stdinOperand = "-"

// The steps after validate (ADR-008 D14 item 3).
var (
	nextValidateAgain = Next{Run: "metais validate <package> --data <data dir> <your --set, --set-json and --inputs-json flags>",
		Why: "Validates the inputs again once the findings are corrected."}
	nextRender = Next{Run: "metais render <package> --data <data dir> <your --set, --set-json and --inputs-json flags> --out <dir>",
		Why: "Renders the outputs from the inputs just validated into a new directory."}
	nextInputsHelp = Next{Run: "metais help inputs", Why: "Explains how inputs are supplied: the data directory, --inputs-json, --set and --set-json."}
)

// validateDeclaration is metais validate (ADR-008 D7).
func validateDeclaration() *Command {
	return &Command{
		Name: "validate", Group: "Supplying and rendering",
		Synopsis:    "validate <package> [<supply flags>] [--resolved] [--json]",
		Description: "Validate inputs against a package, and report every finding at once.",
		Detail: "Opens and verifies the package as inspect does, then assembles the inputs object: " +
			"each file of the data directory, then each --inputs-json in command-line order, then each " +
			"--set and --set-json in command-line order, each replacing what came before at its place. " +
			"Each schema default is applied to a member left absent, and the whole is validated against " +
			"the package's schemas and its input constraints.\n\n" +
			"Every finding is reported in one run: a file that does not parse does not stop the others " +
			"from being read and validated. With no supply flag, nothing is supplied, and every required " +
			"input is reported missing.\n\n" +
			"The result lists, for every input, whether it was supplied and where its value came from. " +
			"--resolved adds the inputs as templates would see them, after coercion and defaults, in " +
			"canonical key order; in human form it is printed as indented JSON after the findings.",
		Role:     finding.RoleConsumer,
		Operands: []args.Operand{{Name: "<package>", Required: true, Description: "A packed package file, or a source directory; - is refused, since a package is read with random access."}},
		Flags: append(slices.Clone(supplyFlags),
			args.Flag{Name: "resolved", Description: "Add the inputs as templates would see them: parsed and coerced, with defaults applied, absent members absent."},
			limitFlag, failOnFlag),
		Format:            "metais.validate/1",
		NextToStdout:      true,
		FindingsAreResult: true,
		Prints:            "Every finding, then the summary line; with --resolved, the inputs as indented JSON.",
		Refusals: append(packageRefusals("validate"),
			Refusal{finding.CodeUsageDataDirUnreadable, "The data directory given to --data does not exist, is not a directory, or cannot be read.", nextInputsHelp},
			Refusal{finding.CodeUsageConflictingFlags, "--inputs-json - is given twice, and standard input is read once.", nextInputsHelp},
		),
		Examples: []Example{
			{"metais validate report-1.0.0.metais --data data", "Validates the inputs in the directory data."},
			{"metais validate report-1.0.0.metais --data data --set /vendor/name=Acme", "Validates them with one value replaced."},
			{"metais validate report-1.0.0.metais --inputs-json - --json", "Validates the inputs object read from standard input, and prints one metais.validate/1 document."},
			{"metais validate report-1.0.0.metais --data data --resolved", "Validates them and prints the inputs as templates would see them."},
		},
		run: runValidate,
	}
}

// validateResult is metais.validate/1's result (ADR-008 D7 items 5 and 6).
type validateResult struct {
	Inputs []validateInput  `json:"inputs"`
	Data   *json.RawMessage `json:"data,omitempty"`
}

type validateInput struct {
	Name     string         `json:"name"`
	Supplied bool           `json:"supplied"`
	From     []validateFrom `json:"from"`
}

// validateFrom is one source of an input's value: the data directory's
// file, an --inputs-json or a path set by its ordinal, or a default.
type validateFrom struct {
	Source   string `json:"source"`
	File     string `json:"file,omitempty"`
	Argument int    `json:"argument,omitempty"`
}

// runValidate is metais validate.
func runValidate(s *session, p args.Parsed) outcome {
	refuse := func(fs ...finding.Finding) outcome { return outcome{findings: fs, next: s.nextSteps(fs)} }
	if p.Operands[0] == "-" {
		return refuse(finding.New(finding.CodeUsagePackageFromStdin, invocation, nil,
			finding.Params{"argument": finding.Int(int64(s.line.at[p.OperandArgs[0]] + 1))}))
	}
	lim, fs := s.limits(p)
	if len(fs) > 0 {
		return refuse(fs...)
	}
	if fs := s.stdinOnce(p); len(fs) > 0 {
		return refuse(fs...)
	}
	pkg, fs := loader.Open(p.Operands[0], loader.Options{Limits: lim}, s.inv.Options.Faults)
	if pkg == nil {
		return refuse(fs...)
	}
	defer pkg.Close()
	res, fs := s.supply(p, pkg, lim)
	if res == nil {
		return refuse(fs...)
	}
	out := &validateResult{Inputs: make([]validateInput, 0, len(res.Inputs))}
	var data []value.Member
	for _, r := range res.Inputs {
		vi := validateInput{Name: r.Name, From: []validateFrom{}}
		for _, f := range r.From {
			vf := validateFrom{Argument: f.Argument}
			switch f.Kind {
			case inputs.FromFile:
				vf.Source, vf.File = "data_dir", f.File
			case inputs.FromInputsJSON:
				vf.Source = "inputs_json"
			case inputs.FromSet:
				vf.Source = "set"
			case inputs.FromSetJSON:
				vf.Source = "set_json"
			case inputs.FromDefault:
				vf.Source = "default"
			}
			vi.Supplied = vi.Supplied || f.Kind != inputs.FromDefault
			vi.From = append(vi.From, vf)
		}
		out.Inputs = append(out.Inputs, vi)
		if r.Value != nil {
			data = append(data, value.Member{Name: r.Name, Value: r.Value})
		}
	}
	var human []byte
	if p.Has("resolved") {
		obj, err := value.NewObject(data...)
		if err != nil {
			panic("cli: " + err.Error())
		}
		raw := json.RawMessage(jsonx.Canonical(obj))
		out.Data = &raw
		text, err := jsonx.Encode(obj, 2)
		if err != nil {
			panic("cli: " + err.Error())
		}
		for _, l := range strings.Split(string(text), "\n") {
			human = append(finding.AppendText(human, l), '\n')
		}
	}
	next := []Next{nextRender}
	if slices.ContainsFunc(res.Findings, func(f finding.Finding) bool { return f.Severity() == finding.SeverityError }) {
		next = []Next{nextValidateAgain, nextInputsHelp}
	}
	dataDir, _ := p.Value(dataFlag.Name)
	return outcome{findings: res.Findings, result: out, human: human, next: declaredSteps(next, s.table), report: true, dataDir: dataDir}
}

// stdinOnce refuses a line that gives --inputs-json - more than once:
// standard input is read once (ADR-008 D7 item 2).
func (s *session) stdinOnce(p args.Parsed) []finding.Finding {
	jsonArgs, jsonAt := p.Values(inputsJSONFlag.Name), p.ArgsOf(inputsJSONFlag.Name)
	seen := false
	for i, v := range jsonArgs {
		if v != stdinOperand {
			continue
		}
		if seen {
			return []finding.Finding{finding.New(finding.CodeUsageConflictingFlags, invocation, nil, finding.Params{
				"argument": finding.Int(int64(s.line.at[jsonAt[i]] + 1)),
				"flag":     finding.String("--" + inputsJSONFlag.Name),
			})}
		}
		seen = true
	}
	return nil
}

// supply reads the supply flags of p into pkg's inputs layer under lim —
// the data directory, then each --inputs-json, then each --set and
// --set-json, in command-line order (ADR-008 D7, ADR-003 D18) — and
// validates the result (ADR-003 D22). It is validate's and render's one
// supply. It returns the result, or no result and the refusal of a data
// directory that cannot be read.
func (s *session) supply(p args.Parsed, pkg *loader.Package, lim limits.Set) (*inputs.Result, []finding.Finding) {
	layer, err := inputs.New(inputs.Package{Manifest: pkg.Manifest, Compiler: pkg.Schemas, Combined: pkg.CombinedValue, Schema: pkg.Combined})
	if err != nil {
		panic("cli: " + err.Error())
	}
	sup := layer.NewSupply(lim)
	if dir, ok := p.Value(dataFlag.Name); ok {
		var de *inputs.DataDirError
		if err := sup.ReadDataDir(dir); errors.As(err, &de) {
			at, _ := p.Arg(dataFlag.Name)
			return nil, []finding.Finding{finding.New(finding.CodeUsageDataDirUnreadable, invocation, nil, finding.Params{
				"argument": finding.Int(int64(s.line.at[at] + 1)),
				"reason":   finding.String(de.Reason),
			})}
		} else if err != nil {
			panic("cli: " + err.Error())
		}
	}
	for _, v := range p.Values(inputsJSONFlag.Name) {
		text := []byte(v)
		if v == stdinOperand {
			text = readStdin(s.inv.Stdin, lim.Value(limits.InputSize))
		}
		sup.AddInputsJSON(text)
	}
	setVals, setAt := p.Values(setFlag.Name), p.ArgsOf(setFlag.Name)
	jsonVals, jsonSetAt := p.Values(setJSONFlag.Name), p.ArgsOf(setJSONFlag.Name)
	for i, j := 0, 0; i < len(setVals) || j < len(jsonVals); {
		if j == len(jsonVals) || i < len(setVals) && setAt[i] < jsonSetAt[j] {
			sup.AddSet(setVals[i])
			i++
		} else {
			sup.AddSetJSON(jsonVals[j])
			j++
		}
	}
	res, err := sup.Validate()
	if err != nil {
		panic("cli: " + err.Error())
	}
	return res, res.Findings
}

// readStdin reads standard input for --inputs-json -: at most limit bytes
// and one more, so that the inputs layer sees an argument over
// input-size as one (ADR-008 D7 item 2). A read that fails part way gives
// what it read, which the inputs layer then refuses as JSON that does not
// end.
func readStdin(r io.Reader, limit int64) []byte {
	if r == nil {
		return nil
	}
	b, _ := io.ReadAll(io.LimitReader(r, limit+1))
	return b
}

// declaredSteps is next without the steps that name a command t does not
// declare: a step names a command a caller can run.
func declaredSteps(next []Next, t []*Command) []Next {
	return slices.DeleteFunc(slices.Clone(next), func(n Next) bool {
		name, _, _ := strings.Cut(strings.TrimPrefix(n.Run, "metais "), " ")
		return lookup(t, name) == nil
	})
}
