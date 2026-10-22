// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

// The usage codes other packages report by name.
const (
	CodeUsageBadValue          Code = "usage.bad_value"
	CodeUsageConflictingFlags  Code = "usage.conflicting_flags"
	CodeUsageDataDirUnreadable Code = "usage.data_dir_unreadable"
	CodeUsageMissingCommand    Code = "usage.missing_command"
	CodeUsageMissingOperand    Code = "usage.missing_operand"
	CodeUsageMissingFlag       Code = "usage.missing_flag"
	CodeUsagePackageFromStdin  Code = "usage.package_from_stdin"
	CodeUsageNameInvalid       Code = "usage.name_invalid"
	CodeUsageNameRequired      Code = "usage.name_required"
	CodeUsageNotPackedPackage  Code = "usage.not_a_packed_package"
	CodeUsageRepeatedFlag      Code = "usage.repeated_flag"
	CodeUsageUnexpectedOperand Code = "usage.unexpected_operand"
	CodeUsageUnknownCommand    Code = "usage.unknown_command"
	CodeUsageUnknownFlag       Code = "usage.unknown_flag"
	CodeUsageUnknownTopic      Code = "usage.unknown_topic"
	CodeUsageFlagExcluded      Code = "usage.flag_excluded"
	CodeUsageBadGlob           Code = "usage.bad_glob"
	CodeUsageBadPattern        Code = "usage.bad_pattern"
	CodeUsageBinaryToTerminal  Code = "usage.binary_to_terminal"
	CodeUsageEntryNotFound     Code = "usage.entry_not_found"
	CodeUsageIsADirectory      Code = "usage.is_a_directory"
	CodeUsageLineOutOfRange    Code = "usage.line_out_of_range"
	CodeUsageNoMatch           Code = "usage.no_match"
	CodeUsageNotText           Code = "usage.not_text"
	CodeUsageUnknownInput      Code = "usage.unknown_input"
	CodeUsageUnknownExample    Code = "usage.unknown_example"
	CodeUsageUnknownOutput     Code = "usage.unknown_output"
	CodeUsageArchiveToTerminal Code = "usage.archive_to_terminal"
	CodeUsageNotSourceDir      Code = "usage.not_a_source_directory"
	CodeUsageFlagRequires      Code = "usage.flag_requires"
	CodeUsageHarnessRequired   Code = "usage.harness_required"
	CodeUsageNotRepositoryRoot Code = "usage.not_repository_root"
	CodeUsageSkillEdited       Code = "usage.skill_edited"
	CodeUsageUnexpandedTilde   Code = "usage.unexpanded_tilde"
)

