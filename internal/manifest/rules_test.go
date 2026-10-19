// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
)

// ruleCase is one manifest: the valid base with one change, and the
// verdict it must get.
type ruleCase struct {
	name   string
	change func(t testing.TB, m tree)
	// ptr and rule are the violation Build must name; ptr "-" for a case
	// that is accepted.
	ptr, rule string
	// byCheck marks a rule the schema does not express (ADR-002 D10 item
	// 6): the schema accepts the manifest and Check refuses it.
	byCheck bool
}

// to sets member key of the object at path.
func to(path, key string, v any) func(testing.TB, tree) {
	return func(t testing.TB, m tree) { set(t, m, path, key, v) }
}

// both applies each change in turn.
func both(changes ...func(testing.TB, tree)) func(testing.TB, tree) {
	return func(t testing.TB, m tree) {
		for _, c := range changes {
			c(t, m)
		}
	}
}

const ok = "-"

// ruleCases are every field rule of ADR-002 D4 to D7, ADR-003 D1 and
// ADR-005 D1, D2 and D7 to D9, each broken once, beside the accepted
// values at and around its bounds.
func ruleCases() []ruleCase {
	r := strings.Repeat
	cs := []ruleCase{
		{name: "base", change: func(testing.TB, tree) {}, ptr: ok},

		// D4 item 2: the root's members.
		{name: "unknown root member", change: to("", "x-notes", "a"), ptr: "/x-notes", rule: ruleUnknown},
		{name: "root member misspelt", change: to("", "homepages", "https://x.org"), ptr: "/homepages", rule: ruleUnknown},
		// D4 item 5: null is never a value.
		{name: "null homepage", change: to("", "homepage", nil), ptr: "/homepage", rule: ruleType},
		{name: "null license", change: to("", "license", nil), ptr: "/license", rule: ruleType},
		{name: "authors not an array", change: to("", "authors", tree{}), ptr: "/authors", rule: ruleType},

		// D5: the name.
		{name: "name upper case", change: to("", "name", "Org.example"), ptr: "/name", rule: ruleName},
		{name: "name double hyphen", change: to("", "name", "a--b"), ptr: "/name", rule: ruleName},
		{name: "name trailing hyphen", change: to("", "name", "status-"), ptr: "/name", rule: ruleName},
		{name: "name leading hyphen", change: to("", "name", "-status"), ptr: "/name", rule: ruleName},
		{name: "name empty segment", change: to("", "name", "a..b"), ptr: "/name", rule: ruleName},
		{name: "name leading dot", change: to("", "name", ".a"), ptr: "/name", rule: ruleName},
		{name: "name underscore", change: to("", "name", "a_b"), ptr: "/name", rule: ruleName},
		{name: "name non-ASCII", change: to("", "name", "café"), ptr: "/name", rule: ruleName},
		{name: "name con", change: to("", "name", "con"), ptr: "/name", rule: ruleName},
		{name: "name con.reports", change: to("", "name", "con.reports"), ptr: "/name", rule: ruleName},
		{name: "name nul.reports.metais", change: to("", "name", "nul.reports.metais"), ptr: "/name", rule: ruleName},
		{name: "name com0", change: to("", "name", "com0"), ptr: "/name", rule: ruleName},
		{name: "name lpt9.x", change: to("", "name", "lpt9.x"), ptr: "/name", rule: ruleName},
		{name: "name 64-character segment", change: to("", "name", "org."+r("a", 64)), ptr: "/name", rule: ruleName},
		{name: "name 129 characters", change: to("", "name", r("a", 63)+"."+r("b", 63)+"."+"c"), ptr: "/name", rule: ruleLength},
		{name: "name empty", change: to("", "name", ""), ptr: "/name", rule: ruleLength},
		{name: "name a number", change: to("", "name", 7), ptr: "/name", rule: ruleType},
		{name: "name console", change: to("", "name", "console"), ptr: ok},
		{name: "name trailing device segment", change: to("", "name", "reports.con"), ptr: ok},
		{name: "name com10", change: to("", "name", "com10"), ptr: ok},
		{name: "name single segment", change: to("", "name", "status-widget"), ptr: ok},
		{name: "name 63-character segment", change: to("", "name", "org."+r("a", 63)), ptr: ok},
		{name: "name 128 characters", change: to("", "name", r("a", 63)+"."+r("b", 62)+".c"), ptr: ok},

		// D6: the version.
		{name: "version leading v", change: to("", "version", "v1.0.0"), ptr: "/version", rule: ruleVersion},
		{name: "version two parts", change: to("", "version", "1.0"), ptr: "/version", rule: ruleVersion},
		{name: "version leading zero", change: to("", "version", "01.0.0"), ptr: "/version", rule: ruleVersion},
		{name: "version pre-release leading zero", change: to("", "version", "1.0.0-01"), ptr: "/version", rule: ruleVersion},
		{name: "version empty pre-release", change: to("", "version", "1.0.0-"), ptr: "/version", rule: ruleVersion},
		{name: "version a number", change: to("", "version", json.RawMessage("1.0")), ptr: "/version", rule: ruleType},
		{name: "version 129 characters", change: to("", "version", "1.0.0-"+r("a", 123)), ptr: "/version", rule: ruleLength},
		{name: "version 128 characters", change: to("", "version", "1.0.0-"+r("a", 122)), ptr: ok},
		{name: "version pre-release", change: to("", "version", "2.1.0-rc.1"), ptr: ok},
		{name: "version build metadata", change: to("", "version", "2.1.0+20261018"), ptr: ok},
		{name: "version zero", change: to("", "version", "0.0.0"), ptr: ok},

		// D4 items 6 and 7: the title, a line of 1 to 200.
		{name: "title empty", change: to("", "title", ""), ptr: "/title", rule: ruleLength},
		{name: "title 201", change: to("", "title", r("t", 201)), ptr: "/title", rule: ruleLength},
		{name: "title leading space", change: to("", "title", " Report"), ptr: "/title", rule: ruleLine},
		{name: "title trailing space", change: to("", "title", "Report "), ptr: "/title", rule: ruleLine},
		{name: "title line feed", change: to("", "title", "A\nB"), ptr: "/title", rule: ruleLine},
		{name: "title tab", change: to("", "title", "A\tB"), ptr: "/title", rule: ruleLine},
		{name: "title NUL", change: to("", "title", "A\u0000B"), ptr: "/title", rule: ruleLine},
		{name: "title DEL", change: to("", "title", "A\u007fB"), ptr: "/title", rule: ruleLine},
		{name: "title C1 control", change: to("", "title", "A\u0085B"), ptr: "/title", rule: ruleLine},
		{name: "title 200", change: to("", "title", r("t", 200)), ptr: ok},
		{name: "title 200 non-ASCII", change: to("", "title", r("é", 200)), ptr: ok},
		{name: "title one character", change: to("", "title", "A"), ptr: ok},
		{name: "title no-break space ends", change: to("", "title", " A "), ptr: ok},

		// D4 item 8: the description, prose of 1 to 1000.
		{name: "description empty", change: to("", "description", ""), ptr: "/description", rule: ruleLength},
		{name: "description 1001", change: to("", "description", r("d", 1001)), ptr: "/description", rule: ruleLength},
		{name: "description leading line feed", change: to("", "description", "\nA"), ptr: "/description", rule: ruleProse},
		{name: "description trailing tab", change: to("", "description", "A\t"), ptr: "/description", rule: ruleProse},
		{name: "description trailing space", change: to("", "description", "A "), ptr: "/description", rule: ruleProse},
		{name: "description carriage return", change: to("", "description", "A\rB"), ptr: "/description", rule: ruleProse},
		{name: "description vertical tab", change: to("", "description", "A\u000bB"), ptr: "/description", rule: ruleProse},
		{name: "description 1000", change: to("", "description", r("d", 1000)), ptr: ok},
		{name: "description inner breaks", change: to("", "description", "A\n\n\tB"), ptr: ok},

		// D4 item 9: the guide, prose of 1 to 16 384 scalar values.
		{name: "guide 16385", change: to("", "guide", r("é", 16385)), ptr: "/guide", rule: ruleLength},
		{name: "guide 16384", change: to("", "guide", r("é", 16384)), ptr: ok},
		{name: "guide trailing line feed", change: to("", "guide", "# Guide\n"), ptr: "/guide", rule: ruleProse},
		{name: "guide an array", change: to("", "guide", []any{"a"}), ptr: "/guide", rule: ruleType},

		// D4 item 10: the authors.
		{name: "authors empty", change: to("", "authors", []any{}), ptr: "/authors", rule: ruleItems},
		{name: "authors 65", change: to("", "authors", authors(65)), ptr: "/authors", rule: ruleItems},
		{name: "authors 64", change: to("", "authors", authors(64)), ptr: ok},
		{name: "author unknown member", change: to("authors/1", "role", "editor"), ptr: "/authors/1/role", rule: ruleUnknown},
		{name: "author without name", change: to("authors/1", "name", del), ptr: "/authors/1", rule: ruleRequired},
		{name: "author not an object", change: to("", "authors", []any{"A. Person"}), ptr: "/authors/0", rule: ruleType},
		{name: "author name 201", change: to("authors/1", "name", r("n", 201)), ptr: "/authors/1/name", rule: ruleLength},
		{name: "email two @", change: to("authors/0", "email", "a@b@c"), ptr: "/authors/0/email", rule: ruleEmail},
		{name: "email no local part", change: to("authors/0", "email", "@example.org"), ptr: "/authors/0/email", rule: ruleEmail},
		{name: "email no domain", change: to("authors/0", "email", "acr@"), ptr: "/authors/0/email", rule: ruleEmail},
		{name: "email space", change: to("authors/0", "email", "a cr@example.org"), ptr: "/authors/0/email", rule: ruleEmail},
		{name: "email 255", change: to("authors/0", "email", r("a", 250)+"@b.cd"), ptr: "/authors/0/email", rule: ruleLength},
		{name: "email 3", change: to("authors/0", "email", "a@b"), ptr: ok},
		{name: "email 254", change: to("authors/0", "email", r("a", 249)+"@b.cd"), ptr: ok},
		{name: "url ftp", change: to("authors/0", "url", "ftp://example.org/"), ptr: "/authors/0/url", rule: ruleURI},

		// D4 item 11: the license's length; its grammar is Check's.
		{name: "license empty", change: to("", "license", ""), ptr: "/license", rule: ruleLength},
		{name: "license 513", change: to("", "license", "LicenseRef-"+r("a", 502)), ptr: "/license", rule: ruleLength},
		{name: "license 512", change: to("", "license", "LicenseRef-"+r("a", 501)), ptr: ok},
		{name: "license lower-case and", change: to("", "license", "MIT and Apache-2.0"), ptr: "/license", rule: string(CodeLicenseExpression), byCheck: true},
		{name: "license DocumentRef", change: to("", "license", "DocumentRef-spdx-tool-1.2:LicenseRef-MIT-Style-2"), ptr: "/license", rule: string(CodeLicenseExpression), byCheck: true},
		{name: "license unbalanced", change: to("", "license", "(MIT OR Apache-2.0"), ptr: "/license", rule: string(CodeLicenseExpression), byCheck: true},
		{name: "license only spaces", change: to("", "license", "   "), ptr: "/license", rule: string(CodeLicenseExpression), byCheck: true},
		{name: "license single identifier", change: to("", "license", "MIT"), ptr: ok},
		{name: "license or later", change: to("", "license", "GPL-2.0+"), ptr: ok},

		// D4 item 12: the homepage.
		{name: "homepage ftp", change: to("", "homepage", "ftp://example.org/"), ptr: "/homepage", rule: ruleURI},
		{name: "homepage relative", change: to("", "homepage", "/acr"), ptr: "/homepage", rule: ruleURI},
		{name: "homepage scheme-relative", change: to("", "homepage", "//example.org/"), ptr: "/homepage", rule: ruleURI},
		{name: "homepage userinfo", change: to("", "homepage", "https://user@example.org/"), ptr: "/homepage", rule: ruleURI},
		{name: "homepage userinfo with password", change: to("", "homepage", "https://user:pw@example.org/"), ptr: "/homepage", rule: ruleURI},
		{name: "homepage upper-case scheme", change: to("", "homepage", "HTTPS://example.org/"), ptr: "/homepage", rule: ruleURI},
		{name: "homepage empty host", change: to("", "homepage", "https:///acr"), ptr: "/homepage", rule: ruleURI},
		{name: "homepage port without host", change: to("", "homepage", "https://:8443/"), ptr: "/homepage", rule: ruleURI},
		{name: "homepage fragment", change: to("", "homepage", "https://example.org/#about"), ptr: "/homepage", rule: ruleURI},
		{name: "homepage space", change: to("", "homepage", "https://example.org/a b"), ptr: "/homepage", rule: ruleURI},
		{name: "homepage bad percent", change: to("", "homepage", "https://example.org/%zz"), ptr: "/homepage", rule: ruleURI},
		{name: "homepage bad IPv6", change: to("", "homepage", "https://[1:2:3]/"), ptr: "/homepage", rule: ruleURI},
		{name: "homepage 2049", change: to("", "homepage", "https://e.org/"+r("a", 2035)), ptr: "/homepage", rule: ruleLength},
		{name: "homepage 2048", change: to("", "homepage", "https://e.org/"+r("a", 2034)), ptr: ok},
		{name: "homepage http IPv4", change: to("", "homepage", "http://192.0.2.1/"), ptr: ok},
		{name: "homepage IPv6", change: to("", "homepage", "https://[2001:db8::1]:8080/a"), ptr: ok},
		{name: "homepage IPv4-mapped IPv6", change: to("", "homepage", "https://[::ffff:192.0.2.1]/"), ptr: ok},
		{name: "homepage percent-encoded", change: to("", "homepage", "https://example.org/a%20b?q=1&r=/x"), ptr: ok},
		{name: "homepage bare host", change: to("", "homepage", "https://example.org"), ptr: ok},

		// ADR-003 D6 item 2: input_constraints.
		{name: "constraints outside inputs", change: to("", "input_constraints", "constraints.schema.json"), ptr: "/input_constraints", rule: rulePath},
		{name: "constraints not a schema file", change: to("", "input_constraints", "inputs/constraints.md"), ptr: "/input_constraints", rule: rulePath},

		// ADR-003 D1: the inputs.
		{name: "inputs empty", change: to("", "inputs", []any{}), ptr: ok},
		{name: "inputs an object", change: to("", "inputs", tree{}), ptr: "/inputs", rule: ruleType},
		{name: "input not an object", change: to("", "inputs", []any{"product"}), ptr: "/inputs/0", rule: ruleType},
		{name: "input unknown member", change: to("inputs/0", "source", "the CRM"), ptr: "/inputs/0/source", rule: ruleUnknown},
		{name: "input without name", change: to("inputs/0", "name", del), ptr: "/inputs/0", rule: ruleRequired},
		{name: "input without title", change: to("inputs/0", "title", del), ptr: "/inputs/0", rule: ruleRequired},
		{name: "input without schema", change: to("inputs/0", "schema", del), ptr: "/inputs/0", rule: ruleRequired},
		{name: "input name upper case", change: to("inputs/0", "name", "Product"), ptr: "/inputs/0/name", rule: ruleIdentifier},
		{name: "input name hyphen", change: to("inputs/0", "name", "the-product"), ptr: "/inputs/0/name", rule: ruleIdentifier},
		{name: "input name dot", change: to("inputs/0", "name", "a.b"), ptr: "/inputs/0/name", rule: ruleIdentifier},
		{name: "input name leading digit", change: to("inputs/0", "name", "1st"), ptr: "/inputs/0/name", rule: ruleIdentifier},
		{name: "input name leading underscore", change: to("inputs/0", "name", "_x"), ptr: "/inputs/0/name", rule: ruleIdentifier},
		{name: "input name 65", change: to("inputs/0", "name", "a"+r("b", 64)), ptr: "/inputs/0/name", rule: ruleIdentifier},
		{name: "input name 64", change: to("inputs/0", "name", "a"+r("b", 63)), ptr: ok},
		{name: "input name size", change: to("inputs/0", "name", "size"), ptr: "/inputs/0/name", rule: ruleReserved},
		{name: "input name nil", change: to("inputs/0", "name", "nil"), ptr: "/inputs/0/name", rule: ruleReserved},
		{name: "input name sizes", change: to("inputs/0", "name", "sizes"), ptr: ok},
		{name: "input title line feed", change: to("inputs/0", "title", "A\nB"), ptr: "/inputs/0/title", rule: ruleLine},
		{name: "input title 201", change: to("inputs/0", "title", r("t", 201)), ptr: "/inputs/0/title", rule: ruleLength},
		{name: "input description empty", change: to("inputs/0", "description", ""), ptr: "/inputs/0/description", rule: ruleLength},
		{name: "input description long", change: to("inputs/0", "description", r("d", 5000)), ptr: ok},
		{name: "input schema outside inputs", change: to("inputs/0", "schema", "schemas/product.schema.json"), ptr: "/inputs/0/schema", rule: rulePath},
		{name: "input schema not JSON or YAML", change: to("inputs/0", "schema", "inputs/product.txt"), ptr: "/inputs/0/schema", rule: rulePath},
		{name: "input schema empty segment", change: to("inputs/0", "schema", "inputs//product.json"), ptr: "/inputs/0/schema", rule: rulePath},
		{name: "input schema yml", change: to("inputs/0", "schema", "inputs/sub/product.yml"), ptr: ok},
		{name: "input meaning not Markdown", change: to("inputs/0", "meaning", "inputs/product.txt"), ptr: "/inputs/0/meaning", rule: rulePath},
		{name: "forms empty", change: to("inputs/0", "forms", []any{}), ptr: "/inputs/0/forms", rule: ruleItems},
		{name: "forms unknown", change: to("inputs/0", "forms", []any{"xml"}), ptr: "/inputs/0/forms/0", rule: ruleEnum},
		{name: "forms repeated", change: to("inputs/0", "forms", []any{"json", "json"}), ptr: "/inputs/0/forms/1", rule: ruleUnique},
		{name: "forms a string", change: to("inputs/0", "forms", "json"), ptr: "/inputs/0/forms", rule: ruleType},
		{name: "forms markdown and text", change: to("inputs/0", "forms", []any{"markdown", "text"}), ptr: ok},
		{name: "csv without forms", change: to("inputs/0", "csv", tree{"delimiter": ";"}), ptr: "/inputs/0/csv", rule: ruleCSVForms},
		{name: "csv without csv form", change: to("inputs/1", "forms", []any{"json"}), ptr: "/inputs/1/csv", rule: ruleCSVForms},
		{name: "csv without delimiter", change: to("inputs/1", "csv", tree{}), ptr: "/inputs/1/csv", rule: ruleRequired},
		{name: "csv colon", change: to("inputs/1", "csv", tree{"delimiter": ":"}), ptr: "/inputs/1/csv/delimiter", rule: ruleEnum},
		{name: "csv unknown member", change: to("inputs/1", "csv", tree{"delimiter": ",", "quote": "'"}), ptr: "/inputs/1/csv/quote", rule: ruleUnknown},
		{name: "csv semicolon", change: to("inputs/1", "csv", tree{"delimiter": ";"}), ptr: ok},
		{name: "csv pipe", change: to("inputs/1", "csv", tree{"delimiter": "|"}), ptr: ok},
		{name: "csv form without csv options", change: to("inputs/1", "csv", del), ptr: ok},
		{name: "collection a string", change: to("inputs/1", "collection", "yes"), ptr: "/inputs/1/collection", rule: ruleType},
		{name: "required a string", change: to("inputs/1", "required", "no"), ptr: "/inputs/1/required", rule: ruleType},
		{name: "input names repeated", change: to("inputs/1", "name", "product"), ptr: "/inputs/1/name", rule: string(CodeInputDuplicateName), byCheck: true},

		// ADR-005 D1: the outputs.
		{name: "outputs empty", change: to("", "outputs", []any{}), ptr: "/outputs", rule: ruleItems},
		{name: "outputs an object", change: to("", "outputs", tree{}), ptr: "/outputs", rule: ruleType},
		{name: "declaration not an object", change: to("", "outputs", []any{"report"}), ptr: "/outputs/0", rule: ruleType},
		{name: "declaration without source", change: to("outputs/0", "template", del), ptr: "/outputs/0", rule: ruleForm},
		{name: "declaration with two sources", change: to("outputs/0", "asset", "report.md"), ptr: "/outputs/0", rule: ruleForm},
		{name: "declaration without id", change: to("outputs/0", "id", del), ptr: "/outputs/0", rule: ruleRequired},
		{name: "id upper case", change: to("outputs/0", "id", "Report"), ptr: "/outputs/0/id", rule: ruleIdentifier},
		{name: "id hyphen", change: to("outputs/0", "id", "the-report"), ptr: "/outputs/0/id", rule: ruleIdentifier},
		{name: "id reserved word", change: to("outputs/0", "id", "size"), ptr: ok},
		{name: "declaration unknown field", change: to("outputs/0", "format", "md"), ptr: "/outputs/0/format", rule: ruleUnknown},
		{name: "top-level first", change: to("outputs/0", "first", true), ptr: "/outputs/0/first", rule: ruleUnknown},
		{name: "top-level stored", change: to("outputs/4", "stored", true), ptr: "/outputs/4/stored", rule: ruleUnknown},
		{name: "asset escape", change: to("outputs/4", "escape", "html"), ptr: "/outputs/4/escape", rule: ruleUnknown},
		{name: "asset newline", change: to("outputs/4", "newline", "crlf"), ptr: "/outputs/4/newline", rule: ruleUnknown},
		{name: "asset for_each", change: to("outputs/4", "for_each", "inputs.x"), ptr: "/outputs/4/for_each", rule: ruleUnknown},
		{name: "asset kind", change: to("outputs/4", "kind", "text"), ptr: "/outputs/4/kind", rule: ruleUnknown},
		{name: "mirror for_each", change: to("outputs/3", "for_each", "inputs.x"), ptr: "/outputs/3/for_each", rule: ruleUnknown},
		{name: "mirror validate", change: to("outputs/3", "validate", "references/x.json"), ptr: "/outputs/3/validate", rule: ruleUnknown},
		{name: "mirror executable", change: to("outputs/3", "executable", true), ptr: "/outputs/3/executable", rule: ruleUnknown},
		{name: "container escape", change: to("outputs/1", "escape", "html"), ptr: "/outputs/1/escape", rule: ruleUnknown},
		{name: "container executable", change: to("outputs/1", "executable", true), ptr: "/outputs/1/executable", rule: ruleUnknown},
		{name: "template for_each without path", change: to("outputs/2", "path", del), ptr: "/outputs/2", rule: ruleForEachPath},
		{name: "container without path", change: to("outputs/1", "path", del), ptr: "/outputs/1", rule: ruleRequired},
		{name: "template without path", change: to("outputs/0", "path", del), ptr: ok},
		{name: "path empty", change: to("outputs/0", "path", ""), ptr: "/outputs/0/path", rule: ruleLength},
		{name: "path a number", change: to("outputs/0", "path", 1), ptr: "/outputs/0/path", rule: ruleType},
		{name: "when empty", change: to("outputs/0", "when", ""), ptr: "/outputs/0/when", rule: ruleLength},
		{name: "when a boolean", change: to("outputs/0", "when", true), ptr: "/outputs/0/when", rule: ruleType},
		{name: "for_each empty", change: to("outputs/2", "for_each", ""), ptr: "/outputs/2/for_each", rule: ruleLength},
		{name: "escape unknown", change: to("outputs/0", "escape", "xml"), ptr: "/outputs/0/escape", rule: ruleEnum},
		{name: "newline unknown", change: to("outputs/0", "newline", "lf"), ptr: "/outputs/0/newline", rule: ruleEnum},
		{name: "escape none", change: to("outputs/0", "escape", "none"), ptr: ok},
		{name: "newline as-written", change: to("outputs/0", "newline", "as-written"), ptr: ok},
		{name: "validate under .compiled", change: to("outputs/0", "validate", ".compiled/schema.json"), ptr: "/outputs/0/validate", rule: rulePath},
		{name: "validate absolute", change: to("outputs/0", "validate", "/references/x.json"), ptr: "/outputs/0/validate", rule: rulePath},
		{name: "validate .compiled-like name", change: to("outputs/0", "validate", "references/.compiled/x.json"), ptr: ok},
		{name: "template leading slash", change: to("outputs/0", "template", "/report.md.liquid"), ptr: "/outputs/0/template", rule: rulePath},
		{name: "asset trailing slash", change: to("outputs/4", "asset", "static/"), ptr: "/outputs/4/asset", rule: rulePath},
		{name: "mirror empty segment", change: to("outputs/3", "mirror", "a//b"), ptr: "/outputs/3/mirror", rule: rulePath},
		{name: "mirror root", change: to("outputs/3", "mirror", "."), ptr: ok},
		{name: "asset empty", change: to("outputs/4", "asset", ""), ptr: "/outputs/4/asset", rule: rulePath},
		{name: "executable a string", change: to("outputs/4", "executable", "true"), ptr: "/outputs/4/executable", rule: ruleType},
		{name: "output ids repeated", change: to("outputs/1", "id", "report"), ptr: "/outputs/1/id", rule: string(CodeOutputDuplicateID), byCheck: true},

		// ADR-005 D7: the kind.
		{name: "kind unknown", change: to("outputs/0", "kind", "rtf"), ptr: "/outputs/0/kind", rule: ruleEnum},
		{name: "kind upper case", change: to("outputs/0", "kind", "HTML"), ptr: "/outputs/0/kind", rule: ruleEnum},
		{name: "kind hash", change: to("outputs/0", "kind", "hash"), ptr: ok},

		// ADR-005 D8: containers and their entries.
		{name: "container not an object", change: to("outputs/1", "container", "zip"), ptr: "/outputs/1/container", rule: ruleType},
		{name: "container unknown member", change: to("outputs/1/container", "level", 9), ptr: "/outputs/1/container/level", rule: ruleUnknown},
		{name: "container without entries", change: to("outputs/1/container", "entries", del), ptr: "/outputs/1/container", rule: ruleRequired},
		{name: "container entries empty", change: to("outputs/1/container", "entries", []any{}), ptr: "/outputs/1/container/entries", rule: ruleItems},
		{name: "container format tar", change: to("outputs/1/container", "format", "tar"), ptr: "/outputs/1/container/format", rule: ruleConst},
		{name: "container without format", change: to("outputs/1/container", "format", del), ptr: ok},
		{name: "entry executable", change: to("outputs/1/container/entries/1", "executable", true), ptr: "/outputs/1/container/entries/1/executable", rule: ruleUnknown},
		{name: "entry first a string", change: to("outputs/1/container/entries/0", "first", "yes"), ptr: "/outputs/1/container/entries/0/first", rule: ruleType},
		{name: "entry stored a number", change: to("outputs/1/container/entries/1", "stored", 1), ptr: "/outputs/1/container/entries/1/stored", rule: ruleType},
		{name: "entry id upper case", change: to("outputs/1/container/entries/1", "id", "Logo"), ptr: "/outputs/1/container/entries/1/id", rule: ruleIdentifier},
		{name: "entry without source", change: to("outputs/1/container/entries/1", "asset", del), ptr: "/outputs/1/container/entries/1", rule: ruleForm},
		{name: "entry container without path", change: to("outputs/1/container/entries/3", "path", del), ptr: "/outputs/1/container/entries/3", rule: ruleRequired},
		{name: "entry for_each without path", change: to("outputs/1/container/entries/2", "for_each", "inputs.x"), ptr: "/outputs/1/container/entries/2", rule: ruleForEachPath},
		{name: "entry for_each with path", change: both(to("outputs/1/container/entries/2", "for_each", "inputs.x"), to("outputs/1/container/entries/2", "path", "{{ item }}.json")), ptr: ok},
		{name: "entry ids repeated", change: to("outputs/1/container/entries/0", "id", "logo"), ptr: "/outputs/1/container/entries/1/id", rule: string(CodeOutputDuplicateID), byCheck: true},
		{name: "entry id as a top-level id", change: to("outputs/1/container/entries/1", "id", "report"), ptr: ok},
		{name: "nested for_each", change: both(to("outputs/5/container/entries/0", "for_each", "item.chapters"), to("outputs/5/container/entries/0", "path", "{{ item }}.xhtml")), ptr: "/outputs/5/container/entries/0/for_each", rule: string(CodeOutputNestedEach), byCheck: true},

		// ADR-005 D9: the profiles.
		{name: "profile unknown", change: to("outputs/1/container", "profile", "pdf"), ptr: "/outputs/1/container/profile", rule: ruleEnum},
		{name: "odf without mimetype", change: to("outputs/1/container/entries/3/container", "mimetype", del), ptr: "/outputs/1/container/entries/3/container", rule: ruleRequired},
		{name: "mimetype with ooxml", change: to("outputs/1/container", "mimetype", "application/zip"), ptr: "/outputs/1/container/mimetype", rule: ruleMimetype},
		{name: "mimetype without profile", change: both(to("outputs/1/container", "profile", del), to("outputs/1/container", "mimetype", "application/zip")), ptr: "/outputs/1/container/mimetype", rule: ruleMimetype},
		{name: "mimetype with space", change: to("outputs/1/container/entries/3/container", "mimetype", "text/plain; charset"), ptr: "/outputs/1/container/entries/3/container/mimetype", rule: ruleMimetype},
		{name: "mimetype without subtype", change: to("outputs/1/container/entries/3/container", "mimetype", "text"), ptr: "/outputs/1/container/entries/3/container/mimetype", rule: ruleMimetype},
		{name: "mimetype two slashes", change: to("outputs/1/container/entries/3/container", "mimetype", "a/b/c"), ptr: "/outputs/1/container/entries/3/container/mimetype", rule: ruleMimetype},
		{name: "mimetype non-ASCII", change: to("outputs/1/container/entries/3/container", "mimetype", "text/café"), ptr: "/outputs/1/container/entries/3/container/mimetype", rule: ruleMimetype},
		{name: "no profile", change: to("outputs/1/container", "profile", del), ptr: ok},
	}
	for _, m := range []string{"metais", "name", "version", "title", "description", "guide", "inputs", "outputs"} {
		c := ruleCase{name: "without " + m, change: to("", m, del), ptr: "", rule: ruleRequired}
		if m == "metais" {
			c.rule = string(CodeVersionMissing)
		}
		cs = append(cs, c)
	}
	return cs
}

