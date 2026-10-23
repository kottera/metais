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
| `collection.as_file` | error | content | input | — | 1.0 | [MTS-SUP-015](06-supplying-inputs.md#mts-sup-015) |
| `collection.directory_name` | error | content | input | — | 1.0 | [MTS-SUP-015](06-supplying-inputs.md#mts-sup-015) |
| `collection.item_name` | error | content | input | `reason` | 1.0 | [MTS-SUP-031](06-supplying-inputs.md#mts-sup-031) |
| `collection.stem_duplicate` | error | content | input | `case` | 1.0 | [MTS-SUP-016](06-supplying-inputs.md#mts-sup-016) |
| `collection.subdirectory` | error | content | input | — | 1.0 | [MTS-SUP-015](06-supplying-inputs.md#mts-sup-015) |
| `csv.after_quote` | error | content | input | — | 1.0 | [MTS-SUP-008](06-supplying-inputs.md#mts-sup-008) |
| `csv.bare_quote` | error | content | input | — | 1.0 | [MTS-SUP-008](06-supplying-inputs.md#mts-sup-008) |
| `csv.field_count` | error | content | input | `expected`, `found` | 1.0 | [MTS-SUP-008](06-supplying-inputs.md#mts-sup-008) |
| `csv.header_duplicate` | error | content | input | `ordinal`, `first` | 1.0 | [MTS-SUP-008](06-supplying-inputs.md#mts-sup-008) |
| `csv.header_empty` | error | content | input | `ordinal` | 1.0 | [MTS-SUP-008](06-supplying-inputs.md#mts-sup-008) |
| `csv.header_missing` | error | content | input | — | 1.0 | [MTS-SUP-008](06-supplying-inputs.md#mts-sup-008) |
| `csv.unterminated_quote` | error | content | input | — | 1.0 | [MTS-SUP-008](06-supplying-inputs.md#mts-sup-008) |
| `example.entry_method` | error | content | example | `path`, `entry`, `expected_method`, `rendered_method` | 1.0 | [MTS-AUT-070](13-authoring.md#mts-aut-070) |
| `example.entry_missing` | error | content | example | `path`, `entry` | 1.0 | [MTS-AUT-067](13-authoring.md#mts-aut-067) |
| `example.entry_order` | error | content | example | `path`, `index`, `expected`, `rendered`; optional: `entry` | 1.0 | [MTS-AUT-068](13-authoring.md#mts-aut-068) |
| `example.entry_unexpected` | error | content | example | `path`, `entry` | 1.0 | [MTS-AUT-067](13-authoring.md#mts-aut-067) |
| `example.expected_unreadable` | error | content | example | `path`, `reason`; optional: `entry` | 1.0 | [MTS-AUT-066](13-authoring.md#mts-aut-066) |
| `example.layout` | error | content | example | `reason` | 1.0 | [MTS-AUT-020](13-authoring.md#mts-aut-020) |
| `example.metadata` | error | content | example | `reason`; optional: `member` | 1.0 | [MTS-AUT-036](13-authoring.md#mts-aut-036) |
| `example.output_differs` | error | content | example | `path`, `binary`, `cause`; optional: `entry`, `crlf`, `diff`, `diff_truncated`, `offset`, `expected_size`, `rendered_size`, `expected_digest`, `rendered_digest` | 1.0 | [MTS-AUT-040](13-authoring.md#mts-aut-040), [MTS-AUT-041](13-authoring.md#mts-aut-041), [MTS-AUT-042](13-authoring.md#mts-aut-042), [MTS-AUT-069](13-authoring.md#mts-aut-069) |
| `example.output_missing` | error | content | example | `path` | 1.0 | [MTS-AUT-040](13-authoring.md#mts-aut-040) |
| `example.output_unexpected` | error | content | example | `path` | 1.0 | [MTS-AUT-040](13-authoring.md#mts-aut-040) |
| `input.additional_item` | error | content | input | `index`, `prefix_count` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.additional_property` | error | content | input | `property`; optional: `suggestion` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.any_of` | error | content | input | `branch_count` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-FND-006](12-findings.md#mts-fnd-006) |
| `input.const` | error | content | input | `expected` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.constraints_failed` | error | content | invocation | `keyword` | 1.0 | [MTS-DAT-012](05-data-specification.md#mts-dat-012) |
| `input.constraints_not_evaluated` | info | content | invocation | — | 1.0 | [MTS-SUP-029](06-supplying-inputs.md#mts-sup-029) |
| `input.contains` | error | content | input | — | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.dependent_required` | error | content | input | `property`, `because` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.encoding_invalid` | error | content | input | `reason`, `offset` | 1.0 | [MTS-SUP-001](06-supplying-inputs.md#mts-sup-001) |
| `input.enum` | error | content | input | `allowed`, `allowed_count`; optional: `suggestion` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.exclusive_maximum` | error | content | input | `limit`; optional: `value` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.exclusive_minimum` | error | content | input | `limit`; optional: `value` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.extension_not_accepted` | error | content | input | `accepted` | 1.0 | [MTS-SUP-015](06-supplying-inputs.md#mts-sup-015), [MTS-SUP-018](06-supplying-inputs.md#mts-sup-018) |
| `input.file_duplicate` | error | content | input | — | 1.0 | [MTS-SUP-018](06-supplying-inputs.md#mts-sup-018) |
| `input.file_unknown` | error | content | input | — | 1.0 | [MTS-SUP-018](06-supplying-inputs.md#mts-sup-018) |
| `input.forbidden` | error | content | input | optional: `property` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.format` | error | content | input | `format` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-DAT-011](05-data-specification.md#mts-dat-011), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.json_arg_invalid` | error | content | invocation | `problem`; optional: `limit` | 1.0 | [MTS-SUP-020](06-supplying-inputs.md#mts-sup-020) |
| `input.json_arg_not_object` | error | content | invocation | `actual` | 1.0 | [MTS-SUP-020](06-supplying-inputs.md#mts-sup-020) |
| `input.json_arg_unknown` | error | content | input | — | 1.0 | [MTS-SUP-020](06-supplying-inputs.md#mts-sup-020) |
| `input.link` | error | content | input | — | 1.0 | [MTS-SEC-006](14-security.md#mts-sec-006) |
| `input.max_contains` | error | content | input | `limit`, `count` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.max_items` | error | content | input | `limit`, `count` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.max_length` | error | content | input | `limit`; optional: `length` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.max_properties` | error | content | input | `limit`, `count` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.maximum` | error | content | input | `limit`; optional: `value` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.min_contains` | error | content | input | `limit`, `count` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.min_items` | error | content | input | `limit`, `count` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.min_length` | error | content | input | `limit`; optional: `length` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.min_properties` | error | content | input | `limit`, `count` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.minimum` | error | content | input | `limit`; optional: `value` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.missing` | error | content | input | — | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-DAT-012](05-data-specification.md#mts-dat-012) |
| `input.multiple_of` | error | content | input | `divisor`; optional: `value` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-DAT-014](05-data-specification.md#mts-dat-014), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.noncharacter` | error | content | input | `codepoint` | 1.0 | [MTS-SUP-004](06-supplying-inputs.md#mts-sup-004) |
| `input.not` | error | content | input | — | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.not_readable` | error | content | input | `reason` | 1.0 | [MTS-SUP-030](06-supplying-inputs.md#mts-sup-030) |
| `input.not_regular_file` | error | content | input | `kind` | 1.0 | [MTS-SEC-006](14-security.md#mts-sec-006), [MTS-SUP-018](06-supplying-inputs.md#mts-sup-018) |
| `input.number_out_of_range` | error | content | input | `reason` | 1.0 | [MTS-SUP-005](06-supplying-inputs.md#mts-sup-005) |
| `input.one_of` | error | content | input | `branch_count` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-FND-006](12-findings.md#mts-fnd-006) |
| `input.one_of.ambiguous` | error | content | input | `branches` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.pattern` | error | content | input | `pattern` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.property_name` | error | content | input | `property` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.raw_unavailable` | error | content | input | `column`; optional: `index`, `output_path` | 1.0 | [MTS-TAG-004](08-filters-and-tags.md#mts-tag-004) |
| `input.required` | error | content | input | `property` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.set_into_unparsed` | error | content | input | — | 1.0 | [MTS-SUP-024](06-supplying-inputs.md#mts-sup-024) |
| `input.set_pointer_invalid` | error | content | invocation | `reason` | 1.0 | [MTS-SUP-021](06-supplying-inputs.md#mts-sup-021) |
| `input.set_traversal` | error | content | input | `reason` | 1.0 | [MTS-SUP-022](06-supplying-inputs.md#mts-sup-022) |
| `input.type` | error | content | input | `expected`, `actual` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.unevaluated_item` | error | content | input | `index` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.unevaluated_property` | error | content | input | `property`; optional: `suggestion` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.unique_items` | error | content | input | `index`, `duplicate_of` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-DAT-013](05-data-specification.md#mts-dat-013), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.unique_keys` | error | content | input | `index`, `duplicate_of` | 1.0 | [MTS-DAT-010](05-data-specification.md#mts-dat-010), [MTS-DAT-017](05-data-specification.md#mts-dat-017), [MTS-DAT-018](05-data-specification.md#mts-dat-018), [MTS-FND-004](12-findings.md#mts-fnd-004) |
| `input.validation.partial` | info | content | input | `unparsed`, `suppressed` | 1.0 | [MTS-SUP-029](06-supplying-inputs.md#mts-sup-029) |
| `internal.error` | error | internal | invocation | `tool_version`, `phase` | 1.0 | — |
| `internal.memory` | error | internal | invocation | `tool_version` | 1.0 | — |
| `json.duplicate_name` | error | content | input | `name` | 1.0 | [MTS-SUP-003](06-supplying-inputs.md#mts-sup-003) |
| `json.lone_surrogate` | error | content | input | — | 1.0 | [MTS-SUP-004](06-supplying-inputs.md#mts-sup-004) |
| `json.noncharacter` | error | content | input | — | 1.0 | [MTS-SUP-004](06-supplying-inputs.md#mts-sup-004) |
| `json.syntax` | error | content | input | `expected`, `found` | 1.0 | [MTS-SUP-003](06-supplying-inputs.md#mts-sup-003) |
| `json.trailing_content` | error | content | input | — | 1.0 | [MTS-SUP-003](06-supplying-inputs.md#mts-sup-003) |
| `limit.exceeded` | error | content | input | `limit`, `bound`, `lowered`; optional: `observed` | 1.0 | [MTS-SEC-005](14-security.md#mts-sec-005), [MTS-SEC-046](14-security.md#mts-sec-046) |
| `limit.exceeded.output` | error | content | output | `limit`, `bound`, `lowered`; optional: `observed` | 1.0 | [MTS-OUT-011](09-outputs.md#mts-out-011), [MTS-PTH-009](11-rendering.md#mts-pth-009), [MTS-SEC-010](14-security.md#mts-sec-010), [MTS-SEC-046](14-security.md#mts-sec-046) |
| `limit.exceeded.template` | error | content | template | `limit`, `bound`, `lowered`; optional: `output_path` | 1.0 | [MTS-OUT-010](09-outputs.md#mts-out-010), [MTS-SEC-008](14-security.md#mts-sec-008), [MTS-SEC-009](14-security.md#mts-sec-009), [MTS-TPL-046](07-template-environment.md#mts-tpl-046) |
| `limit.invalid` | error | usage | invocation | `reason`; optional: `name`, `default`, `unit`, `value` | 1.0 | — |
| `limit.package` | error | refused | package | `limit`, `bound`, `lowered`; optional: `observed` | 1.0 | [MTS-AUT-052](13-authoring.md#mts-aut-052), [MTS-PTH-007](11-rendering.md#mts-pth-007), [MTS-SEC-001](14-security.md#mts-sec-001), [MTS-SEC-011](14-security.md#mts-sec-011), [MTS-SEC-020](14-security.md#mts-sec-020), [MTS-SEC-021](14-security.md#mts-sec-021), [MTS-SEC-022](14-security.md#mts-sec-022), [MTS-SEC-042](14-security.md#mts-sec-042), [MTS-SEC-043](14-security.md#mts-sec-043), [MTS-SEC-044](14-security.md#mts-sec-044), [MTS-SEC-045](14-security.md#mts-sec-045) |
| `manifest.additional_property` | error | content | manifest | `property`; optional: `suggestion` | 1.0 | [MTS-MAN-011](03-manifest.md#mts-man-011), [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.const` | error | content | manifest | `expected` | 1.0 | [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.contains` | error | content | manifest | — | 1.0 | [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.dependent_required` | error | content | manifest | `property`, `because` | 1.0 | [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.enum` | error | content | manifest | `allowed`, `allowed_count`; optional: `suggestion` | 1.0 | [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.file.wrong_name` | error | content | manifest | — | 1.0 | [MTS-MAN-002](03-manifest.md#mts-man-002) |
| `manifest.forbidden` | error | content | manifest | optional: `property` | 1.0 | [MTS-MAN-019](03-manifest.md#mts-man-019) |
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
| `manifest.max_items` | error | content | manifest | `limit`, `count` | 1.0 | [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.max_length` | error | content | manifest | `limit`; optional: `length` | 1.0 | [MTS-MAN-012](03-manifest.md#mts-man-012), [MTS-MAN-014](03-manifest.md#mts-man-014), [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.min_items` | error | content | manifest | `limit`, `count` | 1.0 | [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.min_length` | error | content | manifest | `limit`; optional: `length` | 1.0 | [MTS-MAN-012](03-manifest.md#mts-man-012), [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.not` | error | content | manifest | — | 1.0 | [MTS-MAN-014](03-manifest.md#mts-man-014), [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.number_out_of_range` | error | content | manifest | `reason` | 1.0 | [MTS-MAN-010](03-manifest.md#mts-man-010) |
| `manifest.one_of` | error | content | manifest | `branch_count` | 1.0 | [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.output.duplicate_id` | error | content | manifest | `id` | 1.0 | [MTS-MAN-016](03-manifest.md#mts-man-016) |
| `manifest.output.nested_for_each` | error | content | manifest | — | 1.0 | [MTS-MAN-016](03-manifest.md#mts-man-016) |
| `manifest.path_not_found` | error | content | manifest | `directory` | 1.0 | [MTS-MAN-017](03-manifest.md#mts-man-017) |
| `manifest.pattern` | error | content | manifest | `pattern` | 1.0 | [MTS-MAN-012](03-manifest.md#mts-man-012), [MTS-MAN-014](03-manifest.md#mts-man-014), [MTS-MAN-019](03-manifest.md#mts-man-019), [MTS-VER-008](04-versions.md#mts-ver-008) |
| `manifest.required` | error | content | manifest | `property` | 1.0 | [MTS-MAN-011](03-manifest.md#mts-man-011), [MTS-MAN-019](03-manifest.md#mts-man-019) |
| `manifest.type` | error | content | manifest | `expected`, `actual` | 1.0 | [MTS-MAN-011](03-manifest.md#mts-man-011), [MTS-MAN-019](03-manifest.md#mts-man-019), [MTS-VER-008](04-versions.md#mts-ver-008) |
| `manifest.unique_items` | error | content | manifest | `index`, `duplicate_of` | 1.0 | [MTS-MAN-019](03-manifest.md#mts-man-019) |
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
| `markdown.frontmatter_body_field` | error | content | input | — | 1.0 | [MTS-SUP-012](06-supplying-inputs.md#mts-sup-012) |
| `markdown.frontmatter_not_mapping` | error | content | input | `actual` | 1.0 | [MTS-SUP-012](06-supplying-inputs.md#mts-sup-012) |
| `markdown.frontmatter_unclosed` | error | content | input | — | 1.0 | [MTS-SUP-012](06-supplying-inputs.md#mts-sup-012) |
| `output.additional_item` | error | content | output | `index`, `prefix_count` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.additional_property` | error | content | output | `property`; optional: `suggestion` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.any_of` | error | content | output | `branch_count` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-FND-006](12-findings.md#mts-fnd-006), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.const` | error | content | output | `expected` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.container.missing_entry` | error | content | output | `profile`, `entry` | 1.0 | [MTS-TRN-009](10-transforms.md#mts-trn-009), [MTS-TRN-010](10-transforms.md#mts-trn-010), [MTS-TRN-011](10-transforms.md#mts-trn-011), [MTS-AUT-031](13-authoring.md#mts-aut-031) |
| `output.container.non_ascii` | error | content | output | `profile`, `path` | 1.0 | [MTS-TRN-009](10-transforms.md#mts-trn-009), [MTS-AUT-031](13-authoring.md#mts-aut-031) |
| `output.container.reserved_entry` | error | content | output | `profile`, `path` | 1.0 | [MTS-TRN-010](10-transforms.md#mts-trn-010), [MTS-TRN-011](10-transforms.md#mts-trn-011), [MTS-AUT-031](13-authoring.md#mts-aut-031) |
| `output.container.space` | warning | content | output | `profile`, `path` | 1.0 | [MTS-TRN-011](10-transforms.md#mts-trn-011), [MTS-AUT-031](13-authoring.md#mts-aut-031) |
| `output.container.too_many_entries` | error | content | output | `count` | 1.0 | [MTS-TRN-007](10-transforms.md#mts-trn-007) |
| `output.contains` | error | content | output | — | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.declaration.empty_mirror` | warning | content | output | `directory` | 1.0 | [MTS-OUT-019](09-outputs.md#mts-out-019) |
| `output.declaration.empty_name` | error | content | output | `template` | 1.0 | [MTS-OUT-018](09-outputs.md#mts-out-018) |
| `output.declaration.validate_kind` | error | content | output | `kind` | 1.0 | [MTS-AUT-023](13-authoring.md#mts-aut-023) |
| `output.dependent_required` | error | content | output | `property`, `because` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.destination.case_variant` | error | destination | invocation | `path` | 1.0 | [MTS-RND-017](11-rendering.md#mts-rnd-017) |
| `output.destination.commit_failed` | error | destination | invocation | `reason`, `committed` | 1.0 | [MTS-RND-022](11-rendering.md#mts-rnd-022) |
| `output.destination.in_package` | error | destination | invocation | `entry` | 1.0 | [MTS-AUT-051](13-authoring.md#mts-aut-051) |
| `output.destination.link` | error | destination | invocation | `path` | 1.0 | [MTS-RND-015](11-rendering.md#mts-rnd-015) |
| `output.destination.not_directory` | error | destination | invocation | optional: `path` | 1.0 | [MTS-AUT-028](13-authoring.md#mts-aut-028), [MTS-AUT-050](13-authoring.md#mts-aut-050), [MTS-RND-003](11-rendering.md#mts-rnd-003), [MTS-RND-016](11-rendering.md#mts-rnd-016), [MTS-SMP-006](13-authoring.md#mts-smp-006) |
| `output.destination.not_empty` | error | destination | invocation | `entries`, `more` | 1.0 | [MTS-AUT-028](13-authoring.md#mts-aut-028), [MTS-RND-004](11-rendering.md#mts-rnd-004), [MTS-SMP-006](13-authoring.md#mts-smp-006) |
| `output.destination.not_regular_file` | error | destination | invocation | `path`, `reason` | 1.0 | [MTS-RND-016](11-rendering.md#mts-rnd-016) |
| `output.destination.nothing_written` | error | destination | invocation | `reason` | 1.0 | [MTS-AUT-026](13-authoring.md#mts-aut-026), [MTS-AUT-027](13-authoring.md#mts-aut-027), [MTS-AUT-050](13-authoring.md#mts-aut-050), [MTS-RND-006](11-rendering.md#mts-rnd-006), [MTS-RND-018](11-rendering.md#mts-rnd-018), [MTS-RND-019](11-rendering.md#mts-rnd-019), [MTS-RND-022](11-rendering.md#mts-rnd-022), [MTS-SMP-008](13-authoring.md#mts-smp-008) |
| `output.destination.staging_exists` | error | destination | invocation | — | 1.0 | [MTS-RND-014](11-rendering.md#mts-rnd-014) |
| `output.destination.write_failed` | error | destination | invocation | `reason`, `written` | 1.0 | [MTS-AUT-027](13-authoring.md#mts-aut-027) |
| `output.enum` | error | content | output | `allowed`, `allowed_count`; optional: `suggestion` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.exclusive_maximum` | error | content | output | `limit`; optional: `value` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.exclusive_minimum` | error | content | output | `limit`; optional: `value` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.expression.for_each_type` | error | content | output | `type` | 1.0 | [MTS-OUT-007](09-outputs.md#mts-out-007) |
| `output.forbidden` | error | content | output | optional: `property` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.format` | error | content | output | `format` | 1.0 | [MTS-DAT-011](05-data-specification.md#mts-dat-011), [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.max_contains` | error | content | output | `limit`, `count` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.max_items` | error | content | output | `limit`, `count` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.max_length` | error | content | output | `limit`; optional: `length` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.max_properties` | error | content | output | `limit`, `count` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.maximum` | error | content | output | `limit`; optional: `value` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.min_contains` | error | content | output | `limit`, `count` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.min_items` | error | content | output | `limit`, `count` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.min_length` | error | content | output | `limit`; optional: `length` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.min_properties` | error | content | output | `limit`, `count` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.minimum` | error | content | output | `limit`; optional: `value` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.multiple_of` | error | content | output | `divisor`; optional: `value` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.not` | error | content | output | — | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.one_of` | error | content | output | `branch_count` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-FND-006](12-findings.md#mts-fnd-006), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.one_of.ambiguous` | error | content | output | `branches` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.parse` | error | content | output | `form`, `rule` | 1.0 | [MTS-TRN-014](10-transforms.md#mts-trn-014), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.path.case_collision` | error | content | output | `path`, `member`, `other`; optional: `item` | 1.0 | [MTS-OUT-009](09-outputs.md#mts-out-009), [MTS-PTH-006](11-rendering.md#mts-pth-006), [MTS-PTH-008](11-rendering.md#mts-pth-008) |
| `output.path.character` | error | content | output | `path`, `code_point`; optional: `item` | 1.0 | [MTS-OUT-009](09-outputs.md#mts-out-009), [MTS-PTH-004](11-rendering.md#mts-pth-004), [MTS-PTH-008](11-rendering.md#mts-pth-008) |
| `output.path.duplicate` | error | content | output | `path`; optional: `item` | 1.0 | [MTS-OUT-009](09-outputs.md#mts-out-009), [MTS-PTH-006](11-rendering.md#mts-pth-006), [MTS-PTH-008](11-rendering.md#mts-pth-008) |
| `output.path.encoding` | error | content | output | `path`, `reason`; optional: `item` | 1.0 | [MTS-OUT-009](09-outputs.md#mts-out-009), [MTS-PTH-001](11-rendering.md#mts-pth-001), [MTS-PTH-008](11-rendering.md#mts-pth-008) |
| `output.path.file_directory_conflict` | error | content | output | `path`, `member`, `other`; optional: `item` | 1.0 | [MTS-OUT-009](09-outputs.md#mts-out-009), [MTS-PTH-006](11-rendering.md#mts-pth-006), [MTS-PTH-008](11-rendering.md#mts-pth-008) |
| `output.path.not_nfc` | error | content | output | `path`; optional: `item` | 1.0 | [MTS-OUT-009](09-outputs.md#mts-out-009), [MTS-PTH-002](11-rendering.md#mts-pth-002), [MTS-PTH-008](11-rendering.md#mts-pth-008) |
| `output.path.not_portable` | error | content | output | `path`, `reason`; optional: `item` | 1.0 | [MTS-OUT-009](09-outputs.md#mts-out-009), [MTS-PTH-005](11-rendering.md#mts-pth-005), [MTS-PTH-008](11-rendering.md#mts-pth-008) |
| `output.path.shape` | error | content | output | `path`, `reason`; optional: `item` | 1.0 | [MTS-OUT-009](09-outputs.md#mts-out-009), [MTS-PTH-003](11-rendering.md#mts-pth-003), [MTS-PTH-008](11-rendering.md#mts-pth-008) |
| `output.pattern` | error | content | output | `pattern` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.property_name` | error | content | output | `property` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.required` | error | content | output | `property` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.type` | error | content | output | `expected`, `actual` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.unevaluated_item` | error | content | output | `index` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.unevaluated_property` | error | content | output | `property`; optional: `suggestion` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.unique_items` | error | content | output | `index`, `duplicate_of` | 1.0 | [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
| `output.unique_keys` | error | content | output | `index`, `duplicate_of` | 1.0 | [MTS-DAT-018](05-data-specification.md#mts-dat-018), [MTS-FND-004](12-findings.md#mts-fnd-004), [MTS-TRN-015](10-transforms.md#mts-trn-015) |
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
| `package.container.field_value` | error | refused | package | `offset`, `record`, `header_field`, `value`, `expected` | 1.0 | [MTS-PKG-006](02-container.md#mts-pkg-006), [MTS-PKG-045](02-container.md#mts-pkg-045) |
| `package.container.flags` | error | refused | package | `offset`, `value` | 1.0 | [MTS-PKG-007](02-container.md#mts-pkg-007) |
| `package.container.header_mismatch` | error | refused | package | `offset`, `header_field` | 1.0 | [MTS-PKG-009](02-container.md#mts-pkg-009) |
| `package.container.method` | error | refused | package | `offset`, `method` | 1.0 | [MTS-PKG-007](02-container.md#mts-pkg-007) |
| `package.container.not_zip` | error | refused | package | — | 1.0 | [MTS-PKG-003](02-container.md#mts-pkg-003) |
| `package.container.overlap_or_gap` | error | refused | package | `offset`; optional: `declared` | 1.0 | [MTS-PKG-003](02-container.md#mts-pkg-003), [MTS-PKG-045](02-container.md#mts-pkg-045) |
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
| `package.layout.readme` | error | refused | package | `reason` | 1.0 | [MTS-PKG-071](02-container.md#mts-pkg-071), [MTS-PKG-072](02-container.md#mts-pkg-072) |
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
| `package.not_found` | error | refused | package | — | 1.0 | [MTS-PKG-001](02-container.md#mts-pkg-001) |
| `package.not_readable` | error | refused | package | `reason` | 1.0 | [MTS-PKG-046](02-container.md#mts-pkg-046) |
| `package.version.malformed` | error | refused | package | `implemented` | 1.0 | [MTS-VER-003](04-versions.md#mts-ver-003) |
| `package.version.missing` | error | refused | package | `implemented` | 1.0 | [MTS-VER-001](04-versions.md#mts-ver-001) |
| `package.version.not_string` | error | refused | package | `actual`, `implemented`; optional: `written` | 1.0 | [MTS-VER-002](04-versions.md#mts-ver-002) |
| `package.version.unsupported_major` | error | refused | package | `declared`, `implemented` | 1.0 | [MTS-VER-004](04-versions.md#mts-ver-004) |
| `package.version.unsupported_minor` | error | refused | package | `declared`, `implemented` | 1.0 | [MTS-VER-005](04-versions.md#mts-ver-005) |
| `report.step_skipped` | info | content | report | `steps` | 1.0 | [MTS-AUT-002](13-authoring.md#mts-aut-002) |
| `report.truncated` | info | content | report | `listed`, `omitted` | 1.0 | [MTS-FND-010](12-findings.md#mts-fnd-010) |
| `schema.dialect` | error | content | package | `reason` | 1.0 | [MTS-DAT-003](05-data-specification.md#mts-dat-003) |
| `schema.format_unsupported` | error | content | package | `format` | 1.0 | [MTS-DAT-005](05-data-specification.md#mts-dat-005) |
| `schema.id_duplicate` | error | content | package | `id`; optional: `other` | 1.0 | [MTS-DAT-008](05-data-specification.md#mts-dat-008) |
| `schema.invalid` | error | content | package | `keyword` | 1.0 | [MTS-DAT-003](05-data-specification.md#mts-dat-003), [MTS-DAT-015](05-data-specification.md#mts-dat-015) |
| `schema.invalid_default` | error | content | package | — | 1.0 | [MTS-AUT-007](13-authoring.md#mts-aut-007) |
| `schema.invalid_example` | error | content | package | — | 1.0 | [MTS-AUT-007](13-authoring.md#mts-aut-007) |
| `schema.keyword_unknown` | error | content | package | `keyword` | 1.0 | [MTS-DAT-004](05-data-specification.md#mts-dat-004) |
| `schema.pattern_unsupported` | error | content | package | `construct` | 1.0 | [MTS-DAT-006](05-data-specification.md#mts-dat-006), [MTS-SEC-041](14-security.md#mts-sec-041) |
| `schema.ref_by_path` | error | content | package | `reference`, `id` | 1.0 | [MTS-DAT-008](05-data-specification.md#mts-dat-008) |
| `schema.ref_escapes` | error | content | package | `reference` | 1.0 | [MTS-DAT-007](05-data-specification.md#mts-dat-007) |
| `schema.ref_unresolved` | error | content | package | `reference` | 1.0 | [MTS-DAT-007](05-data-specification.md#mts-dat-007) |
| `schema.required_not_declared` | warning | content | package | `name` | 1.0 | [MTS-AUT-009](13-authoring.md#mts-aut-009) |
| `schema.sample_invalid` | error | content | input | `code` | 1.0 | [MTS-SMP-002](13-authoring.md#mts-smp-002), [MTS-SMP-009](13-authoring.md#mts-smp-009) |
| `schema.sample_unsatisfiable` | error | content | input | `instance`, `reason` | 1.0 | [MTS-SMP-004](13-authoring.md#mts-smp-004), [MTS-SMP-009](13-authoring.md#mts-smp-009) |
| `schema.syntax` | error | content | package | `form`, `rule` | 1.0 | [MTS-DAT-002](05-data-specification.md#mts-dat-002) |
| `schema.unsatisfiable` | error | content | package | — | 1.0 | [MTS-AUT-008](13-authoring.md#mts-aut-008) |
| `schema.unusable_enum_member` | warning | content | package | — | 1.0 | [MTS-AUT-008](13-authoring.md#mts-aut-008) |
| `template.asset_ambiguous` | error | content | template | `asset`, `column`, `output_path` | 1.0 | [MTS-TAG-015](08-filters-and-tags.md#mts-tag-015) |
| `template.asset_not_found` | error | content | template | `asset`, `column` | 1.0 | [MTS-TAG-013](08-filters-and-tags.md#mts-tag-013) |
| `template.asset_not_text` | error | content | template | `asset`, `column`, `output_path` | 1.0 | [MTS-TAG-016](08-filters-and-tags.md#mts-tag-016) |
| `template.asset_not_written` | error | content | template | `asset`, `column`, `output_path` | 1.0 | [MTS-TAG-015](08-filters-and-tags.md#mts-tag-015) |
| `template.asset_tree` | error | content | template | `asset`, `column`, `output_path` | 1.0 | [MTS-TAG-015](08-filters-and-tags.md#mts-tag-015) |
| `template.comparison` | error | content | template | `operator`, `left`, `right`, `column`; optional: `output_path` | 1.0 | [MTS-TPL-023](07-template-environment.md#mts-tpl-023) |
| `template.crlf` | warning | content | template | — | 1.0 | [MTS-AUT-010](13-authoring.md#mts-aut-010) |
| `template.date_format` | error | content | template | `column`; optional: `directive`, `output_path` | 1.0 | [MTS-FLT-013](08-filters-and-tags.md#mts-flt-013), [MTS-MAN-018](03-manifest.md#mts-man-018) |
| `template.date_input` | error | content | template | `type`, `column`; optional: `global`, `output_path` | 1.0 | [MTS-FLT-011](08-filters-and-tags.md#mts-flt-011) |
| `template.division_by_zero` | error | content | template | `filter`, `column`; optional: `output_path` | 1.0 | [MTS-MAN-018](03-manifest.md#mts-man-018), [MTS-TPL-019](07-template-environment.md#mts-tpl-019) |
| `template.duration_format` | error | content | template | `column`; optional: `directive`, `output_path` | 1.0 | [MTS-FLT-022](08-filters-and-tags.md#mts-flt-022), [MTS-MAN-018](03-manifest.md#mts-man-018) |
| `template.duration_input` | error | content | template | `type`, `reason`, `column`; optional: `output_path` | 1.0 | [MTS-FLT-021](08-filters-and-tags.md#mts-flt-021) |
| `template.embed_index` | error | content | template | `input`, `reason`, `column`; optional: `index`, `items`, `type`, `output_path` | 1.0 | [MTS-TAG-002](08-filters-and-tags.md#mts-tag-002) |
| `template.embed_unknown_input` | error | content | template | `column`; optional: `name` | 1.0 | [MTS-TAG-001](08-filters-and-tags.md#mts-tag-001) |
| `template.filter_argument` | error | content | template | `filter`, `argument`, `column`; optional: `output_path` | 1.0 | [MTS-FLT-004](08-filters-and-tags.md#mts-flt-004), [MTS-MAN-018](03-manifest.md#mts-man-018), [MTS-TPL-052](07-template-environment.md#mts-tpl-052) |
| `template.filter_input` | error | content | template | `filter`, `type`, `column`; optional: `output_path` | 1.0 | [MTS-FLT-004](08-filters-and-tags.md#mts-flt-004), [MTS-FLT-009](08-filters-and-tags.md#mts-flt-009), [MTS-TPL-052](07-template-environment.md#mts-tpl-052) |
| `template.include` | error | content | template | `column` | 1.0 | [MTS-MAN-018](03-manifest.md#mts-man-018), [MTS-TPL-002](07-template-environment.md#mts-tpl-002) |
| `template.invalid_utf8` | error | content | template | `column`; optional: `filter`, `tag`, `output_path` | 1.0 | [MTS-TAG-003](08-filters-and-tags.md#mts-tag-003), [MTS-TPL-052](07-template-environment.md#mts-tpl-052) |
| `template.nil_property` | error | content | template | `path`, `column`; optional: `output_path` | 1.0 | [MTS-TPL-030](07-template-environment.md#mts-tpl-030) |
| `template.not_utf8` | error | content | template | — | 1.0 | [MTS-AUT-010](13-authoring.md#mts-aut-010), [MTS-TPL-006](07-template-environment.md#mts-tpl-006) |
| `template.number_range` | error | content | template | `column`; optional: `filter`, `output_path` | 1.0 | [MTS-MAN-018](03-manifest.md#mts-man-018), [MTS-TPL-019](07-template-environment.md#mts-tpl-019) |
| `template.object_output` | error | content | template | `column`; optional: `output_path` | 1.0 | [MTS-TPL-021](07-template-environment.md#mts-tpl-021) |
| `template.ooxml.split_construct` | error | content | template | `cause`, `text`, `text_elements` | 1.0 | [MTS-AUT-019](13-authoring.md#mts-aut-019) |
| `template.ooxml.split_delimiter` | error | content | template | `cause`, `delimiter`, `text`, `text_elements` | 1.0 | [MTS-AUT-019](13-authoring.md#mts-aut-019) |
| `template.partial_name` | error | content | template | `column` | 1.0 | [MTS-TPL-041](07-template-environment.md#mts-tpl-041) |
| `template.partial_not_found` | error | content | template | `partial`, `column` | 1.0 | [MTS-TPL-042](07-template-environment.md#mts-tpl-042), [MTS-TPL-043](07-template-environment.md#mts-tpl-043) |
| `template.provenance_argument` | error | content | template | `reason`, `column`; optional: `argument` | 1.0 | [MTS-TAG-005](08-filters-and-tags.md#mts-tag-005) |
| `template.provenance_style` | error | content | template | `reason`, `column`; optional: `kind`, `style`, `type`, `output_path` | 1.0 | [MTS-TAG-010](08-filters-and-tags.md#mts-tag-010) |
| `template.reserved_name` | error | content | template | `name`, `column` | 1.0 | [MTS-TPL-040](07-template-environment.md#mts-tpl-040) |
| `template.safe_position` | error | content | template | `column` | 1.0 | [MTS-FLT-002](08-filters-and-tags.md#mts-flt-002) |
| `template.slug_empty` | error | content | template | `column`; optional: `output_path` | 1.0 | [MTS-FLT-010](08-filters-and-tags.md#mts-flt-010) |
| `template.syntax` | error | content | template | `column`, `cause` | 1.0 | [MTS-AUT-010](13-authoring.md#mts-aut-010), [MTS-AUT-011](13-authoring.md#mts-aut-011), [MTS-MAN-018](03-manifest.md#mts-man-018), [MTS-TPL-005](07-template-environment.md#mts-tpl-005), [MTS-TPL-006](07-template-environment.md#mts-tpl-006) |
| `template.tag_argument` | error | content | template | `argument`, `type`, `column`; optional: `tag`, `output_path` | 1.0 | [MTS-TPL-013](07-template-environment.md#mts-tpl-013) |
| `template.tag_context` | error | content | template | `tag`, `column` | 1.0 | [MTS-MAN-018](03-manifest.md#mts-man-018), [MTS-TAG-011](08-filters-and-tags.md#mts-tag-011) |
| `template.undefined_property` | error | content | template | `path`, `column`; optional: `output_path` | 1.0 | [MTS-FLT-009](08-filters-and-tags.md#mts-flt-009), [MTS-TPL-029](07-template-environment.md#mts-tpl-029), [MTS-TPL-031](07-template-environment.md#mts-tpl-031), [MTS-TPL-032](07-template-environment.md#mts-tpl-032) |
| `template.undefined_variable` | error | content | template | `name`, `column`; optional: `output_path` | 1.0 | [MTS-OUT-004](09-outputs.md#mts-out-004), [MTS-TPL-003](07-template-environment.md#mts-tpl-003), [MTS-TPL-027](07-template-environment.md#mts-tpl-027), [MTS-TPL-033](07-template-environment.md#mts-tpl-033), [MTS-TPL-034](07-template-environment.md#mts-tpl-034), [MTS-TPL-058](07-template-environment.md#mts-tpl-058) |
| `template.unknown_filter` | error | content | template | `filter`, `column` | 1.0 | [MTS-MAN-018](03-manifest.md#mts-man-018), [MTS-TPL-004](07-template-environment.md#mts-tpl-004) |
| `template.xref.budget_exhausted` | warning | content | package | `bound`, `limit` | 1.0 | [MTS-AUT-018](13-authoring.md#mts-aut-018) |
| `template.xref.dynamic_reference` | warning | content | template | optional: `output` | 1.0 | [MTS-AUT-014](13-authoring.md#mts-aut-014) |
| `template.xref.for_each_not_array` | error | content | output | `types` | 1.0 | [MTS-AUT-016](13-authoring.md#mts-aut-016) |
| `template.xref.index_out_of_range` | error | content | template | `index`; optional: `input`, `output` | 1.0 | [MTS-AUT-014](13-authoring.md#mts-aut-014) |
| `template.xref.missing_asset` | error | content | template | `asset` | 1.0 | [MTS-AUT-032](13-authoring.md#mts-aut-032) |
| `template.xref.missing_partial` | error | content | template | `partial`; optional: `output` | 1.0 | [MTS-AUT-015](13-authoring.md#mts-aut-015) |
| `template.xref.not_an_array` | error | content | template | `types`; optional: `input`, `output` | 1.0 | [MTS-AUT-014](13-authoring.md#mts-aut-014) |
| `template.xref.not_an_object` | error | content | template | `property`, `types`; optional: `input`, `output` | 1.0 | [MTS-AUT-013](13-authoring.md#mts-aut-013) |
| `template.xref.partial_parameter_not_passed` | warning | content | template | `name`; optional: `output` | 1.0 | [MTS-AUT-015](13-authoring.md#mts-aut-015) |
| `template.xref.unknown_field` | error | content | template | `property`, `declared`, `declared_count`; optional: `suggestion`, `input`, `output` | 1.0 | [MTS-AUT-013](13-authoring.md#mts-aut-013) |
| `template.xref.unknown_input` | error | content | template | `name`; optional: `suggestion` | 1.0 | [MTS-AUT-030](13-authoring.md#mts-aut-030) |
| `template.xref.unknown_variable` | error | content | template | `name`; optional: `suggestion`, `output` | 1.0 | [MTS-AUT-012](13-authoring.md#mts-aut-012) |
| `template.xref.unused_input` | warning | content | input | — | 1.0 | [MTS-AUT-017](13-authoring.md#mts-aut-017) |
| `template.xref.unused_template` | warning | content | template | — | 1.0 | [MTS-AUT-017](13-authoring.md#mts-aut-017) |
| `usage.archive_to_terminal` | error | usage | invocation | `argument` | 1.0 | — |
| `usage.bad_glob` | error | usage | invocation | `argument`, `reason` | 1.0 | — |
| `usage.bad_pattern` | error | usage | invocation | `argument`, `reason`; optional: `limit`, `bound` | 1.0 | — |
| `usage.bad_value` | error | usage | invocation | `argument`, `flag`, `reason`; optional: `allowed` | 1.0 | — |
| `usage.binary_to_terminal` | error | usage | invocation | `argument` | 1.0 | — |
| `usage.conflicting_flags` | error | usage | invocation | `argument`, `flag` | 1.0 | — |
| `usage.data_dir_unreadable` | error | usage | invocation | `argument`, `reason` | 1.0 | — |
| `usage.entry_not_found` | error | usage | invocation | `argument` | 1.0 | — |
| `usage.flag_excluded` | error | usage | invocation | `argument`, `command`, `flag`, `other` | 1.0 | — |
| `usage.flag_requires` | error | usage | invocation | `argument`, `command`, `flag`, `requires` | 1.0 | — |
| `usage.harness_required` | error | usage | invocation | `found` | 1.0 | — |
| `usage.is_a_directory` | error | usage | invocation | `argument`, `command` | 1.0 | — |
| `usage.line_out_of_range` | error | usage | invocation | `argument`, `total` | 1.0 | — |
| `usage.missing_command` | error | usage | invocation | — | 1.0 | — |
| `usage.missing_flag` | error | usage | invocation | `command`, `flag` | 1.0 | — |
| `usage.missing_operand` | error | usage | invocation | `command`, `operand` | 1.0 | — |
| `usage.name_invalid` | error | usage | invocation | `argument`, `source`; optional: `suggestion` | 1.0 | — |
| `usage.name_required` | error | usage | invocation | — | 1.0 | — |
| `usage.no_match` | error | usage | invocation | `argument` | 1.0 | — |
| `usage.not_a_packed_package` | error | usage | invocation | `argument` | 1.0 | — |
| `usage.not_a_source_directory` | error | usage | invocation | `argument` | 1.0 | — |
| `usage.not_repository_root` | warning | usage | invocation | — | 1.0 | — |
| `usage.not_text` | error | usage | invocation | `argument`, `command` | 1.0 | — |
| `usage.package_from_stdin` | error | usage | invocation | `argument` | 1.0 | — |
| `usage.repeated_flag` | error | usage | invocation | `argument`, `flag` | 1.0 | — |
| `usage.skill_edited` | error | usage | invocation | `path` | 1.0 | — |
| `usage.unexpanded_tilde` | error | usage | invocation | `argument` | 1.0 | — |
| `usage.unexpected_operand` | error | usage | invocation | `argument`, `command` | 1.0 | — |
| `usage.unknown_command` | error | usage | invocation | `argument` | 1.0 | — |
| `usage.unknown_example` | error | usage | invocation | `argument`, `examples` | 1.0 | — |
| `usage.unknown_flag` | error | usage | invocation | `argument`, `reason`; optional: `flag`, `command` | 1.0 | — |
| `usage.unknown_input` | error | usage | invocation | `argument`, `inputs` | 1.0 | — |
| `usage.unknown_output` | error | usage | invocation | `argument`, `paths`, `path_count` | 1.0 | — |
| `usage.unknown_topic` | error | usage | invocation | `argument` | 1.0 | — |
| `yaml.alias` | error | content | input | — | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
| `yaml.anchor` | error | content | input | — | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
| `yaml.bom` | error | content | input | — | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
| `yaml.character` | error | content | input | `codepoint`, `reason` | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
| `yaml.directive` | error | content | input | — | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
| `yaml.duplicate_key` | error | content | input | — | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
| `yaml.key_empty` | error | content | input | — | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
| `yaml.key_not_scalar` | error | content | input | — | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
| `yaml.merge_key` | error | content | input | — | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
| `yaml.multi_document` | error | content | input | — | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
| `yaml.scalar_refused` | error | content | input | `form` | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
| `yaml.syntax` | error | content | input | — | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
| `yaml.tag` | error | content | input | — | 1.0 | [MTS-SUP-006](06-supplying-inputs.md#mts-sup-006) |
<!-- The end of the generated code catalog. -->

## 12.2 The finding

A finding is one problem an operation reports. Its members are its `code`,
registered in §12.1; its `severity`, `error`, `warning` or `info`, which the
registry gives the code; its `subject`, the kind of thing it is about with
that thing's name where the kind has one; its `location` where the code's
registry entry gives it a span: the origin of the value or file, the file
relative to that origin, the 1-based line and column, in Unicode code points,
where the value starts, and the JSON Pointer of the value; and its `params`,
which the registry declares by name and type. A failure against a schema
also carries `schema`, `field`, `via` and `alternatives` (§12.3), and a
finding that names a second place, such as the earlier of two colliding
declarations, carries it in `related`, each a location with a note. A
message and a hint come with every finding, and a schema failure also
gives the keyword's location in the evaluation; their wording and form are
the implementation's own, and only the other members are compared.

Each code also has a class, which the registry gives it and which says
what kind of failure an error of that code is: `usage`, the operation was
asked for wrongly; `refused`, the package could not be read as a package
at all; `content`, the package, the inputs or the outputs break a rule;
`destination`, where the operation writes refused the write; and
`internal`, a defect in the implementation itself. A class decides how a
tool reports the failure, such as its exit status, and changes nothing a
finding says.

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
argument, pointer, schema entry, schema pointer and parameters, it MUST report
one.

## 12.3 A schema failure

A failure of a value against its schema is reported by the code of the
keyword that failed, in the family of what the schema validates: `input.` for a
supplied input, `manifest.` for the manifest, `output.` for an output. The
`input.` and `output.` families hold every code of the table below. The
`manifest.` family holds only the codes a manifest can be given by the
manifest schema of its revision ([chapter 3](03-manifest.md)): the schema of
1.0 uses none of `multipleOf`, `maximum`, `exclusiveMaximum`, `minimum`,
`exclusiveMinimum`, `format`, `minContains`, `maxContains`, `maxProperties`,
`minProperties`, `items` after `prefixItems`, `unevaluatedProperties`,
`unevaluatedItems` and `propertyNames`; its one `anyOf` stands beneath a
`not`, whose subschema reports nothing; and no value matches two branches of
any of its `oneOf`s. So `manifest.any_of`, `manifest.one_of.ambiguous` and
the codes of those keywords are not registered. The codes of the `input.`
family:

| Keyword | Code |
|---|---|
| `type` | `input.type` |
| `enum` | `input.enum` |
| `const` | `input.const` |
| `multipleOf` | `input.multiple_of` |
| `maximum`, `exclusiveMaximum` | `input.maximum`, `input.exclusive_maximum` |
| `minimum`, `exclusiveMinimum` | `input.minimum`, `input.exclusive_minimum` |
| `maxLength`, `minLength` | `input.max_length`, `input.min_length` |
| `pattern` | `input.pattern` |
| `format` | `input.format` |
| `maxItems`, `minItems` | `input.max_items`, `input.min_items` |
| `uniqueItems` | `input.unique_items` |
| `uniqueKeys` | `input.unique_keys` |
| `contains`, `minContains`, `maxContains` | `input.contains`, `input.min_contains`, `input.max_contains` |
| `maxProperties`, `minProperties` | `input.max_properties`, `input.min_properties` |
| `required` | `input.required` |
| `dependentRequired` | `input.dependent_required` |
| `additionalProperties` | `input.additional_property` |
| `items` after `prefixItems` | `input.additional_item` |
| `unevaluatedProperties` | `input.unevaluated_property` |
| `unevaluatedItems` | `input.unevaluated_item` |
| `propertyNames` | `input.property_name` |
| a `false` schema | `input.forbidden` |
| `not` | `input.not` |
| `anyOf` | `input.any_of` |
| `oneOf`, no branch accepting | `input.one_of` |
| `oneOf`, several branches accepting | `input.one_of.ambiguous` |

<a id="mts-fnd-004"></a>**MTS-FND-004** A processor MUST report each failing
keyword by its code in the table above: `required`, `dependentRequired` and
`additionalProperties` once per member they concern, `uniqueKeys` once per
item whose key repeats and `coverage` once per value it misses, a missing
member at the pointer of the absent member with the line and column of its
parent, an extra or misnamed member at its name, an item at its index, a
repeated key where [MTS-DAT-018](05-data-specification.md#mts-dat-018)
places it, and each other keyword at the value it fails.

<a id="mts-fnd-005"></a>**MTS-FND-005** A processor MUST report a schema
failure with its `schema` member: the package entry of the schema file the
failing keyword is written in, the JSON Pointer of the keyword within that
file, and the keyword, followed through every `$ref` to where the keyword is
authored.

<a id="mts-fnd-006"></a>**MTS-FND-006** Of an `anyOf` or `oneOf` no branch of
which accepts the value, a processor MUST report the failures of the one branch
that its value's type and its `const` and `enum` members do not reject, with a
`via` entry naming the keyword and the branch, and when every branch is so
rejected MUST report one `input.any_of` or `input.one_of` whose `alternatives`
list each branch's failures; and a failure under a `then` or `else` MUST carry a
`via` entry naming the `if` that selected it, with that schema's title.

<a id="mts-fnd-007"></a>**MTS-FND-007** A processor MUST give a finding's
`field` the `title` and, independently, the `description` of the first schema,
walking the evaluation path from the failed keyword back toward the root, whose
instance location is the finding's; for an absent member, of the `properties`
subschema naming it, following its `$ref` chain when it has neither. A schema
at another instance location never supplies a field, so that a parent's title
is never a child's.

## 12.4 Where a finding is

<a id="mts-fnd-008"></a>**MTS-FND-008** A processor MUST locate a finding about
a value at the line and column where the value starts, columns counted in
Unicode code points from 1, a tab one column, a line ended by LF, CRLF or a
lone CR; in a CSV file, the line where the record or the quoted field starts;
on the command line, the line and column within an `--inputs-json` argument
and no line for a path set; and at the value's JSON Pointer within its input,
each `~` and `/` of a member name written `~0` and `~1`.

## 12.5 The report

<a id="mts-fnd-009"></a>**MTS-FND-009** A processor MUST count every finding
of an operation, after removing duplicates, whether it lists it or not.

<a id="mts-fnd-010"></a>**MTS-FND-010** A processor MUST list at most the
`findings` limit of [chapter 14](14-security.md) of an operation's findings,
the first in the order of §12.2, and when it omits any MUST list after them
one `report.truncated`, about the report, giving how many it listed and how
many it omitted.

<a id="mts-fnd-020"></a>**MTS-FND-020** A processor MUST count toward the
`findings` limit every finding it lists, a finding listed within another's
`alternatives` included, taking them in the order it lists them: the
report's findings in the order of §12.2, each followed by the findings of its
`alternatives`, branch by branch in index order, and each of those followed
in turn by the findings of its own.

<a id="mts-fnd-021"></a>**MTS-FND-021** When the `findings` limit is reached
within a finding's `alternatives`, a processor MUST list that finding with
only the findings of its alternatives that come before the limit, MUST count
each finding it leaves out of a branch in that branch's `omitted`, and MUST
list no later finding of the report but the `report.truncated`.

## 12.6 What a finding may say

<a id="mts-fnd-011"></a>**MTS-FND-011** No member of a finding MUST hold a
string value, or part of one, taken from a supplied input: a string is
described by its type and length, numbers, booleans and `null` may be given,
but not a value, its length or a suggestion where its schema is `writeOnly`,
and a `suggestion` names only a value or member name the package declares.

<a id="mts-fnd-012"></a>**MTS-FND-012** A processor MUST carry text from the
package, a title or a description, into a finding exactly as the package
writes it, and MUST write each C0 control, DEL, C1 control and bidirectional
control in it, U+202A to U+202E and U+2066 to U+2069, as `\u` and four
lowercase hexadecimal digits wherever it writes a finding as JSON.
