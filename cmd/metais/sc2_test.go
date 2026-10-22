// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"metais.org/metais/internal/testenv"
)

// incidentReport is the package of PRD §4.2, as a source directory.
const incidentReport = "../../examples/dora-incident-report"

// sc2Stages are the three reports on a major ICT-related incident, as
// field 1.1 of Implementing Regulation (EU) 2025/302, Annex II, names
// them, in the order they are filed.
var sc2Stages = []string{"initial notification", "intermediate report", "final report"}

// sc2Field is one mandatory field of Annex II: the input that holds it,
// its property, the first stage whose report must hold it (every later
// stage's must too), and a value that holds no condition another field
// depends on.
type sc2Field struct {
	input, property string
	from            int
	value           any
}

// sc2Mandatory is every field Annex II marks mandatory, written from the
// glossary's columns and not from the package's schemas, so the test
// holds the schemas to the glossary.
var sc2Mandatory = []sc2Field{
	{"entity", "submitting_entity_name", 0, "Example Bank AG"},                          // 1.2
	{"entity", "submitting_entity_code", 0, "529900EXAMPLE0000012"},                     // 1.3
	{"entity", "affected_entity_types", 0, []string{"credit institution"}},              // 1.4
	{"entity", "primary_contact_name", 0, "Alex Example"},                               // 1.7
	{"entity", "primary_contact_email", 0, "alex@example.com"},                          // 1.8
	{"entity", "primary_contact_telephone", 0, "+49 69 1234567"},                        // 1.9
	{"entity", "second_contact_name", 0, "Sam Example"},                                 // 1.10
	{"entity", "second_contact_email", 0, "sam@example.com"},                            // 1.11
	{"entity", "second_contact_telephone", 0, "+49 69 7654321"},                         // 1.12
	{"entity", "reporting_currency", 0, "EUR"},                                          // 1.15
	{"incident", "reference_code", 0, "INC-1"},                                          // 2.1
	{"incident", "detected_at", 0, "2026-03-02T06:40:00Z"},                              // 2.2
	{"incident", "classified_at", 0, "2026-03-02T08:10:00Z"},                            // 2.3
	{"incident", "description", 0, "Card payments failed."},                             // 2.4
	{"incident", "classification_criteria", 0, []string{"critical services affected"}},  // 2.5
	{"incident", "discovery", 0, "monitoring systems"},                                  // 2.7
	{"incident", "business_continuity_plan_activated", 0, false},                        // 2.9
	{"incident", "occurred_at", 1, "2026-03-02T06:15:00Z"},                              // 3.2
	{"incident", "clients_affected", 1, 100},                                            // 3.4
	{"incident", "clients_affected_percentage", 1, 1.5},                                 // 3.5
	{"incident", "financial_counterparts_affected", 1, 0},                               // 3.6
	{"incident", "financial_counterparts_affected_percentage", 1, 0},                    // 3.7
	{"incident", "impact_figures_basis", 1, []string{"estimates for clients affected"}}, // 3.12
	{"incident", "duration", 1, "00:05:15"},                                             // 3.15
	{"incident", "critical_services_affected", 1, "Card payments."},                     // 3.22
	{"incident", "incident_types", 1, []string{"System failure"}},                       // 3.23
	{"incident", "affected_functional_areas", 1, "Retail payments."},                    // 3.27
	{"incident", "infrastructure_components_affected", 1, "No"},                         // 3.28
	{"incident", "client_financial_interest_impact", 1, "No"},                           // 3.30
	{"incident", "other_authorities_informed", 1, []string{"None"}},                     // 3.31
	{"incident", "temporary_actions", 1, false},                                         // 3.33
	{"incident", "root_cause_categories", 2, []string{"human error"}},                   // 4.1
	{"incident", "root_cause_details", 2, []string{"human error: mistake"}},             // 4.2
	{"incident", "root_cause_information", 2, "A wrong configuration value."},           // 4.5
	{"incident", "resolution_summary", 2, "The value was corrected."},                   // 4.6
	{"incident", "root_cause_addressed_at", 2, "2026-03-09T16:00:00Z"},                  // 4.7
	{"incident", "resolved_at", 2, "2026-03-10T09:00:00Z"},                              // 4.8
	{"incident", "resolution_date_difference", 2, "Resolved on the planned date."},      // 4.9
	{"incident", "economic_impact_threshold", 2, "No threshold reached."},               // 4.12
	{"incident", "gross_costs_and_losses", 2, 180},                                      // 4.13
	{"incident", "financial_recoveries", 2, 0},                                          // 4.14
}

