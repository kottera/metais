# Sources and their terms

Everything in this package — the manifest and its guide, the project schema
and its meaning document, the templates and the examples — is written in
the METAIS project's own words for this package. It reproduces, adapts or
quotes no third-party text, so no third party's reuse terms apply to it.

The package is licensed with the rest of the repository, under the Apache
License 2.0, as its manifest's `license` says.

The repository it renders follows the public conventions of the tools it
is written for, which it names and does not copy:

- the layout of a Go module, `go.mod`'s `module` and `go` lines, and the
  `cmd/` and `internal/` directories, as the Go project documents them;
- the `SPDX-License-Identifier` line, which names a licence by its SPDX
  identifier; the package writes no licence text, and the README it renders
  tells its author to add one;
- the keys of a GitHub Actions workflow file and the public
  `actions/checkout` and `actions/setup-go` actions it names by reference;
- the instructions of a Dockerfile and the lines of a `.dockerignore`;
- the keys of an EditorConfig file.
