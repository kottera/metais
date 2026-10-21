// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// The record formats. A change to what a record means changes its format.
const (
	setFormat = "metais-ux-set/1"
	runFormat = "metais-ux-run/1"
)

// The files of a run set's directory and of each run's.
const (
	setFile        = "set.json"
	recordFile     = "record.json"
	transcriptFile = "transcript.jsonl"
	stderrFile     = "stderr.txt"
	workDir        = "work"
)

// setRecord is set.json: what every run of the set measured. A set
// resumes only on the same commit, binary, fixtures, harness version and
// skill mode, so its runs are of one thing.
type setRecord struct {
	Format string `json:"format"`
	Commit string `json:"commit"`
	Dirty  bool   `json:"dirty"`
	Metais struct {
		SHA256  string `json:"sha256"`
		Version string `json:"version"`
	} `json:"metais"`
	Harness   harnessRecord `json:"harness"`
	WithSkill bool          `json:"with_skill"`
	// Fixtures are the packages packed for the set, by scenario and file.
	Fixtures []fixtureRecord `json:"fixtures"`
	Started  string          `json:"started"`
}

type harnessRecord struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type fixtureRecord struct {
	Scenario string `json:"scenario"`
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
}

// The run statuses.
const (
	// statusComplete: the session ran and ended of itself.
	statusComplete = "complete"
	// statusTimeout: the session ran past -timeout and was stopped.
	statusTimeout = "timeout"
	// statusHarnessError: the harness ended without a result.
	statusHarnessError = "harness-error"
	// statusNotRun: an isolation check failed before the session started,
	// so it never ran.
	statusNotRun = "not-run"
	// statusQuota: the session stopped on a quota or rate limit. The run
	// is run again when the set resumes.
	statusQuota = "quota"
)

// runRecord is a run's record.json. The scorer (PLAN-2026-10-metais-ux
// W1-2) reads it beside the transcript and the session's directory, and
// adds Outcome; the runner writes Outcome as null.
type runRecord struct {
	Format   string `json:"format"`
	Scenario struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Kind          string `json:"kind"`
		AsksForOutput bool   `json:"asks_for_output"`
		Prompt        string `json:"prompt"`
	} `json:"scenario"`
	// ModelRequested is the model as the run asked for it; ModelReported
	// is the model the session reports it ran on, and ModelsUsed every
	// model its usage names.
	ModelRequested string        `json:"model_requested"`
	ModelReported  string        `json:"model_reported"`
	ModelsUsed     []string      `json:"models_used"`
	Harness        harnessRecord `json:"harness"`
	WithSkill      bool          `json:"with_skill"`
	Commit         string        `json:"commit"`
	MetaisSHA256   string        `json:"metais_sha256"`

	Status string `json:"status"`
	// Valid is true when every isolation check passed. An invalid run is
	// recorded and is not scored.
	Valid  bool    `json:"valid"`
	Checks []check `json:"checks"`

	// The runner's clock: when the session started and ended, and the
	// wall time between, in milliseconds.
	Started string `json:"started"`
	Ended   string `json:"ended"`
	WallMS  int64  `json:"wall_ms"`
	// What the harness reports: its turns, its cost in US dollars, its own
	// duration, whether its result is an error, and the result's subtype.
	Turns      int     `json:"turns"`
	CostUSD    float64 `json:"cost_usd"`
	DurationMS int64   `json:"duration_ms"`
	IsError    bool    `json:"is_error"`
	Subtype    string  `json:"subtype"`
	ExitCode   int     `json:"exit_code"`
	ToolUses   int     `json:"tool_uses"`

	// Invocation is the command exactly as the runner started it.
	Invocation struct {
		Path string   `json:"path"`
		Args []string `json:"args"`
		Env  []string `json:"env"`
		Dir  string   `json:"dir"`
	} `json:"invocation"`

	// Fixtures are the files the session's directory started with, and
	// Files every file it holds after the session, each marked added,
	// modified, unchanged or deleted.
	Fixtures []fileEntry `json:"fixtures"`
	Files    []fileEntry `json:"files"`

	// Transcript, Stderr and Work name the transcript, the harness's
	// standard error and the session's directory, relative to the run's.
	Transcript string `json:"transcript"`
	Stderr     string `json:"stderr"`
	Work       string `json:"work"`

	// Verdict is a recorded judgement of a C0 or C1 run (package score's
	// Verdict), written by a judge model or a person, never by the runner.
	// The runner keeps it as written when it rewrites the record.
	Verdict json.RawMessage `json:"verdict,omitempty"`

	// Outcome is the scorer's; the runner writes it when it scores the set.
	Outcome json.RawMessage `json:"outcome"`
}

// writeJSON writes v to path, through a temporary file renamed into place,
// so an interrupted runner never leaves half a record that a resume would
// read as whole.
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".partial"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// readJSON reads path into v, reporting fs.ErrNotExist when it is absent.
func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// readRecord reads a run's record, or nil when the run has none.
func readRecord(runDir string) (*runRecord, error) {
	var r runRecord
	err := readJSON(filepath.Join(runDir, recordFile), &r)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, err
	case r.Format != runFormat:
		return nil, fmt.Errorf("%s: format %q, not %s", runDir, r.Format, runFormat)
	}
	return &r, nil
}

// finished reports whether a recorded run is kept when the set resumes.
// Only a quota pause is run again: every other status is the session's
// behaviour, or the runner's refusal to start it, and is the record.
func finished(r *runRecord) bool { return r != nil && r.Status != statusQuota }