// sc2Triggers are values that meet the condition of every conditional
// field Annex II makes mandatory on a condition the report itself shows:
// every classification criterion in 2.5, a service downtime in 3.16, and
// an "other", a "Yes", a cybersecurity-related incident or a refined
// root cause wherever one asks for more. sc2Conditional is each field
// they make mandatory, and the first stage it is mandatory in.
var (
	sc2Triggers = []sc2Field{
		{"entity", "affected_entity_name", 0, "Example Payments GmbH"}, // 1.5
		{"entity", "parent_undertaking_name", 0, "Example Holding SE"}, // 1.13
		{"incident", "classification_criteria", 0, []string{ // 2.5
			"clients, financial counterparts and transactions affected", "reputational impact",
			"duration and service downtime", "geographical spread", "data losses",
			"critical services affected", "economic impact"}},
		{"incident", "service_downtime", 0, "00:01:00"},                                                // 3.16
		{"incident", "incident_types", 0, []string{"Cybersecurity-related", "Other (please specify)"}}, // 3.23
		{"incident", "threats_and_techniques", 0, []string{"Other (please specify)"}},                  // 3.25
		{"incident", "infrastructure_components_affected", 0, "Yes"},                                   // 3.28
		{"incident", "other_authorities_informed", 0, []string{"Other (please specify)"}},              // 3.31
		{"incident", "temporary_actions", 0, true},                                                     // 3.33
		{"incident", "root_cause_details", 0, []string{ // 4.2
			"process failure: ICT risk management process failure", "process failure: other (please specify)"}},
		{"incident", "recurring_incidents", 0, "Two earlier gateway failures."}, // 4.15
	}
	sc2Conditional = []sc2Field{
		{"entity", "affected_entity_lei", 0, nil},                     // 1.6, with 1.5
		{"entity", "parent_undertaking_lei", 0, nil},                  // 1.14, with 1.13
		{"incident", "geographical_spread_countries", 0, nil},         // 2.6, geographical spread
		{"incident", "restored_at", 1, nil},                           // 3.3, with 3.16
		{"incident", "reputational_impact", 1, nil},                   // 3.13, reputational impact
		{"incident", "reputational_impact_context", 1, nil},           // 3.14
		{"incident", "duration_figures_basis", 1, nil},                // 3.17, duration and service downtime
		{"incident", "member_state_impact_types", 1, nil},             // 3.18, geographical spread
		{"incident", "member_state_impact_description", 1, nil},       // 3.19
		{"incident", "data_losses", 1, nil},                           // 3.20, data losses
		{"incident", "data_losses_description", 1, nil},               // 3.21
		{"incident", "other_incident_types", 1, nil},                  // 3.24, other
		{"incident", "other_techniques", 1, nil},                      // 3.26, other
		{"incident", "infrastructure_components_description", 1, nil}, // 3.29, Yes
		{"incident", "other_authorities", 1, nil},                     // 3.32, other
		{"incident", "temporary_actions_description", 1, nil},         // 3.34, true
		{"incident", "indicators_of_compromise", 1, nil},              // 3.35, cybersecurity-related
		{"incident", "additional_root_causes", 2, nil},                // 4.3, a refined 4.2 category
		{"incident", "other_root_causes", 2, nil},                     // 4.4, an other
		{"incident", "recurring_incidents_occurred_at", 2, nil},       // 4.16, with 4.15
	}
)

