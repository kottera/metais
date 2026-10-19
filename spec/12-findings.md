# 12. Findings

**Normative.** Areas: FND.

Every problem a processor or an authoring tool reports is a finding. This
chapter specifies the finding model and its severities, the catalog of
registered codes, how a finding locates what it reports, and the report that
gathers every finding of one operation in a fixed order.

## 12.1 The code catalog

Every finding code is registered in [findings/registry.json](findings/registry.json),
the one source of this table. Each row gives the code's severity, its class,
the kind of subject its findings name, its parameters, the revision that added
it, and the statements it reports.

<!-- The code catalog below is generated from findings/registry.json by go generate ./internal/guarantees. Do not edit it by hand. -->
| Code | Severity | Class | Subject | Parameters | Since | Statements |
|---|---|---|---|---|---|---|
| `csv.after_quote` | error | content | input | — | 1.0 | — |
| `csv.bare_quote` | error | content | input | — | 1.0 | — |
| `csv.field_count` | error | content | input | `expected`, `found` | 1.0 | — |
| `csv.header_duplicate` | error | content | input | `ordinal`, `first` | 1.0 | — |
| `csv.header_empty` | error | content | input | `ordinal` | 1.0 | — |
| `csv.header_missing` | error | content | input | — | 1.0 | — |
| `csv.unterminated_quote` | error | content | input | — | 1.0 | — |
| `internal.error` | error | internal | invocation | `tool_version`, `phase` | 1.0 | — |
| `limit.exceeded` | error | content | input | `limit`, `bound`, `lowered`; optional: `observed` | 1.0 | — |
| `limit.invalid` | error | usage | invocation | `reason`; optional: `name`, `default`, `unit`, `value` | 1.0 | — |
| `limit.package` | error | refused | package | `limit`, `bound`, `lowered`; optional: `observed` | 1.0 | — |
| `package.container.archive_comment` | error | refused | package | `offset`, `length` | 1.0 | — |
| `package.container.crc_mismatch` | error | refused | package | `offset` | 1.0 | — |
| `package.container.data_descriptor` | error | refused | package | `offset` | 1.0 | — |
| `package.container.deflate_stream` | error | refused | package | `offset`, `reason` | 1.0 | — |
| `package.container.encrypted` | error | refused | package | `offset` | 1.0 | — |
| `package.container.entry_order` | error | refused | package | `offset` | 1.0 | — |
| `package.container.extra_field` | error | refused | package | `offset`, `length` | 1.0 | — |
| `package.container.field_value` | error | refused | package | `offset`, `record`, `header_field`, `value`, `expected` | 1.0 | — |
| `package.container.flags` | error | refused | package | `offset`, `value` | 1.0 | — |
| `package.container.header_mismatch` | error | refused | package | `offset`, `header_field` | 1.0 | — |
| `package.container.method` | error | refused | package | `offset`, `method` | 1.0 | — |
| `package.container.not_zip` | error | refused | package | — | 1.0 | — |
| `package.container.overlap_or_gap` | error | refused | package | `offset`; optional: `declared` | 1.0 | — |
| `package.container.prepended_data` | error | refused | package | `offset`; optional: `declared` | 1.0 | — |
| `package.container.second_end_record` | error | refused | package | `offset` | 1.0 | — |
| `package.container.size_mismatch` | error | refused | package | `offset`, `declared` | 1.0 | — |
| `package.container.trailing_data` | error | refused | package | `offset` | 1.0 | — |
| `package.container.zip64` | error | refused | package | `offset` | 1.0 | — |
| `package.name.case_collision` | error | refused | package | `offset`, `member`, `other` | 1.0 | — |
| `package.name.character` | error | refused | package | `offset`, `code_point` | 1.0 | — |
| `package.name.directory_entry` | error | refused | package | `offset` | 1.0 | — |
| `package.name.duplicate` | error | refused | package | `offset` | 1.0 | — |
| `package.name.encoding` | error | refused | package | `offset`, `reason` | 1.0 | — |
| `package.name.file_directory_conflict` | error | refused | package | `offset`, `member`, `other` | 1.0 | — |
| `package.name.not_nfc` | error | refused | package | `offset` | 1.0 | — |
| `package.name.not_portable` | error | refused | package | `offset`, `reason` | 1.0 | — |
| `package.name.shape` | error | refused | package | `offset`, `reason` | 1.0 | — |
| `report.truncated` | info | content | report | `listed`, `omitted` | 1.0 | — |
| `usage.bad_value` | error | usage | invocation | `argument`, `flag`, `reason`; optional: `allowed` | 1.0 | — |
| `usage.missing_command` | error | usage | invocation | — | 1.0 | — |
| `usage.repeated_flag` | error | usage | invocation | `argument`, `flag` | 1.0 | — |
| `usage.unexpected_operand` | error | usage | invocation | `argument`, `command` | 1.0 | — |
| `usage.unknown_command` | error | usage | invocation | `argument` | 1.0 | — |
| `usage.unknown_flag` | error | usage | invocation | `argument`, `reason`; optional: `flag`, `command` | 1.0 | — |
| `usage.unknown_topic` | error | usage | invocation | `argument` | 1.0 | — |
<!-- The end of the generated code catalog. -->
