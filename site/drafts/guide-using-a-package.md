# Using a package

Someone gave you a `.metais` file. This guide takes you from that file to
the document it makes, and on to rendering it from a program.

It assumes `metais` is [installed](/install/). Its example is a small
package, `status-widget-1.0.0.metais`, which renders an HTML fragment
showing a service's status. Every command works the same way on any
package: put your own file's name where this guide has
`status-widget-1.0.0.metais`.

**Do not unzip the package.** A `.metais` file is a zip archive, but
`metais` reads it in place and verifies every entry as it reads it. Every
command below takes the file exactly as you received it.

## The route

1. Find out what the package needs: `metais inspect`.
2. Read what each input means, and list its fields: `metais schema`.
3. Put your data in a data directory, one file per input.
4. Check it: `metais validate`.
5. Make the document: `metais render`.

Every command ends with a `Next:` list of the commands to run after it, so
you can follow the route from the tool itself.
`metais help <command>` explains any command in full.

## 1. Find out what the package needs

    metais inspect status-widget-1.0.0.metais

`inspect` verifies the whole package, then prints what it is: its title
and description, the author's guide, each input with the file names it is
read from, the outputs it renders, its references and examples, and the
commands to run next.

Read the guide. It is the author's own instructions for supplying the
data, and it often says where the data comes from. For this package, the
one input is `status`, read from `status.json`, and the one output is
`status.html`.

## 2. Read what each input means

    metais schema status-widget-1.0.0.metais --fields

`schema --fields` lists every field of every input, one line each: where
the field is, its type, whether it is required, its constraints, and its
title. For this package:

    /status                     object  required  additionalProperties=false  Service status
    /status/components          array   optional  maxItems=20 minItems=1      Components
    /status/components/*/name   string  required  maxLength=80 minLength=1    Component name
    /status/components/*/state  any     required  enum=["operational",…]      Component state
    /status/message             string  optional  maxLength=280 minLength=1   Message
    /status/service             string  required  maxLength=80 minLength=1    Service
    /status/state               any     required  enum=["operational",…]      State
    /status/updated             string  required  format="date-time"          Updated

A field such as a component's `name` is required only when its parent,
here a component, is given.

Give it an input's name, `metais schema status-widget-1.0.0.metais status
--fields`, to list one input. Leave out `--fields` to print the whole JSON
Schema, with each field's description.

If someone asks you what the package needs from them, this listing is the
answer: every input, and every required field of each.

Many packages also give an input a meaning document, which explains what
the data is and what good data looks like. When a package has one,
`inspect` lists it in its `Next:` lines as a `metais read` of the
package's `inputs/` entry: run it, and read the whole document before you
gather the data. This package's single input needs none.

## 3. Put your data in a data directory

Make a directory, here `data`, and put one file in it for each input,
named as `inspect` listed it: `data/status.json`.

    {
      "service": "Payments API",
      "state": "degraded",
      "updated": "2026-10-23T09:30:00Z",
      "message": "Card payments are slower than usual."
    }

**The data is yours to supply.** A real document is made only from real
facts. If you do not have a value the package requires, find it or ask
for it. Do not guess it, and do not copy it from the package's examples:
an example is the author's illustration, kept to test the package, and
none of its values are yours.

## 4. Check your data

    metais validate status-widget-1.0.0.metais --data data

`validate` reports every problem in one run, each with the file, line and
column it is at, the field it concerns, the rule it breaks and how to fix
it:

    status.json:4:14: error: Updated is not in the date-time format. [input.format]
      at /status/updated (input status: Service status)
      …
      hint: Write a value in the date-time format.
    1 error, 0 warnings, 0 info

Fix what it reports and run it again, until it reports `0 errors`.

## 5. Make the document

    metais render status-widget-1.0.0.metais --data data --out widget

`render` checks the data again, renders every output in memory, and
writes them into the new directory `widget` only when all of them
succeed. It lists each file it wrote, with its size and digest.

- `--dry-run` lists what it would write, and writes nothing.
- `--out` must name a new or empty directory. To render again into the
  same one, add `--overwrite`: it replaces the files the render writes,
  and leaves every other file there alone.

Rendering is pure. Nothing from the package runs, nothing is fetched from
the network, and the same package and data give the same bytes on every
platform.

## See what a package produces, before you have the data

To preview a package, render it from data that `metais` generates from
the package's schemas:

    metais render status-widget-1.0.0.metais --sample --out preview

The values are fictional, so the preview shows the document's shape, not
your document. Show it as a preview, and never as a real document.

The package's examples are not a preview: they are the author's tests.

## Start from generated data

To start your data from a file of the right shape, generate one and
replace every value with your own:

    metais sample status-widget-1.0.0.metais --out data

`--seed <n>` generates different values, for when you want fictional data
of your own rather than the default set:

    metais sample status-widget-1.0.0.metais --seed 7 --out data7

## Render from a program

A package is often filled in by a program: a nightly job that fetches the
status and renders the widget. The job writes its data where the package
expects it, and runs `render`. Here is the job as `render-status.sh`:

    #!/bin/sh
    # Renders the status widget from data/status.json into widget/.
    set -eu
    metais render status-widget-1.0.0.metais --data data --out widget --overwrite

`render` validates the data first, and writes nothing when any of it is
in error, so the job never publishes a broken document. Its exit status
says what happened: `0` when the document was written, `1` when the data
is invalid, and `4` or `5` when the package or the destination was
refused. `metais help exit-codes` lists every one.

To test the job before it has real data, give it generated data:

    metais sample status-widget-1.0.0.metais --out data
    ./render-status.sh

A program that wants to read the result can add `--json` to any command,
which prints one JSON document: whether it succeeded, every finding with
its stable code, and what was written. `metais help json` describes every
document.

## Escape hatches

The route above is the one to follow. When it does not fit, these are
there:

- **Values on the command line.** `--set /status/state=outage` sets one
  value over the data directory's; `--set-json` sets a JSON value, and
  `--inputs-json` supplies whole inputs. `metais help inputs` says which
  source wins.
- **One file to standard output.**
  `metais render status-widget-1.0.0.metais --data data --out - --only status.html`
  writes that file's bytes alone.
- **Reading the package.** `metais ls`, `metais read`, `metais outline`
  and `metais grep` list, print, outline and search its entries, in place:

      metais ls status-widget-1.0.0.metais -r
      metais grep status-widget-1.0.0.metais degraded references/

- **Changing the package itself.** That is writing a package, not using
  one: `metais unpack` turns it back into a source directory.
  [Writing a package](/guide/writing-a-package/) explains the rest.
- **An agent doing this for you.** `metais skill` installs a skill that
  teaches a coding agent this route.
