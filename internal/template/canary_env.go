// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build metais_canary_env

package template

import (
	"os"
	"strings"

	"metais.org/metais/internal/liquid/value"
)

// The env canary (ADR-009 D16 item 2): the binding set gains env, the
// host's environment as an object, read when the process starts, before
// hostenv.Isolate clears it. It is a deliberate leak, built only under
// its tag, which the SC7 harness must catch on the environment's channel.
func init() {
	var ms []value.Member
	seen := map[string]bool{}
	for _, kv := range os.Environ() {
		k, v, _ := strings.Cut(kv, "=")
		s, err := value.NewString(v)
		if err != nil || seen[k] {
			continue
		}
		seen[k] = true
		ms = append(ms, value.Member{Name: k, Value: s})
	}
	env, err := value.NewObject(ms...)
	if err != nil {
		panic(err)
	}
	canaryGlobals = func() []value.Member { return []value.Member{{Name: "env", Value: env}} }
}
