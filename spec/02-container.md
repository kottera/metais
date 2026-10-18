# 2. Container

**Normative.** Areas: CON.

A packed package is a zip archive in a canonical profile, in which every header
field is fixed except an entry's compressed data, its checksum and its sizes.
This chapter specifies that profile, the names an entry may carry, the closed
set of top-level names a package lays out, the `.compiled/` files a packed
package carries, and the digest of every entry and of the package as a whole,
which a processor verifies before it does anything else with a package.
