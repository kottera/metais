// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"testing"
)

const fieldSchema = `{
  "properties": {
    "vendor": {"title": "Vendor", "description": "Who supplies it.", "$ref": "common.schema.json#/$defs/vendor"},
    "contact": {"$ref": "common.schema.json#/$defs/contact"},
    "remarks": {"title": "Remarks", "allOf": [{"minLength": 10}]},
    "address": {"title": "Address", "type": "object", "properties": {"street": {"type": "string"}}},
    "password": {"type": "string", "writeOnly": true, "minLength": 8},
    "pin": {"writeOnly": true, "allOf": [{"minimum": 1000}]},
    "colour": {"writeOnly": true, "enum": ["red", "green"]}
  }
}`

const commonSchema = `{
  "$defs": {
    "vendor": {"description": "The target's description.", "type": "object"},
    "contact": {
      "title": "Contact", "description": "How to reach them.", "type": "object",
      "properties": {"email": {"$ref": "#/$defs/email"}},
      "required": ["email", "phone"]
    },
    "email": {"title": "Email address", "format": "email"}
  }
}`

func ref(kl, sp string, inst []string, children ...Failure) Failure {
	return Failure{Keyword: "$ref", KeywordLocation: kl, Entry: "vendor.schema.json", SchemaPointer: sp, InstancePointer: inst, Children: children}
}

func common(kw, kl, sp string, inst []string, data Data) Failure {
	return Failure{Keyword: kw, KeywordLocation: kl, Entry: "common.schema.json", SchemaPointer: sp, InstancePointer: inst, Data: data}
}

// TestFieldLookup is ADR-006 D6 item 9 and the cases *Conformance* names:
// a title beside a $ref, a title only on the $ref's target, a failing
// keyword inside allOf under a titled property, and a parent's title not
// used for a child; the title and the description found independently; an
// absent member's field from properties/<name>, following its $ref chain.
func TestFieldLookup(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": fieldSchema, "common.schema.json": commonSchema})
	vendorType := common("type", "/properties/vendor/$ref/type", "/$defs/vendor/type", path("vendor"), Data{"want": []string{"object"}, "got": "string"})
	contactType := common("type", "/properties/contact/$ref/type", "/$defs/contact/type", path("contact"), Data{"want": []string{"object"}, "got": "string"})
	remarks := at("minLength", "/properties/remarks/allOf/0/minLength", path("remarks"), Data{"limit": 10, "length": 4})
	cases := []struct {
		name string
		f    Failure
		want []string
	}{
		{"a title beside the $ref, the description nearer",
			ref("/properties/vendor/$ref", "/properties/vendor/$ref", path("vendor"), vendorType),
			[]string{`input.type "/vendor" {"actual":"string","expected":["object"]} field=Vendor|The target's description.`}},
		{"a title only on the target", ref("/properties/contact/$ref", "/properties/contact/$ref", path("contact"), contactType),
			[]string{`input.type "/contact" {"actual":"string","expected":["object"]} field=Contact|How to reach them.`}},
		{"a title only on the target, with no node for the $ref", contactType,
			[]string{`input.type "/contact" {"actual":"string","expected":["object"]} field=Contact|How to reach them.`}},
		{"inside allOf under a titled property", at("allOf", "/properties/remarks/allOf", path("remarks"), nil, remarks),
			[]string{`input.min_length "/remarks" {"length":4,"limit":10} field=Remarks|`}},
		{"inside allOf, with no node for it", remarks,
			[]string{`input.min_length "/remarks" {"length":4,"limit":10} field=Remarks|`}},
		{"a parent's title is not a child's", at("properties", "/properties/address/properties", path("address"), nil,
			at("type", "/properties/address/properties/street/type", path("address", "street"), Data{"want": []string{"string"}, "got": "number"})),
			[]string{`input.type "/address/street" {"actual":"number","expected":["string"]}`}},
		{"absent members", ref("/properties/contact/$ref", "/properties/contact/$ref", path("contact"),
			common("required", "/properties/contact/$ref/required", "/$defs/contact/required", path("contact"), Data{"missing": []string{"email", "phone"}})),
			[]string{
				`input.required "/contact/email" {"property":"email"} field=Email address|`,
				`input.required "/contact/phone" {"property":"phone"}`,
			}},
	}
	for _, c := range cases {
		checkBriefs(t, c.name, mapAll(t, target(d), c.f), c.want...)
	}
}

// TestWriteOnly is ADR-006 D9 item 5: where the schema at the value's
// location is writeOnly, no value, no length and no suggestion appears.
func TestWriteOnly(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": fieldSchema, "common.schema.json": commonSchema})
	checkBriefs(t, "writeOnly", mapAll(t, target(d),
		at("minLength", "/properties/password/minLength", path("password"), Data{"limit": 8, "length": 4}),
		at("minimum", "/properties/pin/allOf/0/minimum", path("pin"), Data{"limit": 1000, "value": 12}),
		at("enum", "/properties/colour/enum", path("colour"), Data{"allowed": []string{`"red"`, `"green"`}, "got": "Red"}),
		// The control: not writeOnly, the same fields appear.
		at("minLength", "/properties/remarks/allOf/0/minLength", path("remarks"), Data{"limit": 10, "length": 4}),
	),
		`input.enum "/colour" {"allowed":["\"red\"","\"green\""],"allowed_count":2}`,
		`input.min_length "/password" {"limit":8}`,
		`input.minimum "/pin" {"limit":1000}`,
		`input.min_length "/remarks" {"length":4,"limit":10} field=Remarks|`,
	)
}
