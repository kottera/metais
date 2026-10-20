# Release

The release is one version of a product, as its users receive it. It
gives the heading and the list of the release notes, and every line of
the announcement.

- **product** is the product's name as its users know it, such as
  "Lantern". It is not the name of a repository or a team.
- **version** is the version the users install, such as "2.4.0". It names
  the release notes' file, `release-2.4.0.md`.
- **date** is the day the release shipped, written YYYY-MM-DD. It is not
  the day the notes were written.
- **highlight** is optional: one or two sentences on what matters most,
  written for a user deciding whether to upgrade. Leave it out when no
  change stands above the others.
- **changes** lists every change a user would notice, at least one, in
  the order the notes list them. Each has a **kind**, `added`, `changed`
  or `fixed`, and a **text**: one sentence saying what is different, in
  the user's words rather than the code's. "Exports keep their column
  order" is a good text; "Refactor the CSV writer" is not.
