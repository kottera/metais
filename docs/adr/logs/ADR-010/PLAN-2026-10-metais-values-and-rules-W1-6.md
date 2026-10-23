# ADR-010 — W1-6: the schema compiler registers the rules vocabulary (D5)

**2026-10-23, PLAN-2026-10-metais-values-and-rules W1-6.** D5 item 1's one
call site of `jsonschema.NewCompiler` now also registers ADR-003 Amendment
A2's rules vocabulary and switches on `AssertVocabs`. The vocabulary's
error kind embeds `*kind.Group` only for the text method the library's
`ErrorKind` requires, so `internal/schema` still imports no
`golang.org/x/text`, and the adapter maps the kind from its typed fields
without rendering any text (D5 item 5). `santhosh-tekuri/jsonschema/v6`
stays at v6.0.3; `go.mod` and `go.sum` are unchanged.