// TestSC2IncidentReportStages is SC2 (PRD §26): the incident report
// package of §4.2 accepts each stage's report with exactly that stage's
// mandatory fields, and reports every mandatory field a stage omits. For
// each stage, the built binary validates:
//
//   - the report holding exactly the fields mandatory at that stage, which
//     is accepted, though it omits every field mandatory only later;
//   - the report holding none of them, which reports each, and nothing
//     else;
//   - the previous stage's report, unchanged, which reports exactly the
//     fields the stage adds, so each report amends the one before it;
//   - and the stage's report with every condition met, which reports
//     exactly the conditional fields the stage makes mandatory.
func TestSC2IncidentReportStages(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	work := testenv.Dir(t)
	bin := filepath.Join(work, "metais")
	build := goCommand(t, "build", "-trimpath", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	pkg, err := filepath.Abs(incidentReport)
	if err != nil {
		t.Fatal(err)
	}
	home := testenv.Dir(t)

	// validate validates the stage's report, holding fields, and returns
	// whether it was accepted and each field reported missing, as
	// "<input>.<property>". Any other finding fails the test.
	validate := func(t *testing.T, stage int, fields ...[]sc2Field) (bool, []string) {
		t.Helper()
		data := testenv.Dir(t)
		in := map[string]map[string]any{"entity": {}, "incident": {}}
		for _, fs := range fields {
			for _, f := range fs {
				in[f.input][f.property] = f.value
			}
		}
		files := map[string]any{"stage": sc2Stages[stage], "entity": in["entity"], "incident": in["incident"]}
		for name, v := range files {
			b, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(data, name+".json"), b, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(bin, "validate", pkg, "--data", data, "--json")
		cmd.Dir = work
		cmd.Env = []string{"HOME=" + home, "NO_COLOR=1"}
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		var exit *exec.ExitError
		if err := cmd.Run(); err != nil && !errors.As(err, &exit) {
			t.Fatal(err)
		}
		var doc struct {
			OK       bool `json:"ok"`
			Findings []struct {
				Code    string `json:"code"`
				Subject struct {
					Name string `json:"name"`
				} `json:"subject"`
				Params struct {
					Property string `json:"property"`
				} `json:"params"`
			} `json:"findings"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
			t.Fatalf("metais validate: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.Bytes(), stderr.Bytes())
		}
		var missing []string
		for _, f := range doc.Findings {
			if f.Code != "input.required" {
				t.Fatalf("a finding that is not a missing field: %s\n%s", f.Code, stdout.Bytes())
			}
			missing = append(missing, f.Subject.Name+"."+f.Params.Property)
		}
		slices.Sort(missing)
		return doc.OK, missing
	}
	// due is each field of fs mandatory at stage, sorted.
	due := func(stage int, fs ...[]sc2Field) []string {
		set := map[string]bool{}
		for _, list := range fs {
			for _, f := range list {
				if f.from <= stage {
					set[f.input+"."+f.property] = true
				}
			}
		}
		return slices.Sorted(maps.Keys(set))
	}
	// upTo is the fields of fs mandatory at stage, with their values.
	upTo := func(stage int, fs []sc2Field) []sc2Field {
		return slices.DeleteFunc(slices.Clone(fs), func(f sc2Field) bool { return f.from > stage })
	}

	for stage, name := range sc2Stages {
		t.Run(name, func(t *testing.T) {
			t.Run("exactly its mandatory fields", func(t *testing.T) {
				if ok, missing := validate(t, stage, upTo(stage, sc2Mandatory)); !ok || len(missing) > 0 {
					t.Errorf("refused, missing %q", missing)
				}
			})
			t.Run("none of them", func(t *testing.T) {
				ok, missing := validate(t, stage)
				if want := due(stage, sc2Mandatory); ok || !slices.Equal(missing, want) {
					t.Errorf("ok %t, missing %q\nwant missing %q", ok, missing, want)
				}
			})
			if stage > 0 {
				t.Run("the previous report", func(t *testing.T) {
					ok, missing := validate(t, stage, upTo(stage-1, sc2Mandatory))
					want := slices.DeleteFunc(due(stage, sc2Mandatory), func(k string) bool {
						return slices.Contains(due(stage-1, sc2Mandatory), k)
					})
					if ok || !slices.Equal(missing, want) {
						t.Errorf("ok %t, missing %q\nwant missing %q", ok, missing, want)
					}
				})
			}
			t.Run("every condition met", func(t *testing.T) {
				ok, missing := validate(t, stage, upTo(stage, sc2Mandatory), sc2Triggers)
				if want := due(stage, sc2Conditional); ok || !slices.Equal(missing, want) {
					t.Errorf("ok %t, missing %q\nwant missing %q", ok, missing, want)
				}
			})
		})
	}
}