// usageCodes are the usage family's registry entries, in code order.
var usageCodes = []Entry{
	{
		Code:     CodeUsageArchiveToTerminal,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of --out's value, -, among the arguments after metais.", Always: true},
		},
		Message:    "Argument {argument} writes an archive to standard output, which is a terminal, and an archive is not written to a terminal.",
		Hint:       "Redirect standard output to a file or a pipe; a render can instead select one file with --only <path>, whose bytes may go to a terminal.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageBadGlob,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the glob among the arguments after metais.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "Why the glob is refused: double_star (** that does not stand alone as a segment), class (a character class with no closing ], or a range whose ends are reversed), escape (a \\ with nothing after it), empty_segment (an empty glob, or one with a leading or doubled /), or encoding (it is not UTF-8).", Always: true},
		},
		Message:    "The glob at argument {argument} is refused ({reason}).",
		Hint:       "Run `metais help globs` for the grammar: ** stands alone as a segment, a class is closed by ], and there are no braces.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageBadPattern,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the pattern among the arguments after metais.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "Why the pattern is refused: syntax (it is not RE2 syntax as Go's regexp defines it), pattern_length (it is longer than the pattern-length limit), regex_program (its compiled program has more instructions than the regex-program limit), or regex_work (matching it against the lines searched needs more than the regex-work limit).", Always: true},
			{Name: "limit", Type: TypeString, Meaning: "The limit's name, as metais limits lists it, when the reason is a limit."},
			{Name: "bound", Type: TypeNumber, Meaning: "The limit's value in this command, its default or the lower value given with --limit, when the reason is a limit."},
		},
		Message:    "The grep pattern at argument {argument} is refused ({reason}).",
		Hint:       "Write the pattern in RE2 syntax (https://github.com/google/re2/wiki/Syntax), within the limits `metais limits` lists; a simpler pattern, or globs that select fewer entries, needs less work.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageBadValue,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag whose value is refused, written with its leading --.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "Why the value is refused: missing_value (the flag takes a value and none follows it), boolean_value (the flag takes no value and was written with =), or value_not_allowed (the value is not one the flag accepts).", Always: true},
			{Name: "allowed", Type: TypeArray, Items: TypeString, Meaning: "Every value the flag accepts, present when the flag accepts a closed set of values."},
		},
		Message:    "The value of {flag} at argument {argument} is refused ({reason}).",
		Hint:       "Give {flag} a value it accepts, as --name value or --name=value; a flag that takes no value is written alone, without =.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageBinaryToTerminal,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the entry's path among the arguments after metais.", Always: true},
		},
		Message:    "The entry at argument {argument} is binary, and a binary entry is not written to a terminal.",
		Hint:       "Redirect standard output to a file, or write the entry with `metais extract`; with --json, metais read prints it in base64.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageConflictingFlags,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position, among the arguments after metais, of the flag that conflicts with an earlier one.", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag, written with its leading --.", Always: true},
		},
		Message:    "{flag} at argument {argument} reads standard input, which an earlier {flag} already reads.",
		Hint:       "Read standard input once: give - to one {flag}, and the other inputs inline or in the data directory.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageDataDirUnreadable,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position, among the arguments after metais, of the --data flag.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "Why the data directory cannot be read: not_found, permission_denied, not_regular_file (it is not a directory), too_large (it holds more entries than the collection-items limit) or io_error.", Always: true},
		},
		Message:    "The data directory given at argument {argument} cannot be read ({reason}).",
		Hint:       "Give --data a directory that exists and that you can read, holding one file per input; run `metais inspect {package}` to see each input's file names.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageEntryNotFound,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the path among the arguments after metais.", Always: true},
		},
		Message:    "Argument {argument} names no entry and no directory of the package.",
		Hint:       "Run `metais ls <package>` for the entries: a path is matched byte for byte and case-sensitively, with / as its separator.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageFlagExcluded,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position, among the arguments after metais, of the flag refused.", Always: true},
			{Name: "command", Type: TypeString, Meaning: "The command the line names.", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag refused, written with its leading --.", Always: true},
			{Name: "other", Type: TypeString, Meaning: "The flag it cannot be given with, written with its leading --.", Always: true},
		},
		Message:    "{flag} at argument {argument} cannot be given with {other}.",
		Hint:       "Give one of {flag} and {other}; run `metais {command} --help` for how its flags combine.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageFlagRequires,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position, among the arguments after metais, of the flag refused.", Always: true},
			{Name: "command", Type: TypeString, Meaning: "The command the line names.", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag refused, written with its leading --.", Always: true},
			{Name: "requires", Type: TypeString, Meaning: "The flag it is accepted only with, written with its leading --.", Always: true},
		},
		Message:    "{flag} at argument {argument} is accepted only with {requires}.",
		Hint:       "Add {requires}, or leave out {flag}; run `metais {command} --help` for how its flags combine.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageHarnessRequired,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "found", Type: TypeString, Meaning: "Which of .claude/ and .agents/ the directory holds: neither, or both.", Always: true},
		},
		Message:    "metais skill needs direction: no --harness is given, and the directory holds {found} of .claude/ and .agents/, so it does not settle which harness to write the skill for.",
		Hint:       "Name the harness with --harness: claude, codex, gemini, cursor or opencode. Give --harness once for each of .claude/ and .agents/ to write both.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageIsADirectory,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the path among the arguments after metais.", Always: true},
			{Name: "command", Type: TypeString, Meaning: "The command that reads one entry.", Always: true},
		},
		Message:    "Argument {argument} names a directory of the package, and metais {command} reads one entry.",
		Hint:       "Run `metais ls <package>` with the directory for the entries beneath it.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageLineOutOfRange,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of --lines among the arguments after metais.", Always: true},
			{Name: "total", Type: TypeNumber, Meaning: "The number of lines the entry has.", Always: true},
		},
		Message:    "The range of --lines at argument {argument} starts beyond the entry's {total} lines.",
		Hint:       "Start the range at a line from 1 to the entry's last; an end beyond the last line is clamped to it.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:       CodeUsageMissingCommand,
		Severity:   SeverityError,
		Class:      ClassUsage,
		Subject:    SubjectInvocation,
		Message:    "No command was given; metais runs one command, named after any global flags.",
		Hint:       "To read a package, run `metais inspect <package>`: a package is read only with metais, in place, so do not unzip it or open it with any other zip tool or library. `metais help` lists every command.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "usage.missing_flag",
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "command", Type: TypeString, Meaning: "The command that requires the flag.", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag, written with its leading --.", Always: true},
		},
		Message:    "metais {command} requires the flag {flag}.",
		Hint:       "Run `metais {command} --help` for the flags it takes.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageMissingOperand,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "command", Type: TypeString, Meaning: "The command that requires the operand.", Always: true},
			{Name: "operand", Type: TypeString, Meaning: "The operand's placeholder, as the command's usage line writes it.", Always: true},
		},
		Message:    "metais {command} requires the operand {operand}.",
		Hint:       "Run `metais {command} --help` for the operands it takes.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageNameInvalid,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position, among the arguments after metais, of the argument the name was taken from: --name, or the directory operand.", Always: true},
			{Name: "source", Type: TypeString, Meaning: "Where the name was taken from: name, the --name flag; or directory, the last element of the directory operand.", Always: true},
			{Name: "suggestion", Type: TypeString, Meaning: "A name the rule accepts, made from the refused one by lower-casing it and writing each run of other characters as one hyphen; present when the name came from --name and one exists. A name from the directory is part of a path the caller gave, which is never printed."},
		},
		Message:    "The package name taken from argument {argument} does not satisfy the name rule.",
		Hint:       "Give a name with --name: lower-case ASCII letters and digits in runs joined by single hyphens, in segments joined by dots, at most 128 characters.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:       CodeUsageNameRequired,
		Severity:   SeverityError,
		Class:      ClassUsage,
		Subject:    SubjectInvocation,
		Message:    "metais init needs --name when the directory is omitted or its last element is . or ..; the name is never taken from the working directory.",
		Hint:       "Give the package's name with --name, such as `metais init --name acme-report`.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageNoMatch,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the glob among the arguments after metais.", Always: true},
		},
		Message:    "The glob at argument {argument} selects no entry of the package; nothing was written.",
		Hint:       "Run `metais ls <package> -r` for the entries, and `metais help globs` for the grammar: *, ? and ** never enter a segment that begins with a dot unless the glob names it.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageNotPackedPackage,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
		},
		Message:    "Argument {argument} names a source directory; metais unpack reads a packed package.",
		Hint:       "Give a packed package file. A source directory is already the editable tree unpack writes; nothing was written.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageNotSourceDir,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
		},
		Message:    "Argument {argument} names a packed package, and this needs a source directory.",
		Hint:       "Unpack it with metais unpack and give the source directory it writes; nothing was written.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:       CodeUsageNotRepositoryRoot,
		Severity:   SeverityWarning,
		Class:      ClassUsage,
		Subject:    SubjectInvocation,
		Message:    "The directory metais skill wrote into holds no .git entry: a harness started in it or below it finds the skill, and one started at the repository's top level does not.",
		Hint:       "To serve the whole repository, run metais skill at its top level. With --dir ~, for your own scope, this is expected.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageNotText,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the entry's path among the arguments after metais.", Always: true},
			{Name: "command", Type: TypeString, Meaning: "The command that reads the entry as lines.", Always: true},
		},
		Message:    "The entry at argument {argument} is binary, not valid UTF-8 with no NUL byte, and metais {command} reads lines of text.",
		Hint:       "Write the entry to a file with `metais extract`; metais read without --lines and --number prints its bytes.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsagePackageFromStdin,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
		},
		Message:    "Argument {argument} names standard input as the package; a package is read from a file.",
		Hint:       "Write the package to a file and give its path; a zip is read with random access, which standard input does not offer.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageRepeatedFlag,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position, among the arguments after metais, of the flag's second occurrence.", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag given more than once, written with its leading --.", Always: true},
		},
		Message:    "{flag} is given again at argument {argument}; it may be given only once.",
		Hint:       "Give {flag} once: a repeated flag is refused rather than letting the last one win.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageSkillEdited,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "path", Type: TypeString, Meaning: "The skill file's path, relative to the directory.", Always: true},
		},
		Message:    "The skill at {path} is not a copy metais wrote, or was edited since, so it was left as it is.",
		Hint:       "Compare it with `metais skill --print`, and give --overwrite to replace it.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnexpandedTilde,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of --dir's value among the arguments after metais.", Always: true},
		},
		Message:    "Argument {argument} gives --dir a path whose first element is a literal ~, which the shell did not expand.",
		Hint:       "A shell expands ~ only at the start of a word, so --dir=~ reaches metais unexpanded: write --dir ~ with a space, or the directory's path. A directory named ~ is written ./~.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnexpectedOperand,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
			{Name: "command", Type: TypeString, Meaning: "The command that does not take the operand.", Always: true},
		},
		Message:    "Argument {argument} is an operand that metais {command} does not take.",
		Hint:       "Run `metais {command} --help` for the operands it takes; after --, every argument is an operand.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnknownCommand,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
		},
		Message:    "Argument {argument} names no metais command; a command is never abbreviated and has no alias.",
		Hint:       "Run `metais help` for the commands.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     "usage.unknown_example",
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the example's name among the arguments after metais.", Always: true},
			{Name: "examples", Type: TypeArray, Items: TypeString, Meaning: "Every example of the package, in byte order of name.", Always: true},
		},
		Message:    "Argument {argument} names no example of the package.",
		Hint:       "Give one of the examples the finding lists, each a directory under examples/.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnknownFlag,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
			{Name: "reason", Type: TypeString, Meaning: "Why the argument is refused: unknown (it names no flag accepted where it is written; before the command name only the global flags are), abbreviated (it is a prefix of a flag's name, and no flag may be abbreviated), or grouped (one-letter aliases written together, which they never are).", Always: true},
			{Name: "flag", Type: TypeString, Meaning: "The flag the argument abbreviates, or whose one-letter alias begins the group, written with its leading --; present when the reason is abbreviated or grouped."},
			{Name: "command", Type: TypeString, Meaning: "The command whose flags the argument was read against, present when a command was named."},
		},
		Message:    "Argument {argument} is not a flag this command line accepts ({reason}).",
		Hint:       "Run `metais help` for the global flags, or the command's --help for its own; write each flag's name in full, and each one-letter alias on its own.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnknownInput,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the input's name among the arguments after metais.", Always: true},
			{Name: "inputs", Type: TypeArray, Items: TypeString, Meaning: "Every input the package declares, in the manifest's order.", Always: true},
		},
		Message:    "Argument {argument} names no input of the package.",
		Hint:       "Give one of the inputs the finding lists; `metais inspect <package>` shows each with what it means.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnknownOutput,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of --only's value among the arguments after metais.", Always: true},
			{Name: "paths", Type: TypeArray, Items: TypeString, Meaning: "The first 20 paths of the files the render writes, in byte order.", Always: true},
			{Name: "path_count", Type: TypeNumber, Meaning: "How many files the render writes.", Always: true},
		},
		Message:    "Argument {argument} names no file the render writes; {path_count} files are written.",
		Hint:       "Give --only one of the paths the finding lists; `metais render <package> --out <dir> --dry-run` lists every file with its path.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeUsageUnknownTopic,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "argument", Type: TypeNumber, Meaning: "The 1-based position of the refused argument among the arguments after metais.", Always: true},
		},
		Message:    "Argument {argument} names no command or help topic.",
		Hint:       "Run `metais help` for the commands and the topics.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
}
