# Articles

The articles are the site's content: a directory named `articles` in the
data directory, holding one Markdown file per article. Each file gives one
page, `articles/<stem>.html`, where the stem is the file's name without
`.md`, and one entry in the index and in the feed.

## Naming the files

The articles are read in the order of their file names, and the index and
the feed list them in the reverse of that order, so the last name comes
first. Start each name with the article's date, so that order is the order
the articles were published and the newest is at the top:

```
articles/
  2026-09-14-starting-a-field-notebook.md
  2026-10-02-what-to-write-down.md
```

The stem is the page's file name and part of its address in the feed, so
use lower-case letters, digits and hyphens, and do not rename a file once
the site is published: its page moves, and a feed reader sees a new entry.

## Frontmatter

Each file opens with frontmatter: a line `---`, YAML fields, and another
line `---`. Everything after it is the body.

```markdown
---
title: What to write down
date: 2026-10-02
summary: The five things every entry records, and why the weather is one of them.
tags: [method, weather]
---
Body text starts here.
```

- **title** is the article's headline, on its page, in the index and in the
  feed.
- **date** is the day it was published, as `YYYY-MM-DD`. The page and the
  index print it as a day, a month and a year; the feed gives it as the
  entry's update time, and its newest date as the feed's.
- **summary** is one or two sentences on what the article is about, shown
  beneath its title in the index and as the feed entry's summary. Leave it
  out and the index shows the title alone. Write it for someone deciding
  whether to read on: say what the reader learns, not that there is an
  article.
- **tags** are short topic words, printed after the date on the page and
  given as the feed entry's categories. Leave them out for none.
- **draft**, when `true`, keeps the article off the site: it gets no page
  and is in neither the index nor the feed. Leave it out, or write `false`,
  to publish.

## The body

The body is the article's text. It is written as plain Markdown blocks,
each separated from the next by a blank line, and this package writes each
block as one HTML element:

- a block that opens with `# ` or `## ` has a section heading as its first
  line, and one that opens with `### ` a subsection heading; any lines after
  the heading's are a paragraph beneath it (the article's title is the
  page's one top-level heading, so `#` and `##` both write the level beneath
  it);
- a block whose lines each open with `- ` is a bulleted list, one item per
  line;
- a block that opens with a line of three backticks is code, shown as it is
  written up to its closing line of three backticks; a code block holds no
  blank line;
- any other block is a paragraph, its line breaks read as spaces.

Every text is written escaped: inline Markdown such as `*emphasis*` or a
link, and any HTML in the body, is shown as the characters written, never
as markup. So a body can never change a page's structure or run a script,
and what the supplier wrote is what the reader sees.
