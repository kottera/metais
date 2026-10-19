// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// The suite's own schemas (ADR-011 D7 item 3), by file name below
// spec/schemas/conformance/1/.
const (
	vectorSchema         = "vector.schema.json"
	findingsSchema       = "findings.schema.json"
	driverRequestSchema  = "driver-request.schema.json"
	driverResponseSchema = "driver-response.schema.json"
	recordSchema         = "record.schema.json"
)

// schemaBase is the $id prefix of the suite's schemas.
const schemaBase = "https://metais.org/schemas/conformance/1/"

// noLoader refuses every schema the compiler was not given: the suite's
// schemas hold no external reference, and nothing is fetched (ADR-011 D7
// items 4 and 7).
type noLoader struct{}

func (noLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("no schema is loaded from %s", url)
}

// schemaSet is the compiled suite schemas of one spec tree.
type schemaSet struct {
	byName map[string]*jsonschema.Schema
}

var (
	schemaMu    sync.Mutex
	schemaCache = map[string]*schemaSet{}
)

// suiteSchemas compiles the suite's schemas from the spec tree holding
// suite, at ../schemas/conformance/1/, once per tree.
func suiteSchemas(suite string) (*schemaSet, error) {
	dir := filepath.Join(suite, "..", "schemas", "conformance", "1")
	schemaMu.Lock()
	defer schemaMu.Unlock()
	if s, ok := schemaCache[dir]; ok {
		return s, nil
	}
	set, err := compileSchemas(dir)
	if err != nil {
		return nil, err
	}
	schemaCache[dir] = set
	return set, nil
}

// compileSchemas compiles every suite schema in dir, each checked against
// the 2020-12 meta-schema.
func compileSchemas(dir string) (*schemaSet, error) {
	c := jsonschema.NewCompiler()
	c.UseLoader(noLoader{})
	names := []string{vectorSchema, findingsSchema, driverRequestSchema, driverResponseSchema, recordSchema}
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return nil, err
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		if err := c.AddResource(schemaBase+n, doc); err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
	}
	set := &schemaSet{byName: map[string]*jsonschema.Schema{}}
	for _, n := range names {
		s, err := c.Compile(schemaBase + n)
		if err != nil {
			return nil, fmt.Errorf("%s does not compile: %w", n, err)
		}
		set.byName[n] = s
	}
	return set, nil
}

// validator returns a function validating a JSON document against the
// named schema.
func (s *schemaSet) validator(name string) func([]byte) error {
	sch := s.byName[name]
	return func(b []byte) error {
		inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
		if err != nil {
			return err
		}
		return sch.Validate(inst)
	}
}
