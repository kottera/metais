// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"strconv"
	"strings"

	"metais.org/metais/internal/cli/args"
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/version"
)

// versionResult is metais.version/1's result: ADR-002 D8 item 1's content.
type versionResult struct {
	Tool          toolVersion          `json:"tool"`
	Specification specificationVersion `json:"specification"`
}

type toolVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type specificationVersion struct {
	Implements []string `json:"implements"`
}

// toolName is the tool's name, as metais version prints it and the
// provenance tag records it (ADR-004 D14 item 2).
const toolName = "metais"

// makerLine is the last line of metais version's text output, the one
// place metais names its maker (ADR-008 A14, ADR-013 D2 item 2). The JSON
// form carries no member for it.
const makerLine = "METAIS is made by Kottera."

// runVersion is metais version: the tool's version, from internal/version,
// every specification version it implements and, in text, its maker line.
func runVersion(*session, args.Parsed) outcome {
	v, impl := version.String(), version.Implements()
	human := toolName + " " + v + "\nspecification " + strings.Join(impl, ", ") + "\n" + makerLine + "\n"
	return outcome{
		result: &versionResult{toolVersion{toolName, v}, specificationVersion{impl}},
		human:  []byte(human),
	}
}

// limitsResult is metais.limits/1's result: ADR-009 D3's table, each limit
// with its name, default, unit and class (ADR-009 D4 item 3).
type limitsResult struct {
	Limits []limitRow `json:"limits"`
}

type limitRow struct {
	Name    string `json:"name"`
	Default int64  `json:"default"`
	Unit    string `json:"unit"`
	Class   string `json:"class"`
}

// runLimits is metais limits: internal/limits' table, in its order, and
// nothing else.
func runLimits(*session, args.Parsed) outcome {
	all := limits.All()
	res := &limitsResult{Limits: make([]limitRow, len(all))}
	rows := [][]string{{"NAME", "DEFAULT", "UNIT", "CLASS"}}
	for i, l := range all {
		res.Limits[i] = limitRow{l.Name, l.Default, l.Unit.String(), l.Class.String()}
		rows = append(rows, []string{l.Name, humanDefault(l), l.Unit.String(), l.Class.String()})
	}
	return outcome{result: res, human: columns(rows)}
}

// humanDefault is a limit's default as the human table shows it: a size
// in the largest of GiB, MiB and KiB it is a whole number of, otherwise
// the number.
func humanDefault(l limits.Limit) string {
	if l.Unit == limits.Bytes && l.Default > 0 {
		for _, u := range []struct {
			name string
			size int64
		}{{"GiB", 1 << 30}, {"MiB", 1 << 20}, {"KiB", 1 << 10}} {
			if l.Default%u.size == 0 {
				return strconv.FormatInt(l.Default/u.size, 10) + u.name
			}
		}
	}
	return strconv.FormatInt(l.Default, 10)
}

// columns lays rows out in columns two spaces apart, each as wide as its
// widest cell, the last unpadded.
func columns(rows [][]string) []byte {
	var width []int
	for _, r := range rows {
		for i, c := range r {
			if i == len(width) {
				width = append(width, 0)
			}
			width[i] = max(width[i], len(c))
		}
	}
	var b []byte
	for _, r := range rows {
		for i, c := range r {
			b = append(b, c...)
			if i < len(r)-1 {
				b = append(b, strings.Repeat(" ", width[i]-len(c)+2)...)
			}
		}
		b = append(b, '\n')
	}
	return b
}

// runHelp is metais help: the overview, a command's help, or a topic.
func runHelp(s *session, p args.Parsed) outcome {
	if len(p.Operands) == 0 {
		return s.overview()
	}
	name := p.Operands[0]
	if len(p.Operands) == 2 {
		// Only the refusals topic takes a command after it.
		at := s.line.at[p.OperandArgs[1]]
		if name != "refusals" {
			f := finding.New(finding.CodeUsageUnexpectedOperand, invocation, nil,
				finding.Params{"argument": finding.Int(int64(at + 1)), "command": finding.String("help")})
			return outcome{findings: []finding.Finding{f}, next: s.nextSteps([]finding.Finding{f})}
		}
		c := lookup(s.table, p.Operands[1])
		if c == nil {
			f := finding.New(finding.CodeUsageUnknownTopic, invocation, nil, finding.Params{"argument": finding.Int(int64(at + 1))})
			return outcome{findings: []finding.Finding{f}, next: s.nextSteps([]finding.Finding{f})}
		}
		return s.topicHelp(commandRefusalsTopic(c))
	}
	if c := lookup(s.table, name); c != nil {
		return s.commandHelp(c)
	}
	if t, ok := lookupTopic(name); ok {
		return s.topicHelp(t)
	}
	f := finding.New(finding.CodeUsageUnknownTopic, invocation, nil,
		finding.Params{"argument": finding.Int(int64(s.line.at[p.OperandArgs[0]] + 1))})
	return outcome{findings: []finding.Finding{f}, next: s.nextSteps([]finding.Finding{f})}
}
