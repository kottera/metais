// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package author

import (
	"errors"
	"path"
	"path/filepath"
	"strings"

	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/manifest"
)

// NameFromDir returns the package name init takes from dir, the directory
// as the caller wrote it: its last element (ADR-007 D1 item 2). It is
// false when dir is empty, or its last element is . or .., where the
// caller must give --name: the tool never looks up the working
// directory's name (ADR-009 D12).
func NameFromDir(dir string) (string, bool) {
	if dir == "" {
		return "", false
	}
	base := filepath.Base(dir)
	if base == "." || base == ".." || base == string(filepath.Separator) {
		return "", false
	}
	return base, true
}

// ValidName reports whether name satisfies ADR-002 D5's name rule.
func ValidName(name string) bool { return manifest.ValidName(name) }

// SuggestName returns a name ADR-002 D5's rule accepts, made from name by
// lower-casing ASCII letters, writing every run of other characters but .
// as one -, and dropping what is left empty; and false when nothing valid
// remains.
func SuggestName(name string) (string, bool) {
	var segs []string
	for _, seg := range strings.Split(name, ".") {
		var b strings.Builder
		dash := false
		for _, r := range seg {
			switch {
			case r >= 'A' && r <= 'Z':
				r += 'a' - 'A'
				fallthrough
			case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
				if dash && b.Len() > 0 {
					b.WriteByte('-')
				}
				b.WriteRune(r)
				dash = false
			default:
				dash = true
			}
		}
		if s := b.String(); s != "" {
			segs = append(segs, strings.TrimRight(s[:min(len(s), 63)], "-"))
		}
	}
	s := strings.Join(segs, ".")
	if len(s) > 128 {
		s = strings.TrimRight(s[:128], ".-")
	}
	if len(segs) > 0 && !ValidName(s) {
		// The leading segment is a Windows device name.
		s = segs[0] + "-package" + strings.TrimPrefix(s, segs[0])
	}
	return s, ValidName(s)
}

// MaxListed is how many of the entries found in a directory that is not
// empty init reports (ADR-007 D1 item 3).
const MaxListed = 10

// NotEmptyError is init's refusal of a directory that is not empty.
type NotEmptyError struct {
	// Entries are the first MaxListed names found, in byte order, that
	// make the directory not empty.
	Entries []string
	// More is whether there are entries beyond Entries.
	More bool
}

func (e *NotEmptyError) Error() string { return "author: the directory is not empty" }

// WriteError is a failure that stopped init (ADR-007 D1 item 6). Op and
// Reason are hostfs's; Written are the files written before it, in the
// order written, none of which was removed.
type WriteError struct {
	Op      string
	Reason  hostfs.Reason
	Written []string
}

func (e *WriteError) Error() string {
	return "author: init stopped at " + e.Op + ": " + string(e.Reason)
}

// Init writes the scaffold for a package called name into dir, the
// caller's own path for it, and returns the paths it wrote, relative to
// dir (ADR-007 D1). With dryRun it writes nothing and returns the paths it
// would write.
//
// dir is created, with its missing parents, when it does not exist. An
// existing dir must be empty, apart from the quiet list (IsQuiet), or Init
// refuses with a *NotEmptyError and writes nothing. A dir that is not a
// directory, or that cannot be opened or listed, is a *WriteError with
// nothing written. Init never overwrites, and it removes nothing, even
// after a failed write: what it wrote is in the *WriteError.
func Init(dir, name string, dryRun bool) ([]string, error) {
	files := Scaffold(name)
	if dir == "" {
		dir = "."
	}
	w, rest, err := hostfs.OpenNearest(dir)
	if err != nil {
		return nil, writeError(err, nil)
	}
	defer w.Close()
	if rest == "." {
		if err := checkEmpty(w, files); err != nil {
			return nil, err
		}
	}
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	if dryRun {
		return paths, nil
	}
	if err := w.MkdirAll(rest); err != nil {
		return nil, writeError(err, nil)
	}
	sub, err := w.Sub(rest)
	if err != nil {
		return nil, writeError(err, nil)
	}
	var written []string
	for _, f := range files {
		if parent := path.Dir(f.Path); parent != "." {
			if err := sub.MkdirAll(parent); err != nil {
				return written, writeError(err, written)
			}
		}
		if beforeCreate != nil {
			beforeCreate(f.Path)
		}
		if err := sub.Create(f.Path, false, strings.NewReader(string(f.Data))); err != nil {
			return written, writeError(err, written)
		}
		written = append(written, f.Path)
	}
	if err := w.Close(); err != nil {
		return written, writeError(err, written)
	}
	return written, nil
}

// beforeCreate, when set, runs before each file is created. Only this
// package's tests set it, to make a write fail part way through.
var beforeCreate func(path string)

// checkEmpty refuses the directory w when it holds anything but quiet
// entries, or a quiet entry the scaffold writes, since init never
// overwrites.
func checkEmpty(w *hostfs.Writer, files []File) error {
	entries, err := w.List(".", int(limits.Get(limits.Entries).Default))
	if hostfs.ReasonOf(err) == hostfs.TooLarge {
		return &NotEmptyError{More: true}
	}
	if err != nil {
		return writeError(err, nil)
	}
	e := &NotEmptyError{}
	for _, en := range entries {
		if IsQuiet(en) && !writes(files, en.Name) {
			continue
		}
		if len(e.Entries) == MaxListed {
			e.More = true
			break
		}
		e.Entries = append(e.Entries, en.Name)
	}
	if len(e.Entries) > 0 {
		return e
	}
	return nil
}

func writes(files []File, name string) bool {
	for _, f := range files {
		if f.Path == name {
			return true
		}
	}
	return false
}

// IsQuiet reports whether a top-level entry of a source directory is on
// ADR-007 D7 item 3's quiet list: a name beginning with . other than
// .compiled, or a regular file whose name ends in .metais.
func IsQuiet(e hostfs.Entry) bool {
	if strings.HasPrefix(e.Name, ".") {
		return e.Name != ".compiled"
	}
	return e.Kind == hostfs.KindFile && strings.HasSuffix(e.Name, ".metais")
}

// writeError is err, from hostfs, as a *WriteError after written.
func writeError(err error, written []string) error {
	var he *hostfs.Error
	if !errors.As(err, &he) {
		he = &hostfs.Error{Op: "write", Reason: hostfs.IOError}
	}
	return &WriteError{Op: he.Op, Reason: he.Reason, Written: written}
}