// authors returns n authors.
func authors(n int) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = tree{"name": "Author"}
	}
	return out
}

// TestRules holds the model's rules to the manifest schema (W2-16.2,
// W2-16.3): for every case, Build accepts the manifest exactly when the
// schema accepts it and Check finds nothing, and when it refuses, it names
// the rule broken where it is broken. Each refusing case is a negative
// test of its rule, and a case beside it at the rule's bound is accepted.
func TestRules(t *testing.T) {
	isolate(t)
	for _, c := range ruleCases() {
		t.Run(c.name, func(t *testing.T) {
			m := base()
			c.change(t, m)
			text := encode(t, m)
			root, _ := parse(t, text)

			schemaOK, why := schemaAccepts(t, text)
			checked := Check(root, Source{File: ManifestFile})
			got, err := Build(root)

			wantSchema := c.ptr == ok || c.byCheck
			if schemaOK != wantSchema {
				t.Errorf("the schema accepts: %v, want %v (%s)", schemaOK, wantSchema, why)
			}
			if (len(checked) == 0) != (c.ptr == ok || !c.byCheck && !strings.HasPrefix(c.rule, "package.")) {
				t.Errorf("Check reported %v", codes(checked))
			}
			if (err == nil) != (schemaOK && len(checked) == 0) {
				t.Errorf("Build: %v; the schema accepts: %v; Check reported %v", err, schemaOK, codes(checked))
			}
			if c.ptr == ok {
				if err != nil || got == nil {
					t.Fatalf("Build refused: %v", err)
				}
				return
			}
			var inv *InvalidError
			if !errors.As(err, &inv) || !errors.Is(err, ErrInvalid) {
				t.Fatalf("Build: %v, want an *InvalidError", err)
			}
			if got != nil {
				t.Error("Build returned a model beside its refusal")
			}
			if want := (Violation{c.ptr, c.rule}); !slices.Contains(inv.Violations, want) {
				t.Errorf("Build's violations %v do not include %v", inv.Violations, want)
			}
		})
	}
}
