# Security policy

`metais` is built to open a package from a stranger safely. A package is
untrusted input, and a way to make the tool do something with one that the
specification says it must not is a security problem we want to hear
about.

## Reporting a vulnerability

Report it privately through GitHub's private vulnerability reporting:
open the repository's **Security** tab and choose **Report a
vulnerability**. Only the reporter and the project's maintainers see the
report and the discussion that follows.

**Do not open a public issue, pull request or discussion** for a
vulnerability, and do not describe it anywhere public before a fix is
released.

## What is in scope

**The tool's handling of an untrusted package**, and of the inputs given
with it. Chapter 14 of the specification, [`spec/14-security.md`](spec/14-security.md),
is the contract. A finding is in scope when, with a package, an input or a
command line it is given, `metais`:

- reads an entry from outside the package, or reads or writes a file
  outside the paths the command names: a traversal, an absolute path, a
  link, a device name or a case or Unicode trick that escapes confinement;
- lets a fact about the host reach a template, an output, a finding or a
  message: an environment variable, a file outside the package and the
  named inputs, the working directory, the user, the host name, the
  process or the clock;
- runs anything a package carries, or touches the network;
- exhausts memory, disk or CPU past the limits it declares: a zip bomb,
  an expansion the limits on size, count, ratio or depth do not stop;
- accepts a package the specification says must be refused, where the
  difference lets a package escape any of the above;
- crashes or hangs on a package or input.

**The install routes:** the install script served from metais.org, the
Homebrew tap, the release archives with their `SHA256SUMS`, and
`go install` of the module. A finding is in scope when a route can be
made to install bytes other than the ones the release published, to skip
or defeat the checksum check, or to write outside the directory it says it
installs to.

A defect in the specification's text that lets a package escape the
security model of chapter 14 is in scope too, and is corrected as a
security erratum ([`spec/ERRATA.md`](spec/ERRATA.md)).

Out of scope: what a rendered output does once you run or publish it
yourself (the tool renders text; it does not vouch for it), a problem that
needs an attacker who can already change your `metais` binary or your
machine, and a vulnerability in a dependency that `metais` cannot reach.
Report that last one to the dependency's own project.

## What to include

- what the problem is and what an attacker gains from it;
- the `metais --version` output, and your operating system and
  architecture;
- the smallest package, inputs and command line that reproduce it; attach
  the package itself (a packed `.metais` file, or its source directory),
  not a link to one;
- what happened and what you expected, with the command's output and its
  `--json` output where it has one;
- for an install route, the route, the command you ran and its full
  output.

## What to expect

We will acknowledge the report in the advisory, confirm or rule out the
problem, and keep you told of progress there. A confirmed vulnerability is
fixed in a new release of the tool and, where the text was at fault, in an
erratum to the specification, and is disclosed in a GitHub security
advisory once the fix is released. We credit you in the advisory unless
you ask us not to.

Fixes are made to the latest release. Report against it, or against
`main`, where you can.
