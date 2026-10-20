# Project

The project is what a new Go repository is for. Every file the package
renders is written from it, and the names it gives decide where those files
go.

- **name** is the project's command, as a user types it: `weather`, or
  `weather-cli`. The command's source is `cmd/<name>/main.go`. The package
  that does its work is `internal/<package>/`, where the package's name is
  the project's name with its hyphens removed (`weathercli`), because a Go
  package name holds no hyphen. A name that would make a Go keyword, or
  `main`, is refused, since neither can name an importable package.
- **module** is the Go module path, which `go.mod` declares and every
  import of the project's own packages begins with. Start it with a domain
  the owner controls (`example.com/weather`), or with the repository's
  address where it will be hosted (`github.com/example/weather`).
- **description** says what the command does, in one line with no final
  full stop: "Prints the weather for a place". It is the README's first
  line, the command's doc comment, and what the command prints until its
  author makes it do more.
- **owner** is who holds the copyright, and **year** the year the
  repository is created. Every Go file begins with a copyright line naming
  both, and the README repeats it.
- **license** is the SPDX identifier of the licence the project is
  released under, such as `Apache-2.0` or `MIT`, or a `LicenseRef-`
  identifier for terms of the owner's own, such as
  `LicenseRef-Proprietary`. Every Go file's second line is an
  `SPDX-License-Identifier` line naming it, and the README names it too.
  The package writes no licence text: add it as `LICENSE` yourself.
- **go** is the Go release the module is written for, such as `1.24`,
  which is the default.
- **commands** lists further commands the repository holds, each with a
  name and a one-line summary. Each gets `cmd/<name>/main.go`. Give each a
  name of its own, and none the project's name, since two commands cannot
  share a directory.
- **ci**, when true, adds a GitHub Actions workflow,
  `.github/workflows/ci.yml`, which runs `scripts/check.sh` on every push
  to `main` and every pull request.
- **container**, when true, adds a `Dockerfile` that builds the command
  into an image holding the binary alone, and a `.dockerignore` that keeps
  the repository's history out of the build.

Leave an option out, or false, and its files are not written at all.
