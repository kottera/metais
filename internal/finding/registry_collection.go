// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

// collectionCodes are the collection family's registry entries, in code order.
var collectionCodes = []Entry{
	{
		Code:       "collection.as_file",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "The collection input {input} is supplied as a file, and a collection is a directory.",
		Hint:       "Make a directory named after the input and put one file in it per item; the file was not read.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-015"},
	},
	{
		Code:       "collection.directory_name",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "The collection {input} is supplied as a directory whose name has an extension.",
		Hint:       "Name the collection's directory after the input alone, with no extension; it was not read.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-015"},
	},
	{
		Code:     "collection.item_name",
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectInput,
		Params: []ParamSpec{
			{Name: "reason", Type: TypeString, Meaning: "Why the name cannot be an item's: encoding (it is not UTF-8) or character (it holds a backslash or a colon, which no file system a package is portable to names).", Always: true},
		},
		Message:    "An item of the collection {input} has a file name the tool does not read ({reason}).",
		Hint:       "Rename the file with a UTF-8 name that holds no backslash and no colon; the file was not read.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-031"},
	},
	{
		Code:     "collection.stem_duplicate",
		Severity: SeverityError,
		Class:    ClassContent,
		Subject:  SubjectInput,
		Params: []ParamSpec{
			{Name: "case", Type: TypeBoolean, Meaning: "Whether the two stems differ only in letter case, rather than being equal.", Always: true},
		},
		Message:    "Two items of the collection {input} have stems that are equal or differ only in letter case.",
		Hint:       "Rename one of them: the stem is the item's identity, and a case-insensitive file system cannot hold both. The later item was not read.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-016"},
	},
	{
		Code:       "collection.subdirectory",
		Severity:   SeverityError,
		Class:      ClassContent,
		Subject:    SubjectInput,
		Message:    "The collection {input} holds a subdirectory, and a collection's items are files.",
		Hint:       "Move the files up into the collection's directory, or remove the subdirectory; it was not read.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{"MTS-SUP-015"},
	},
}
