# Errata

An erratum corrects a statement of a released revision that is wrong: a
contradiction between statements, or between a statement and a vector; an
impossibility, which no implementation could obey; an ambiguity, which two
implementations have read two ways; or a security defect, where the text
accepts a package or an input that lets a package escape the security model of
chapter 14.

An erratum is fixed in the text at once, marking the statement it corrects
`amended by` or `withdrawn by` its identifier, and published in the next
revision. For a released revision the vectors are the tie-breaker: every
correction but a security one rewrites the text to say what the released
vectors already prove, and a security correction changes a released vector's
expectation only from acceptance to refusal.

Each erratum has the identifier `E-<revision>-<NNN>`, naming the revision it
corrects and numbered in order of report within that revision, and an entry of
this form:

```
## E-<revision>-<NNN> — <one-line title>
- Kind: contradiction | impossibility | ambiguity | security
- Affects: <statement identifier> (<revision>)
- Reported: <date>
- Correction: <the statement's text as corrected>
- Vectors: <the vectors added and changed>
- Files: <path> sha256:<new digest>   (one line per released file it changes)
- Applied: <date>; published in <revision>
```

No erratum has been recorded.
