# Site configuration

The site configuration is what every page and the feed say about the site
as a whole. It is one file, `site.yaml` or `site.json`, in the data
directory.

- **title** is the site's name. It heads the index, ends every article
  page's window title, sits at the top of every page as the link home, and
  is the feed's title. Keep it short: "Field Notes", not a sentence.
- **url** is the absolute address the site is published at, such as
  `https://notes.example.org/` or `https://example.org/notes/`. The pages
  link to one another by relative paths and do not need it; the feed does,
  because a feed reader resolves nothing against the page it found the feed
  on. The feed's identifier and every entry's link are made from it, with a
  `/` added when it does not end with one. Give the address the site will
  live at, not a preview address: a feed reader treats an entry whose
  identifier changes as a new entry.
- **description** is one or two sentences about the site, printed beneath
  the index's heading and given as the feed's subtitle. Leave it out and
  neither is written.
- **author** is who writes the site: a person's name or a team's. Every
  page's footer names it, and it is the feed's author.
- **language** is the language the site is written in, as a BCP 47 tag:
  `en`, `fr`, `pt-BR`. It sets the pages' `lang` and the feed's `xml:lang`,
  which screen readers and translation tools read. It is `en` when left out.

```yaml
title: Field Notes
url: https://notes.example.org/
description: Short notes on keeping a field notebook.
author: Sam Rivera
language: en
```
