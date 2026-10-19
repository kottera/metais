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
| `example.layout` | error | content | example | `reason` | 1.0 | [MTS-AUT-020](13-authoring.md#mts-aut-020) |
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
| `limit.package` | error | refused | package | `limit`, `bound`, `lowered`; optional: `observed` | 1.0 | [MTS-SEC-001](14-security.md#mts-sec-001) |
| `manifest.additional_item` | error | content | manifest | `index`, `prefix_count` | 1.0 | — |
| `manifest.additional_property` | error | content | manifest | `property`; optional: `suggestion` | 1.0 | [MTS-MAN-011](03-manifest.md#mts-man-011), [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.any_of` | error | content | manifest | `branch_count` | 1.0 | — |
| `manifest.const` | error | content | manifest | `expected` | 1.0 | — |
| `manifest.contains` | error | content | manifest | — | 1.0 | — |
| `manifest.dependent_required` | error | content | manifest | `property`, `because` | 1.0 | — |
| `manifest.enum` | error | content | manifest | `allowed`, `allowed_count`; optional: `suggestion` | 1.0 | [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.exclusive_maximum` | error | content | manifest | `limit`; optional: `value` | 1.0 | — |
| `manifest.exclusive_minimum` | error | content | manifest | `limit`; optional: `value` | 1.0 | — |
| `manifest.file.wrong_name` | error | content | manifest | — | 1.0 | [MTS-MAN-002](03-manifest.md#mts-man-002) |
| `manifest.forbidden` | error | content | manifest | optional: `property` | 1.0 | — |
| `manifest.format` | error | content | manifest | `format` | 1.0 | — |
| `manifest.guide_link_missing` | warning | content | manifest | `target` | 1.0 | [MTS-AUT-006](13-authoring.md#mts-aut-006) |
| `manifest.input.collection_shape` | error | content | manifest | — | 1.0 | [MTS-AUT-022](13-authoring.md#mts-aut-022) |
| `manifest.input.duplicate_name` | error | content | manifest | `name` | 1.0 | [MTS-MAN-016](03-manifest.md#mts-man-016) |
| `manifest.input.form_shape` | error | content | manifest | `form`, `types` | 1.0 | [MTS-AUT-022](13-authoring.md#mts-aut-022) |
| `manifest.json.duplicate_name` | error | content | manifest | `name` | 1.0 | [MTS-MAN-009](03-manifest.md#mts-man-009) |
| `manifest.json.encoding` | error | content | manifest | `reason`, `offset` | 1.0 | [MTS-MAN-009](03-manifest.md#mts-man-009) |
| `manifest.json.lone_surrogate` | error | content | manifest | — | 1.0 | [MTS-MAN-009](03-manifest.md#mts-man-009) |
| `manifest.json.noncharacter` | error | content | manifest | — | 1.0 | [MTS-MAN-009](03-manifest.md#mts-man-009) |
| `manifest.json.syntax` | error | content | manifest | `expected`, `found` | 1.0 | [MTS-MAN-009](03-manifest.md#mts-man-009) |
| `manifest.json.trailing_content` | error | content | manifest | — | 1.0 | [MTS-MAN-009](03-manifest.md#mts-man-009) |
| `manifest.license.expression` | error | content | manifest | `reason`, `position` | 1.0 | [MTS-MAN-013](03-manifest.md#mts-man-013) |
| `manifest.max_contains` | error | content | manifest | `limit`, `count` | 1.0 | — |
| `manifest.max_items` | error | content | manifest | `limit`, `count` | 1.0 | — |
| `manifest.max_length` | error | content | manifest | `limit`; optional: `length` | 1.0 | [MTS-MAN-012](03-manifest.md#mts-man-012), [MTS-MAN-014](03-manifest.md#mts-man-014), [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.max_properties` | error | content | manifest | `limit`, `count` | 1.0 | — |
| `manifest.maximum` | error | content | manifest | `limit`; optional: `value` | 1.0 | — |
| `manifest.min_contains` | error | content | manifest | `limit`, `count` | 1.0 | — |
| `manifest.min_items` | error | content | manifest | `limit`, `count` | 1.0 | [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.min_length` | error | content | manifest | `limit`; optional: `length` | 1.0 | — |
| `manifest.min_properties` | error | content | manifest | `limit`, `count` | 1.0 | — |
| `manifest.minimum` | error | content | manifest | `limit`; optional: `value` | 1.0 | — |
| `manifest.multiple_of` | error | content | manifest | `divisor`; optional: `value` | 1.0 | — |
| `manifest.not` | error | content | manifest | — | 1.0 | [MTS-MAN-014](03-manifest.md#mts-man-014), [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.number_out_of_range` | error | content | manifest | `reason` | 1.0 | [MTS-MAN-010](03-manifest.md#mts-man-010) |
| `manifest.one_of` | error | content | manifest | `branch_count` | 1.0 | — |
| `manifest.one_of.ambiguous` | error | content | manifest | `branches` | 1.0 | — |
| `manifest.output.duplicate_id` | error | content | manifest | `id` | 1.0 | [MTS-MAN-016](03-manifest.md#mts-man-016) |
| `manifest.output.nested_for_each` | error | content | manifest | — | 1.0 | [MTS-MAN-016](03-manifest.md#mts-man-016) |
| `manifest.path_not_found` | error | content | manifest | `directory` | 1.0 | [MTS-MAN-017](03-manifest.md#mts-man-017) |
| `manifest.pattern` | error | content | manifest | `pattern` | 1.0 | [MTS-MAN-012](03-manifest.md#mts-man-012), [MTS-MAN-014](03-manifest.md#mts-man-014), [MTS-MAN-019](03-manifest.md#mts-man-019), [MTS-VER-008](04-versions.md#mts-ver-008) |
| `manifest.property_name` | error | content | manifest | `property` | 1.0 | — |
| `manifest.required` | error | content | manifest | `property` | 1.0 | [MTS-MAN-011](03-manifest.md#mts-man-011), [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.type` | error | content | manifest | `expected`, `actual` | 1.0 | [MTS-MAN-011](03-manifest.md#mts-man-011), [MTS-MAN-019](03-manifest.md#mts-man-019), [MTS-VER-008](04-versions.md#mts-ver-008) |
| `manifest.unevaluated_item` | error | content | manifest | `index` | 1.0 | — |
| `manifest.unevaluated_property` | error | content | manifest | `property`; optional: `suggestion` | 1.0 | — |
| `manifest.unique_items` | error | content | manifest | `index`, `duplicate_of` | 1.0 | — |
| `manifest.yaml.alias` | error | content | manifest | — | 1.0 | [MTS-MAN-005](03-manifest.md#mts-man-005) |
| `manifest.yaml.anchor` | error | content | manifest | — | 1.0 | [MTS-MAN-005](03-manifest.md#mts-man-005) |
| `manifest.yaml.bom` | error | content | manifest | — | 1.0 | [MTS-MAN-003](03-manifest.md#mts-man-003) |
| `manifest.yaml.character` | error | content | manifest | `codepoint`, `reason` | 1.0 | [MTS-MAN-003](03-manifest.md#mts-man-003) |
| `manifest.yaml.directive` | error | content | manifest | — | 1.0 | [MTS-MAN-004](03-manifest.md#mts-man-004) |
| `manifest.yaml.duplicate_key` | error | content | manifest | — | 1.0 | [MTS-MAN-006](03-manifest.md#mts-man-006) |
| `manifest.yaml.encoding` | error | content | manifest | `reason`, `offset` | 1.0 | [MTS-MAN-003](03-manifest.md#mts-man-003) |
| `manifest.yaml.key_empty` | error | content | manifest | — | 1.0 | [MTS-MAN-006](03-manifest.md#mts-man-006) |
| `manifest.yaml.key_not_scalar` | error | content | manifest | — | 1.0 | [MTS-MAN-006](03-manifest.md#mts-man-006) |
| `manifest.yaml.merge_key` | error | content | manifest | — | 1.0 | [MTS-MAN-005](03-manifest.md#mts-man-005) |
| `manifest.yaml.multi_document` | error | content | manifest | — | 1.0 | [MTS-MAN-004](03-manifest.md#mts-man-004) |
| `manifest.yaml.root_not_mapping` | error | content | manifest | — | 1.0 | [MTS-MAN-004](03-manifest.md#mts-man-004) |
| `manifest.yaml.scalar_refused` | error | content | manifest | `form` | 1.0 | [MTS-MAN-007](03-manifest.md#mts-man-007) |
| `manifest.yaml.syntax` | error | content | manifest | — | 1.0 | [MTS-MAN-008](03-manifest.md#mts-man-008) |
| `manifest.yaml.tag` | error | content | manifest | — | 1.0 | [MTS-MAN-005](03-manifest.md#mts-man-005) |
| `markdown.frontmatter_body_field` | error | content | input | — | 1.0 | — |
| `markdown.frontmatter_not_mapping` | error | content | input | `actual` | 1.0 | — |
| `markdown.frontmatter_unclosed` | error | content | input | — | 1.0 | — |
| `output.additional_item` | error | content | output | `index`, `prefix_count` | 1.0 | — |
| `output.additional_property` | error | content | output | `property`; optional: `suggestion` | 1.0 | — |
| `output.any_of` | error | content | output | `branch_count` | 1.0 | — |
| `output.const` | error | content | output | `expected` | 1.0 | — |
| `output.contains` | error | content | output | — | 1.0 | — |
| `output.declaration.validate_kind` | error | content | output | `kind` | 1.0 | [MTS-AUT-023](13-authoring.md#mts-aut-023) |
| `output.dependent_required` | error | content | output | `property`, `because` | 1.0 | — |
| `output.destination.not_directory` | error | destination | invocation | — | 1.0 | — |
| `output.destination.not_empty` | error | destination | invocation | `entries`, `more` | 1.0 | — |
| `output.destination.write_failed` | error | destination | invocation | `reason`, `written` | 1.0 | — |
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
| `package.compiled.mismatch` | error | refused | package | — | 1.0 | [MTS-MAN-022](03-manifest.md#mts-man-022), [MTS-PKG-025](02-container.md#mts-pkg-025) |
| `package.compiled.missing` | error | refused | package | — | 1.0 | [MTS-PKG-024](02-container.md#mts-pkg-024) |
| `package.compiled.unknown_file` | error | refused | package | — | 1.0 | [MTS-PKG-024](02-container.md#mts-pkg-024) |
| `package.container.archive_comment` | error | refused | package | `offset`, `length` | 1.0 | [MTS-PKG-003](02-container.md#mts-pkg-003) |
| `package.container.crc_mismatch` | error | refused | package | `offset` | 1.0 | [MTS-PKG-011](02-container.md#mts-pkg-011) |
| `package.container.data_descriptor` | error | refused | package | `offset` | 1.0 | [MTS-PKG-007](02-container.md#mts-pkg-007) |
| `package.container.deflate_stream` | error | refused | package | `offset`, `reason` | 1.0 | [MTS-PKG-011](02-container.md#mts-pkg-011) |
| `package.container.encrypted` | error | refused | package | `offset` | 1.0 | [MTS-PKG-007](02-container.md#mts-pkg-007) |
| `package.container.entry_order` | error | refused | package | `offset` | 1.0 | [MTS-PKG-010](02-container.md#mts-pkg-010) |
| `package.container.extra_field` | error | refused | package | `offset`, `length` | 1.0 | [MTS-PKG-008](02-container.md#mts-pkg-008) |
| `package.container.field_value` | error | refused | package | `offset`, `record`, `header_field`, `value`, `expected` | 1.0 | [MTS-PKG-006](02-container.md#mts-pkg-006) |
| `package.container.flags` | error | refused | package | `offset`, `value` | 1.0 | [MTS-PKG-007](02-container.md#mts-pkg-007) |
| `package.container.header_mismatch` | error | refused | package | `offset`, `header_field` | 1.0 | [MTS-PKG-009](02-container.md#mts-pkg-009) |
| `package.container.method` | error | refused | package | `offset`, `method` | 1.0 | [MTS-PKG-007](02-container.md#mts-pkg-007) |
| `package.container.not_zip` | error | refused | package | — | 1.0 | [MTS-PKG-003](02-container.md#mts-pkg-003) |
| `package.container.overlap_or_gap` | error | refused | package | `offset`; optional: `declared` | 1.0 | [MTS-PKG-003](02-container.md#mts-pkg-003) |
| `package.container.prepended_data` | error | refused | package | `offset`; optional: `declared` | 1.0 | [MTS-PKG-003](02-container.md#mts-pkg-003) |
| `package.container.second_end_record` | error | refused | package | `offset` | 1.0 | [MTS-PKG-005](02-container.md#mts-pkg-005) |
| `package.container.size_mismatch` | error | refused | package | `offset`, `declared` | 1.0 | [MTS-PKG-011](02-container.md#mts-pkg-011) |
| `package.container.trailing_data` | error | refused | package | `offset` | 1.0 | [MTS-PKG-003](02-container.md#mts-pkg-003) |
| `package.container.zip64` | error | refused | package | `offset` | 1.0 | [MTS-PKG-004](02-container.md#mts-pkg-004) |
| `package.digest.entries_mismatch` | error | refused | package | `reason` | 1.0 | [MTS-PKG-028](02-container.md#mts-pkg-028) |
| `package.digest.mismatch` | error | refused | package | `listed`, `computed` | 1.0 | [MTS-PKG-029](02-container.md#mts-pkg-029) |
| `package.digest.not_canonical` | error | refused | package | `reason` | 1.0 | [MTS-PKG-027](02-container.md#mts-pkg-027) |
| `package.digest.package_mismatch` | error | refused | package | `listed`, `computed` | 1.0 | [MTS-PKG-030](02-container.md#mts-pkg-030) |
| `package.layout.compiled_in_source` | error | refused | package | — | 1.0 | [MTS-PKG-022](02-container.md#mts-pkg-022) |
| `package.layout.empty_directory` | error | refused | package | — | 1.0 | [MTS-PKG-023](02-container.md#mts-pkg-023) |
| `package.layout.missing_manifest` | error | refused | package | — | 1.0 | [MTS-PKG-020](02-container.md#mts-pkg-020) |
| `package.layout.non_regular_file` | error | refused | package | — | 1.0 | [MTS-PKG-023](02-container.md#mts-pkg-023) |
| `package.layout.not_packed` | warning | content | package | — | 1.0 | [MTS-AUT-004](13-authoring.md#mts-aut-004) |
| `package.layout.os_litter` | warning | content | package | — | 1.0 | [MTS-AUT-004](13-authoring.md#mts-aut-004) |
| `package.layout.unknown_top_level` | error | refused | package | — | 1.0 | [MTS-PKG-019](02-container.md#mts-pkg-019) |
| `package.name.case_collision` | error | refused | package | `member`, `other`; optional: `offset` | 1.0 | [MTS-PKG-017](02-container.md#mts-pkg-017) |
| `package.name.character` | error | refused | package | `code_point`; optional: `offset` | 1.0 | [MTS-PKG-015](02-container.md#mts-pkg-015) |
| `package.name.directory_entry` | error | refused | package | optional: `offset` | 1.0 | [MTS-PKG-018](02-container.md#mts-pkg-018) |
| `package.name.duplicate` | error | refused | package | optional: `offset` | 1.0 | [MTS-PKG-017](02-container.md#mts-pkg-017) |
| `package.name.encoding` | error | refused | package | `reason`; optional: `offset` | 1.0 | [MTS-PKG-012](02-container.md#mts-pkg-012) |
| `package.name.file_directory_conflict` | error | refused | package | `member`, `other`; optional: `offset` | 1.0 | [MTS-PKG-017](02-container.md#mts-pkg-017) |
| `package.name.not_nfc` | error | refused | package | optional: `offset` | 1.0 | [MTS-PKG-013](02-container.md#mts-pkg-013) |
| `package.name.not_portable` | error | refused | package | `reason`; optional: `offset` | 1.0 | [MTS-PKG-016](02-container.md#mts-pkg-016) |
| `package.name.shape` | error | refused | package | `reason`; optional: `offset` | 1.0 | [MTS-PKG-014](02-container.md#mts-pkg-014) |
| `package.not_found` | error | refused | package | — | 1.0 | — |
| `package.not_readable` | error | refused | package | `reason` | 1.0 | — |
| `package.version.malformed` | error | refused | package | `implemented` | 1.0 | [MTS-VER-003](04-versions.md#mts-ver-003) |
| `package.version.missing` | error | refused | package | `implemented` | 1.0 | [MTS-VER-001](04-versions.md#mts-ver-001) |
| `package.version.not_string` | error | refused | package | `actual`, `implemented`; optional: `written` | 1.0 | [MTS-VER-002](04-versions.md#mts-ver-002) |
| `package.version.unsupported_major` | error | refused | package | `declared`, `implemented` | 1.0 | [MTS-VER-004](04-versions.md#mts-ver-004) |
| `package.version.unsupported_minor` | error | refused | package | `declared`, `implemented` | 1.0 | [MTS-VER-005](04-versions.md#mts-ver-005) |
| `report.step_skipped` | info | content | report | `steps` | 1.0 | [MTS-AUT-002](13-authoring.md#mts-aut-002) |
| `report.truncated` | info | content | report | `listed`, `omitted` | 1.0 | — |
| `schema.dialect` | error | content | package | `reason` | 1.0 | [MTS-DAT-003](05-data-specification.md#mts-dat-003) |
| `schema.format_unsupported` | error | content | package | `format` | 1.0 | [MTS-DAT-005](05-data-specification.md#mts-dat-005) |
| `schema.id_duplicate` | error | content | package | `id`; optional: `other` | 1.0 | [MTS-DAT-008](05-data-specification.md#mts-dat-008) |
| `schema.invalid` | error | content | package | `keyword` | 1.0 | [MTS-DAT-003](05-data-specification.md#mts-dat-003) |
| `schema.invalid_default` | error | content | package | — | 1.0 | [MTS-AUT-007](13-authoring.md#mts-aut-007) |
| `schema.invalid_example` | error | content | package | — | 1.0 | [MTS-AUT-007](13-authoring.md#mts-aut-007) |
| `schema.keyword_unknown` | error | content | package | `keyword` | 1.0 | [MTS-DAT-004](05-data-specification.md#mts-dat-004) |
| `schema.pattern_unsupported` | error | content | package | `construct` | 1.0 | [MTS-DAT-006](05-data-specification.md#mts-dat-006) |
| `schema.ref_by_path` | error | content | package | `reference`, `id` | 1.0 | [MTS-DAT-008](05-data-specification.md#mts-dat-008) |
| `schema.ref_escapes` | error | content | package | `reference` | 1.0 | [MTS-DAT-007](05-data-specification.md#mts-dat-007) |
| `schema.ref_unresolved` | error | content | package | `reference` | 1.0 | [MTS-DAT-007](05-data-specification.md#mts-dat-007) |
| `schema.required_not_declared` | warning | content | package | `name` | 1.0 | [MTS-AUT-009](13-authoring.md#mts-aut-009) |
| `schema.sample_unsatisfiable` | error | content | input | `instance`, `reason` | 1.0 | — |
| `schema.syntax` | error | content | package | `form`, `rule` | 1.0 | [MTS-DAT-002](05-data-specification.md#mts-dat-002) |
| `schema.unsatisfiable` | error | content | package | — | 1.0 | [MTS-AUT-008](13-authoring.md#mts-aut-008) |
| `schema.unusable_enum_member` | warning | content | package | — | 1.0 | [MTS-AUT-008](13-authoring.md#mts-aut-008) |
| `template.crlf` | warning | content | template | — | 1.0 | [MTS-AUT-010](13-authoring.md#mts-aut-010) |
| `template.division_by_zero` | error | content | template | `filter`, `column` | 1.0 | [MTS-MAN-018](03-manifest.md#mts-man-018) |
| `template.filter_argument` | error | content | template | `filter`, `argument`, `column` | 1.0 | [MTS-MAN-018](03-manifest.md#mts-man-018) |
| `template.filter_input` | error | content | template | `filter`, `type`, `column` | 1.0 | — |
| `template.include` | error | content | template | `column` | 1.0 | [MTS-MAN-018](03-manifest.md#mts-man-018) |
| `template.not_utf8` | error | content | template | — | 1.0 | [MTS-AUT-010](13-authoring.md#mts-aut-010) |
| `template.number_range` | error | content | template | `column` | 1.0 | [MTS-MAN-018](03-manifest.md#mts-man-018) |
| `template.ooxml.split_construct` | error | content | template | `cause`, `text`, `text_elements` | 1.0 | [MTS-AUT-019](13-authoring.md#mts-aut-019) |
| `template.ooxml.split_delimiter` | error | content | template | `cause`, `delimiter`, `text`, `text_elements` | 1.0 | [MTS-AUT-019](13-authoring.md#mts-aut-019) |
| `template.syntax` | error | content | template | `column`, `cause` | 1.0 | [MTS-AUT-010](13-authoring.md#mts-aut-010), [MTS-AUT-011](13-authoring.md#mts-aut-011), [MTS-MAN-018](03-manifest.md#mts-man-018) |
| `template.tag_context` | error | content | template | `tag`, `column` | 1.0 | [MTS-MAN-018](03-manifest.md#mts-man-018) |
| `template.unknown_filter` | error | content | template | `filter`, `column` | 1.0 | [MTS-MAN-018](03-manifest.md#mts-man-018) |
| `template.xref.budget_exhausted` | warning | content | package | `bound`, `limit` | 1.0 | [MTS-AUT-018](13-authoring.md#mts-aut-018) |
| `template.xref.dynamic_reference` | warning | content | template | optional: `output` | 1.0 | [MTS-AUT-014](13-authoring.md#mts-aut-014) |
| `template.xref.for_each_not_array` | error | content | output | `types` | 1.0 | [MTS-AUT-016](13-authoring.md#mts-aut-016) |
| `template.xref.index_out_of_range` | error | content | template | `index`; optional: `input`, `output` | 1.0 | [MTS-AUT-014](13-authoring.md#mts-aut-014) |
| `template.xref.missing_asset` | error | content | template | `asset` | 1.0 | — |
| `template.xref.missing_partial` | error | content | template | `partial`; optional: `output` | 1.0 | [MTS-AUT-015](13-authoring.md#mts-aut-015) |
| `template.xref.not_an_array` | error | content | template | `types`; optional: `input`, `output` | 1.0 | [MTS-AUT-014](13-authoring.md#mts-aut-014) |
| `template.xref.not_an_object` | error | content | template | `property`, `types`; optional: `input`, `output` | 1.0 | [MTS-AUT-013](13-authoring.md#mts-aut-013) |
| `template.xref.partial_parameter_not_passed` | warning | content | template | `name`; optional: `output` | 1.0 | [MTS-AUT-015](13-authoring.md#mts-aut-015) |
| `template.xref.unknown_field` | error | content | template | `property`, `declared`, `declared_count`; optional: `suggestion`, `input`, `output` | 1.0 | [MTS-AUT-013](13-authoring.md#mts-aut-013) |
| `template.xref.unknown_input` | error | content | template | `name`; optional: `suggestion` | 1.0 | — |
| `template.xref.unknown_variable` | error | content | template | `name`; optional: `suggestion`, `output` | 1.0 | [MTS-AUT-012](13-authoring.md#mts-aut-012) |
| `template.xref.unused_input` | warning | content | input | — | 1.0 | [MTS-AUT-017](13-authoring.md#mts-aut-017) |
| `template.xref.unused_template` | warning | content | template | — | 1.0 | [MTS-AUT-017](13-authoring.md#mts-aut-017) |
| `usage.bad_value` | error | usage | invocation | `argument`, `flag`, `reason`; optional: `allowed` | 1.0 | — |
| `usage.missing_command` | error | usage | invocation | — | 1.0 | — |
| `usage.missing_operand` | error | usage | invocation | `command`, `operand` | 1.0 | — |
| `usage.name_invalid` | error | usage | invocation | `argument`, `source`; optional: `suggestion` | 1.0 | — |
| `usage.name_required` | error | usage | invocation | — | 1.0 | — |
| `usage.package_from_stdin` | error | usage | invocation | `argument` | 1.0 | — |
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

## 12.2 The finding

A finding is one problem an operation reports. Its members are its `code`,
registered in §12.1; its `severity`, `error`, `warning` or `info`, which the
registry gives the code; its `subject`, the kind of thing it is about with
that thing's name where the kind has one; its `location` where the code's
registry entry gives it a span: the origin of the value or file, the file
relative to that origin, the 1-based line and column, in Unicode code points,
where the value starts, and the JSON Pointer of the value; and its `params`,
which the registry declares by name and type. A message and a hint come with
every finding; their wording is the implementation's own, and only the
members above are compared.

<a id="mts-fnd-001"></a>**MTS-FND-001** A processor MUST report each finding
with a code registered in §12.1, with the severity and the subject kind its
registry entry gives, with a location exactly when the entry gives the code a
span, and with the parameters the entry declares, each of its declared type and
each the entry marks as always present.

<a id="mts-fnd-002"></a>**MTS-FND-002** A processor MUST report that an
operation failed whenever it reports a finding of severity `error`, and MUST
then neither use nor print anything of what it was refusing.

<a id="mts-fnd-003"></a>**MTS-FND-003** A processor MUST report an
operation's findings in one order, compared member by member, strings by their
UTF-8 bytes: the subject's kind, in the order `invocation`, `package`,
`manifest`, `schema`, `input`, `template`, `asset`, `reference`, `output`,
`example`, `report`; its name, an absent name first; the location's origin, in
the order `package`, `example`, `data_dir`, `inputs_json`, `set`, `default`,
`sample`, `output`; its file, an absent file first, then its argument; its
line, then its column, a finding with no line first; its pointer, token by
token, an array index comparing as a number and a prefix before its
extensions; the code; the schema's entry, then its pointer; and the canonical
JSON of the parameters. Of two findings equal in code, subject, origin, file,
argument, pointer and parameters, it MUST report one.
