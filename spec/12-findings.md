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
| `internal.error` | error | internal | invocation | `tool_version`, `phase` | 1.0 | — |
| `limit.invalid` | error | usage | invocation | `reason`; optional: `name`, `default`, `unit`, `value` | 1.0 | — |
| `report.truncated` | info | content | report | `listed`, `omitted` | 1.0 | — |
| `usage.bad_value` | error | usage | invocation | `argument`, `flag`, `reason`; optional: `allowed` | 1.0 | — |
| `usage.missing_command` | error | usage | invocation | — | 1.0 | — |
| `usage.repeated_flag` | error | usage | invocation | `argument`, `flag` | 1.0 | — |
| `usage.unexpected_operand` | error | usage | invocation | `argument`, `command` | 1.0 | — |
| `usage.unknown_command` | error | usage | invocation | `argument` | 1.0 | — |
| `usage.unknown_flag` | error | usage | invocation | `argument`, `reason`; optional: `flag`, `command` | 1.0 | — |
| `usage.unknown_topic` | error | usage | invocation | `argument` | 1.0 | — |
<!-- The end of the generated code catalog. -->
