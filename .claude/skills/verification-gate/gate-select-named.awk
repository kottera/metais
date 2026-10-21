# Copyright 2026 Kottera
# SPDX-License-Identifier: Apache-2.0
#
# gate-select-named.awk prints the paths the Go files it reads name, for
# gate-select.sh's data rule, one per line: the directory of the file that
# names it (the file's package's, a path from the module root), a kind and
# the path, relative to the module root, as "<dir><TAB><kind><TAB><path>":
#
#   E<TAB><path>    the path, and every path below it
#   P<TAB><prefix>  every path with the prefix: a glob, a format verb or a
#                   concatenation follows it in the literal
#   F<TAB><file>    the one file at the root (emit says when)
#   S<TAB><path>    any path, in any directory, that holds the path as whole
#                   names (joined says when)
#
# A test may read a path from its own directory, where go test runs it, or
# from the root, through a go command run there or a helper that joins it,
# so each literal names both. A ".." above the root stops at the root.
#
# The arguments of a Join are read together: each run of literal arguments
# is one path, and those literals are not read alone. Every other
# double-quoted literal is read where it stands, in code, in comments and in
# raw strings.

# clean resolves "." and ".." in the slash-separated path s; a ".." above
# the root is dropped.
function clean(s,    n, i, c, kept, k, parts) {
	n = split(s, parts, "/")
	k = 0
	for (i = 1; i <= n; i++) {
		c = parts[i]
		if (c == "" || c == ".") continue
		if (c == "..") { if (k > 0) k--; continue }
		kept[++k] = c
	}
	s = ""
	for (i = 1; i <= k; i++) s = s (i > 1 ? "/" : "") kept[i]
	return s
}
function out(kind, s) {
	if (s != "") print dir "\t" kind "\t" s
}
# emit prints what the literal path s names, read from the package's
# directory and from the module root: "E" a path and everything below it,
# "P" every path with the prefix, and "F" one file at the root. A literal
# written without "./" or "../" and of one name ("docs", "internal") names
# only the root's file of that name: such a word is a class, a field or an
# entry name far more often than a directory, and a test reads a directory
# at the root by a Join or from its own directory, which name it as a path.
function emit(s, prefix, join,    rel, b, full) {
	rel = s ~ /^\.\.?(\/|$)/
	if (match(s, /[*?[%{\\]/)) {
		s = substr(s, 1, RSTART - 1)
		prefix = 1
	}
	if (!rel && !join && index(s, "/") == 0) {
		if (!prefix) out("F", s)
		return
	}
	for (b = 0; b < 2; b++) {
		full = b ? s : dir "/" s
		if (prefix) out("P", cleanpre(full))
		else out("E", clean(full))
	}
}
# cleanpre is clean for a prefix, whose last name may be incomplete.
function cleanpre(s,    head, tail) {
	if (!match(s, /\/[^\/]*$/)) return s == "." || s == ".." ? "" : s
	head = clean(substr(s, 1, RSTART - 1))
	tail = substr(s, RSTART + 1)
	if (tail == "." || tail == "..") { head = clean(head "/" tail); tail = "" }
	if (head == "") return tail
	return head "/" tail
}
# joined emits a run of a Join's literal arguments. After a base the test
# computes (Join(suite, "..", "statements.json")), a run that climbs with
# ".." leaves every directory the test names, so its tail names any path
# that holds it, from any directory: "S".
function joined(run, based) {
	if (based && run ~ /^\.\.(\/|$)/) {
		while (run ~ /^\.\.?(\/|$)/) sub(/^\.\.?(\/|$)/, "", run)
		if (match(run, /[*?[%{\\]/)) {
			run = substr(run, 1, RSTART - 1)
			sub(/[^\/]*$/, "", run)
		}
		out("S", clean(run))
		return
	}
	emit(run, 0, 1)
}
function blank(k,    b) {
	b = ""
	while (k-- > 0) b = b " "
	return b
}
# lit returns the end of the string literal that opens at i in line, or 0.
function lit(line, i,    n, c) {
	n = length(line)
	for (i++; i <= n; i++) {
		c = substr(line, i, 1)
		if (c == "\\") i++
		else if (c == "\"") return i
	}
	return 0
}
FNR == 1 {
	raw = 0
	dir = FILENAME
	if (!sub(/\/[^\/]*$/, "", dir)) dir = "."
}
# args reads the arguments of the call whose "(" ends just before i in
# line into a[1..nargs], each from at[] to to[], and returns whether the
# call closes on the line.
function args(line, i,    n, c, e, depth, arg, argfrom) {
	n = length(line)
	depth = 0; arg = ""; argfrom = i; nargs = 0
	for (; i <= n; i++) {
		c = substr(line, i, 1)
		if (c == "\"") { e = lit(line, i); if (!e) return 0; arg = arg substr(line, i, e - i + 1); i = e; continue }
		if (c == "(" || c == "[" || c == "{") depth++
		else if ((c == ")" || c == "]" || c == "}") && depth > 0) depth--
		else if ((c == "," || c == ")") && depth == 0) {
			a[++nargs] = arg; at[nargs] = argfrom; to[nargs] = i - 1
			arg = ""; argfrom = i + 1
			if (c == ")") return 1
			continue
		}
		arg = arg c
	}
	return 0
}
# unclosed reports whether a Join on line does not close on it.
function unclosed(line,    from, k) {
	from = 1
	while ((k = index(substr(line, from), "Join(")) > 0) {
		from += k - 1 + 5
		if (!args(line, from)) return 1
	}
	return 0
}
index($0, "\"") == 0 && index($0, "`") == 0 && index($0, "Join(") == 0 { next }
{
	line = $0
	# A Join split over lines is read as one line, up to twenty of them.
	for (folds = 0; folds < 20 && unclosed(line); folds++) {
		if ((getline more) <= 0) break
		line = line " " more
	}
	n = length(line)
	# Each Join whose arguments close: every run of literal arguments is
	# one path, and its literals are not read alone.
	from = 1
	while ((k = index(substr(line, from), "Join(")) > 0) {
		from += k - 1 + 5
		if (!args(line, from)) continue
		run = ""; inrun = 0; based = 0
		for (j = 1; j <= nargs; j++) {
			v = a[j]; gsub(/^[ \t]+|[ \t]+$/, "", v)
			if (v ~ /^"[^"\\]*"$/) {
				if (!inrun) runbased = based
				run = run (inrun ? "/" : "") substr(v, 2, length(v) - 2)
				inrun = 1
				line = substr(line, 1, at[j] - 1) blank(to[j] - at[j] + 1) substr(line, to[j] + 1)
			} else {
				if (inrun) joined(run, runbased)
				run = ""; inrun = 0; based = 1
			}
		}
		if (inrun) joined(run, runbased)
	}
	# Every other string literal, in code, comments and raw strings alike.
	comment = 0
	for (i = 1; i <= n; i++) {
		if (!match(substr(line, i), /["`'\/]/)) break
		i += RSTART - 1
		c = substr(line, i, 1)
		if (c == "`") { if (!comment) raw = !raw; continue }
		if (c == "/") {
			if (substr(line, i + 1, 1) == "/" && !raw) comment = 1
			continue
		}
		if (c == "'") {
			if (raw || comment) continue
			for (i++; i <= n; i++) { c = substr(line, i, 1); if (c == "\\") i++; else if (c == "'") break }
			continue
		}
		e = lit(line, i)
		if (!e) break
		s = substr(line, i + 1, e - i - 1)
		rest = substr(line, e + 1)
		emit(s, rest ~ /^[ \t]*\+/, 0)
		i = e
	}
}
