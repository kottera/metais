## What changes, and why

<!-- What the change does, and the issue it answers: "Closes #N". -->

## The specification comes first

<!--
A change to what a package means, or to what the tool does with one,
changes the specification in spec/ and the conformance vectors that prove
it, in this pull request. Where the tool and the specification disagree,
the tool is wrong. CONTRIBUTING.md, "Changing the format", has the order.
-->

- [ ] This changes no behaviour of the format or the tool, **or** it changes `spec/` and the conformance vectors that prove it.
- [ ] Every behaviour it adds or changes has a test that fails without it, and every new refusal has a negative test.
- [ ] I ran `regenerate.sh` and committed what it changed, if the change touches a generated file's source.

## How it was tested

<!-- The commands you ran and what they printed: at least `go test -race -count=1` over what you changed. -->

## Sign-off

- [ ] Every commit carries `Signed-off-by:`, made with `git commit -s`, certifying the [Developer Certificate of Origin](https://developercertificate.org/). CI checks every commit.
