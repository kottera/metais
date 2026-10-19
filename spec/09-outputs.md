# 9. Outputs

**Normative.** Areas: OUT.

The manifest declares the files a render produces. This chapter specifies the
`outputs` section, path templates, outputs repeated over a collection with
`for_each` and conditional on `when`, asset copies, and the shorthand that
mirrors the templates tree into the output.

## 9.1 Template outputs

A template output renders one file from one template under `templates/`.
Its `path` names the file, relative to the root of the output tree; a path
that holds no Liquid markup is written as it is.

<a id="mts-out-001"></a>**MTS-OUT-001** A processor MUST write a template
output to the path its declaration's `path` gives and, when the declaration
omits `path`, to the template's path relative to `templates/` with a final
`.liquid` removed: `ci/workflow.yml.liquid` is written to `ci/workflow.yml`,
and `notes.txt` to `notes.txt`.

<a id="mts-out-002"></a>**MTS-OUT-002** A processor writing to a file system
with POSIX modes MUST write a template output whose declaration has
`executable: true` with its owner's execute bit set, mode `0755` less the
process's umask, and every other output without it, mode `0644` less the
umask.
