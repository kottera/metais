// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
	"fmt"

	"metais.org/metais/internal/liquid/value"
)

// toLibrary converts a value of the data model to the form the library
// reads (ADR-010 D5 item 4): map[string]any, []any, string, bool, nil, and
// for a number json.Number holding the literal text it was written as,
// never a float64, so the library compares and divides the number written
// rather than a re-rounded double.
func toLibrary(v value.Value) any {
	switch v := v.(type) {
	case value.Null:
		return nil
	case value.Bool:
		return bool(v)
	case value.Number:
		return json.Number(v.Literal())
	case value.String:
		return v.Text()
	case value.Array:
		out := make([]any, 0, v.Len())
		for _, e := range v.All() {
			out = append(out, toLibrary(e))
		}
		return out
	case value.Object:
		out := make(map[string]any, v.Len())
		for k, e := range v.All() {
			out[k] = toLibrary(e)
		}
		return out
	}
	panic(fmt.Sprintf("schema: a value of type %T is not of the data model", v))
}
