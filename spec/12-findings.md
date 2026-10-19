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
| `collection.as_file` | error | content | input | — | 1.0 | — |
| `collection.directory_name` | error | content | input | — | 1.0 | — |
| `collection.item_name` | error | content | input | `reason` | 1.0 | — |
| `collection.stem_duplicate` | error | content | input | `case` | 1.0 | — |
| `collection.subdirectory` | error | content | input | — | 1.0 | — |
| `csv.after_quote` | error | content | input | — | 1.0 | — |
| `csv.bare_quote` | error | content | input | — | 1.0 | — |
| `csv.field_count` | error | content | input | `expected`, `found` | 1.0 | — |
| `csv.header_duplicate` | error | content | input | `ordinal`, `first` | 1.0 | — |
| `csv.header_empty` | error | content | input | `ordinal` | 1.0 | — |
| `csv.header_missing` | error | content | input | — | 1.0 | — |
| `csv.unterminated_quote` | error | content | input | — | 1.0 | — |
| `input.additional_item` | error | content | input | `index`, `prefix_count` | 1.0 | — |
| `input.additional_property` | error | content | input | `property`; optional: `suggestion` | 1.0 | — |
| `input.any_of` | error | content | input | `branch_count` | 1.0 | — |
| `input.const` | error | content | input | `expected` | 1.0 | — |
| `input.constraints_failed` | error | content | invocation | `keyword` | 1.0 | — |
| `input.constraints_not_evaluated` | info | content | invocation | — | 1.0 | — |
| `input.contains` | error | content | input | — | 1.0 | — |
| `input.dependent_required` | error | content | input | `property`, `because` | 1.0 | — |
| `input.encoding_invalid` | error | content | input | `reason`, `offset` | 1.0 | — |
| `input.enum` | error | content | input | `allowed`, `allowed_count`; optional: `suggestion` | 1.0 | — |
| `input.exclusive_maximum` | error | content | input | `limit`; optional: `value` | 1.0 | — |
| `input.exclusive_minimum` | error | content | input | `limit`; optional: `value` | 1.0 | — |
| `input.extension_not_accepted` | error | content | input | `accepted` | 1.0 | — |
| `input.file_duplicate` | error | content | input | — | 1.0 | — |
| `input.file_unknown` | error | content | input | — | 1.0 | — |
| `input.forbidden` | error | content | input | optional: `property` | 1.0 | — |
| `input.format` | error | content | input | `format` | 1.0 | — |
| `input.json_arg_invalid` | error | content | invocation | `problem`; optional: `limit` | 1.0 | — |
| `input.json_arg_not_object` | error | content | invocation | `actual` | 1.0 | — |
| `input.json_arg_unknown` | error | content | input | — | 1.0 | — |
| `input.link` | error | content | input | — | 1.0 | — |
| `input.max_contains` | error | content | input | `limit`, `count` | 1.0 | — |
| `input.max_items` | error | content | input | `limit`, `count` | 1.0 | — |
| `input.max_length` | error | content | input | `limit`; optional: `length` | 1.0 | — |
| `input.max_properties` | error | content | input | `limit`, `count` | 1.0 | — |
| `input.maximum` | error | content | input | `limit`; optional: `value` | 1.0 | — |
| `input.min_contains` | error | content | input | `limit`, `count` | 1.0 | — |
| `input.min_items` | error | content | input | `limit`, `count` | 1.0 | — |
| `input.min_length` | error | content | input | `limit`; optional: `length` | 1.0 | — |
| `input.min_properties` | error | content | input | `limit`, `count` | 1.0 | — |
| `input.minimum` | error | content | input | `limit`; optional: `value` | 1.0 | — |
| `input.missing` | error | content | input | — | 1.0 | — |
| `input.multiple_of` | error | content | input | `divisor`; optional: `value` | 1.0 | — |
| `input.noncharacter` | error | content | input | `codepoint` | 1.0 | — |
| `input.not` | error | content | input | — | 1.0 | — |
| `input.not_readable` | error | content | input | `reason` | 1.0 | — |
| `input.not_regular_file` | error | content | input | `kind` | 1.0 | — |
| `input.number_out_of_range` | error | content | input | `reason` | 1.0 | — |
| `input.one_of` | error | content | input | `branch_count` | 1.0 | — |
| `input.one_of.ambiguous` | error | content | input | `branches` | 1.0 | — |
| `input.pattern` | error | content | input | `pattern` | 1.0 | — |
| `input.property_name` | error | content | input | `property` | 1.0 | — |
| `input.required` | error | content | input | `property` | 1.0 | — |
| `input.set_into_unparsed` | error | content | input | — | 1.0 | — |
| `input.set_pointer_invalid` | error | content | invocation | `reason` | 1.0 | — |
| `input.set_traversal` | error | content | input | `reason` | 1.0 | — |
| `input.type` | error | content | input | `expected`, `actual` | 1.0 | — |
| `input.unevaluated_item` | error | content | input | `index` | 1.0 | — |
| `input.unevaluated_property` | error | content | input | `property`; optional: `suggestion` | 1.0 | — |
| `input.unique_items` | error | content | input | `index`, `duplicate_of` | 1.0 | — |
| `input.validation.partial` | info | content | input | `unparsed`, `suppressed` | 1.0 | — |
| `internal.error` | error | internal | invocation | `tool_version`, `phase` | 1.0 | — |
| `json.duplicate_name` | error | content | input | `name` | 1.0 | — |
| `json.lone_surrogate` | error | content | input | — | 1.0 | — |
| `json.noncharacter` | error | content | input | — | 1.0 | — |
| `json.syntax` | error | content | input | `expected`, `found` | 1.0 | — |
| `json.trailing_content` | error | content | input | — | 1.0 | — |
| `limit.exceeded` | error | content | input | `limit`, `bound`, `lowered`; optional: `observed` | 1.0 | — |
| `limit.invalid` | error | usage | invocation | `reason`; optional: `name`, `default`, `unit`, `value` | 1.0 | — |
| `limit.package` | error | refused | package | `limit`, `bound`, `lowered`; optional: `observed` | 1.0 | — |
| `manifest.additional_item` | error | content | manifest | `index`, `prefix_count` | 1.0 | — |
| `manifest.additional_property` | error | content | manifest | `property`; optional: `suggestion` | 1.0 | — |
| `manifest.any_of` | error | content | manifest | `branch_count` | 1.0 | — |
| `manifest.const` | error | content | manifest | `expected` | 1.0 | — |
| `manifest.contains` | error | content | manifest | — | 1.0 | — |
| `manifest.dependent_required` | error | content | manifest | `property`, `because` | 1.0 | — |
| `manifest.enum` | error | content | manifest | `allowed`, `allowed_count`; optional: `suggestion` | 1.0 | — |
| `manifest.exclusive_maximum` | error | content | manifest | `limit`; optional: `value` | 1.0 | — |
| `manifest.exclusive_minimum` | error | content | manifest | `limit`; optional: `value` | 1.0 | — |
| `manifest.forbidden` | error | content | manifest | optional: `property` | 1.0 | — |
| `manifest.format` | error | content | manifest | `format` | 1.0 | — |
| `manifest.input.duplicate_name` | error | content | manifest | `name` | 1.0 | — |
| `manifest.license.expression` | error | content | manifest | `reason`, `position` | 1.0 | — |
| `manifest.max_contains` | error | content | manifest | `limit`, `count` | 1.0 | — |
| `manifest.max_items` | error | content | manifest | `limit`, `count` | 1.0 | — |
| `manifest.max_length` | error | content | manifest | `limit`; optional: `length` | 1.0 | — |
| `manifest.max_properties` | error | content | manifest | `limit`, `count` | 1.0 | — |
| `manifest.maximum` | error | content | manifest | `limit`; optional: `value` | 1.0 | — |
| `manifest.min_contains` | error | content | manifest | `limit`, `count` | 1.0 | — |
| `manifest.min_items` | error | content | manifest | `limit`, `count` | 1.0 | — |
| `manifest.min_length` | error | content | manifest | `limit`; optional: `length` | 1.0 | — |
| `manifest.min_properties` | error | content | manifest | `limit`, `count` | 1.0 | — |
| `manifest.minimum` | error | content | manifest | `limit`; optional: `value` | 1.0 | — |
| `manifest.multiple_of` | error | content | manifest | `divisor`; optional: `value` | 1.0 | — |
| `manifest.not` | error | content | manifest | — | 1.0 | — |
| `manifest.number_out_of_range` | error | content | manifest | `reason` | 1.0 | — |
| `manifest.one_of` | error | content | manifest | `branch_count` | 1.0 | — |
| `manifest.one_of.ambiguous` | error | content | manifest | `branches` | 1.0 | — |
| `manifest.output.duplicate_id` | error | content | manifest | `id` | 1.0 | — |
| `manifest.output.nested_for_each` | error | content | manifest | — | 1.0 | — |
| `manifest.pattern` | error | content | manifest | `pattern` | 1.0 | — |
| `manifest.property_name` | error | content | manifest | `property` | 1.0 | — |
| `manifest.required` | error | content | manifest | `property` | 1.0 | — |
| `manifest.type` | error | content | manifest | `expected`, `actual` | 1.0 | — |
| `manifest.unevaluated_item` | error | content | manifest | `index` | 1.0 | — |
| `manifest.unevaluated_property` | error | content | manifest | `property`; optional: `suggestion` | 1.0 | — |
| `manifest.unique_items` | error | content | manifest | `index`, `duplicate_of` | 1.0 | — |
| `manifest.yaml.alias` | error | content | manifest | — | 1.0 | — |
| `manifest.yaml.anchor` | error | content | manifest | — | 1.0 | — |
| `manifest.yaml.bom` | error | content | manifest | — | 1.0 | — |
| `manifest.yaml.character` | error | content | manifest | `codepoint`, `reason` | 1.0 | — |
| `manifest.yaml.directive` | error | content | manifest | — | 1.0 | — |
| `manifest.yaml.duplicate_key` | error | content | manifest | — | 1.0 | — |
| `manifest.yaml.encoding` | error | content | manifest | `reason`, `offset` | 1.0 | — |
| `manifest.yaml.key_empty` | error | content | manifest | — | 1.0 | — |
| `manifest.yaml.key_not_scalar` | error | content | manifest | — | 1.0 | — |
| `manifest.yaml.merge_key` | error | content | manifest | — | 1.0 | — |
| `manifest.yaml.multi_document` | error | content | manifest | — | 1.0 | — |
| `manifest.yaml.root_not_mapping` | error | content | manifest | — | 1.0 | — |
| `manifest.yaml.scalar_refused` | error | content | manifest | `form` | 1.0 | — |
| `manifest.yaml.syntax` | error | content | manifest | — | 1.0 | — |
| `manifest.yaml.tag` | error | content | manifest | — | 1.0 | — |
| `markdown.frontmatter_body_field` | error | content | input | — | 1.0 | — |
| `markdown.frontmatter_not_mapping` | error | content | input | `actual` | 1.0 | — |
| `markdown.frontmatter_unclosed` | error | content | input | — | 1.0 | — |
| `output.additional_item` | error | content | output | `index`, `prefix_count` | 1.0 | — |
| `output.additional_property` | error | content | output | `property`; optional: `suggestion` | 1.0 | — |
| `output.any_of` | error | content | output | `branch_count` | 1.0 | — |
| `output.const` | error | content | output | `expected` | 1.0 | — |
| `output.contains` | error | content | output | — | 1.0 | — |
| `output.dependent_required` | error | content | output | `property`, `because` | 1.0 | — |
| `output.enum` | error | content | output | `allowed`, `allowed_count`; optional: `suggestion` | 1.0 | — |
| `output.exclusive_maximum` | error | content | output | `limit`; optional: `value` | 1.0 | — |
| `output.exclusive_minimum` | error | content | output | `limit`; optional: `value` | 1.0 | — |
| `output.forbidden` | error | content | output | optional: `property` | 1.0 | — |
| `output.format` | error | content | output | `format` | 1.0 | — |
| `output.max_contains` | error | content | output | `limit`, `count` | 1.0 | — |
| `output.max_items` | error | content | output | `limit`, `count` | 1.0 | — |
| `output.max_length` | error | content | output | `limit`; optional: `length` | 1.0 | — |
| `output.max_properties` | error | content | output | `limit`, `count` | 1.0 | — |
| `output.maximum` | error | content | output | `limit`; optional: `value` | 1.0 | — |
| `output.min_contains` | error | content | output | `limit`, `count` | 1.0 | — |
| `output.min_items` | error | content | output | `limit`, `count` | 1.0 | — |
| `output.min_length` | error | content | output | `limit`; optional: `length` | 1.0 | — |
| `output.min_properties` | error | content | output | `limit`, `count` | 1.0 | — |
| `output.minimum` | error | content | output | `limit`; optional: `value` | 1.0 | — |
| `output.multiple_of` | error | content | output | `divisor`; optional: `value` | 1.0 | — |
| `output.not` | error | content | output | — | 1.0 | — |
| `output.one_of` | error | content | output | `branch_count` | 1.0 | — |
| `output.one_of.ambiguous` | error | content | output | `branches` | 1.0 | — |
| `output.pattern` | error | content | output | `pattern` | 1.0 | — |
| `output.property_name` | error | content | output | `property` | 1.0 | — |
| `output.required` | error | content | output | `property` | 1.0 | — |
| `output.type` | error | content | output | `expected`, `actual` | 1.0 | — |
| `output.unevaluated_item` | error | content | output | `index` | 1.0 | — |
| `output.unevaluated_property` | error | content | output | `property`; optional: `suggestion` | 1.0 | — |
| `output.unique_items` | error | content | output | `index`, `duplicate_of` | 1.0 | — |
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
| `package.version.malformed` | error | refused | package | `implemented` | 1.0 | — |
| `package.version.missing` | error | refused | package | `implemented` | 1.0 | — |
| `package.version.not_string` | error | refused | package | `actual`, `implemented`; optional: `written` | 1.0 | — |
| `package.version.unsupported_major` | error | refused | package | `declared`, `implemented` | 1.0 | — |
| `package.version.unsupported_minor` | error | refused | package | `declared`, `implemented` | 1.0 | — |
| `report.truncated` | info | content | report | `listed`, `omitted` | 1.0 | — |
| `schema.dialect` | error | content | package | `reason` | 1.0 | — |
| `schema.format_unsupported` | error | content | package | `format` | 1.0 | — |
| `schema.id_duplicate` | error | content | package | `id`; optional: `other` | 1.0 | — |
| `schema.invalid` | error | content | package | `keyword` | 1.0 | — |
| `schema.keyword_unknown` | error | content | package | `keyword` | 1.0 | — |
| `schema.pattern_unsupported` | error | content | package | `construct` | 1.0 | — |
| `schema.ref_by_path` | error | content | package | `reference`, `id` | 1.0 | — |
| `schema.ref_escapes` | error | content | package | `reference` | 1.0 | — |
| `schema.ref_unresolved` | error | content | package | `reference` | 1.0 | — |
| `template.xref.budget_exhausted` | warning | content | package | `bound`, `limit` | 1.0 | — |
| `template.xref.dynamic_reference` | warning | content | template | optional: `output` | 1.0 | — |
| `template.xref.for_each_not_array` | error | content | output | `types` | 1.0 | — |
| `template.xref.index_out_of_range` | error | content | template | `index`; optional: `input`, `output` | 1.0 | — |
| `template.xref.missing_asset` | error | content | template | `asset` | 1.0 | — |
| `template.xref.missing_partial` | error | content | template | `partial`; optional: `output` | 1.0 | — |
| `template.xref.not_an_array` | error | content | template | `types`; optional: `input`, `output` | 1.0 | — |
| `template.xref.not_an_object` | error | content | template | `property`, `types`; optional: `input`, `output` | 1.0 | — |
| `template.xref.partial_parameter_not_passed` | warning | content | template | `name`; optional: `output` | 1.0 | — |
| `template.xref.unknown_field` | error | content | template | `property`, `declared`, `declared_count`; optional: `suggestion`, `input`, `output` | 1.0 | — |
| `template.xref.unknown_input` | error | content | template | `name`; optional: `suggestion` | 1.0 | — |
| `template.xref.unknown_variable` | error | content | template | `name`; optional: `suggestion`, `output` | 1.0 | — |
| `template.xref.unused_input` | warning | content | input | — | 1.0 | — |
| `template.xref.unused_template` | warning | content | template | — | 1.0 | — |
| `usage.bad_value` | error | usage | invocation | `argument`, `flag`, `reason`; optional: `allowed` | 1.0 | — |
| `usage.missing_command` | error | usage | invocation | — | 1.0 | — |
| `usage.repeated_flag` | error | usage | invocation | `argument`, `flag` | 1.0 | — |
| `usage.unexpected_operand` | error | usage | invocation | `argument`, `command` | 1.0 | — |
| `usage.unknown_command` | error | usage | invocation | `argument` | 1.0 | — |
| `usage.unknown_flag` | error | usage | invocation | `argument`, `reason`; optional: `flag`, `command` | 1.0 | — |
| `usage.unknown_topic` | error | usage | invocation | `argument` | 1.0 | — |
| `yaml.alias` | error | content | input | — | 1.0 | — |
| `yaml.anchor` | error | content | input | — | 1.0 | — |
| `yaml.bom` | error | content | input | — | 1.0 | — |
| `yaml.character` | error | content | input | `codepoint`, `reason` | 1.0 | — |
| `yaml.directive` | error | content | input | — | 1.0 | — |
| `yaml.duplicate_key` | error | content | input | — | 1.0 | — |
| `yaml.key_empty` | error | content | input | — | 1.0 | — |
| `yaml.key_not_scalar` | error | content | input | — | 1.0 | — |
| `yaml.merge_key` | error | content | input | — | 1.0 | — |
| `yaml.multi_document` | error | content | input | — | 1.0 | — |
| `yaml.scalar_refused` | error | content | input | `form` | 1.0 | — |
| `yaml.syntax` | error | content | input | — | 1.0 | — |
| `yaml.tag` | error | content | input | — | 1.0 | — |
<!-- The end of the generated code catalog. -->
