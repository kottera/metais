// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

// The keyword fields an adapter writes in Data, by Failure.Keyword. They
// are the contract between the adapter and Map: Map reads nothing else of
// a failure, and refuses a failure whose fields are missing or of the
// wrong type. A number is an int or a Number.
//
//	type                    want []string (the type names allowed), got string (the value's JSON type)
//	enum                    allowed []string (each allowed value's RFC 8785 text, in the schema's order);
//	                        got string (the value, only when it is a string; read for the suggestion, never reported)
//	const                   expected string (the value's RFC 8785 text)
//	required                missing []string
//	dependentRequired       missing []string, because string (the present member)
//	propertyNames           property string (one failure per member name refused); its children are not read
//	minProperties ...       limit, count (also maxProperties, minItems, maxItems, minContains, maxContains)
//	minLength, maxLength    limit, length (code points)
//	pattern                 pattern string
//	format                  format string
//	minimum ...             limit, value (also maximum, exclusiveMinimum, exclusiveMaximum)
//	multipleOf              divisor, value
//	uniqueItems             duplicates []int (two indexes, the earlier first)
//	uniqueKeys              duplicates []int (the first item with the key, then the item repeating it);
//	                        at []string (the tokens of the keyword's one pointer, absent when it has several);
//	                        keys []string (every key pointer, as the schema writes it)
//	coverage                missing string (the RFC 8785 text of one member of values no item holds; one failure per member);
//	                        keys []string (the key pointer, as the schema writes it)
//	anyOf, oneOf            branch_count int; for a oneOf more than one branch of which matched,
//	                        matched []int (two of them) and no children
//	false                   prefix_count int, when the false schema is items' (one failure per item refused)
//
// A false schema the instance reached is a failure with Keyword "false",
// SchemaPointer the false schema's own pointer and InstancePointer the
// value it refused: one per member under additionalProperties or
// unevaluatedProperties, one per item under items or unevaluatedItems.
// Every other node is a keyword. The applicators ($ref, $dynamicRef,
// allOf, anyOf, oneOf, not, if, then, else, properties, patternProperties,
// additionalProperties, unevaluatedProperties, items, prefixItems,
// unevaluatedItems, contains, propertyNames, dependentSchemas) carry their
// subschemas' failures as children; an adapter may leave out a node for
// any of them except anyOf, oneOf, not, if, then, else and propertyNames,
// whose failures Map tells apart by their node.
const (
	dataWant        = "want"
	dataGot         = "got"
	dataAllowed     = "allowed"
	dataExpected    = "expected"
	dataMissing     = "missing"
	dataBecause     = "because"
	dataProperty    = "property"
	dataLimit       = "limit"
	dataCount       = "count"
	dataLength      = "length"
	dataPattern     = "pattern"
	dataFormat      = "format"
	dataValue       = "value"
	dataDivisor     = "divisor"
	dataDuplicates  = "duplicates"
	dataAt          = "at"
	dataKeys        = "keys"
	dataBranchCount = "branch_count"
	dataMatched     = "matched"
	dataPrefixCount = "prefix_count"
)

// KeywordFalse is the Keyword of a failure of a false schema.
const KeywordFalse = "false"
