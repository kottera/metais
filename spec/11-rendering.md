# 11. Rendering

**Normative.** Areas: RND, PTH.

A render turns a package and its inputs into files, and gives the same bytes
on every platform. This chapter specifies what makes a render deterministic,
how its outputs are staged and written all at once or not at all, when an
existing file may be overwritten, the dry run, rendering to a stream and the
tar profile that stream follows, and the rule every portable path obeys.
