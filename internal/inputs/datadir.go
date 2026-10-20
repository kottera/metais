// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"cmp"
	"errors"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/schema"
	"github.com/kottera/metais/internal/unitext"
)

// ErrDataDir is wrapped by ReadDataDir's refusal of the data directory
// itself: the caller's own argument, which the command line reports.
var ErrDataDir = errors.New("inputs: the data directory cannot be read")

// DataDirError is ReadDataDir's refusal of the directory the caller
// named. Reason is ADR-006 D9 item 8's closed set, or too_large for a
// directory holding more entries than collection-items, which bounds the
// data directory's listing as it bounds a collection's. It carries no
// path.
type DataDirError struct {
	Reason string
}

func (e *DataDirError) Error() string { return ErrDataDir.Error() + ": " + e.Reason }
func (e *DataDirError) Unwrap() error { return ErrDataDir }

// The reasons a data directory's entry is refused, as dirSource answers
// them: ADR-006 D9 item 8's closed set, and three of hostfs's own.
const (
	readTooLarge   = "too_large"
	readLink       = "link"
	readNotRegular = "not_regular_file"
	readIsDir      = "is_directory"
	readIOError    = "io_error"
)

// entryKind is what a directory entry is, from its Lstat: nothing is
// followed to find it (ADR-009 D7 item 2).
type entryKind uint8

const (
	entryFile entryKind = iota + 1
	entryDir
	entryLink
	entryOther
)

type dirEntry struct {
	name string // the host's name, as its bytes are
	kind entryKind
	size int64
}

// dirSource is the read side of a directory the layer reads: its
// listings and its regular files, each by a name relative to it. Every
// error is a reason, which names no path.
type dirSource struct {
	list func(name string, max int) ([]dirEntry, string)
	read func(name string, limit int64) ([]byte, string)
}

// ReadDataDir reads the data directory at path, the caller's own name for
// it (ADR-003 D16, ADR-009 D7): the step of D18 item 1.1. path may be a
// symbolic link, which the caller named (ADR-009 D7 item 4). Inside it,
// the layer lists the entries and opens only those a declared input could
// be, by D16 item 2's naming rule, and the items of those collections;
// no link is followed and only regular files are opened. Every finding is
// recorded in the supply.
//
// It returns an error wrapping ErrDataDir, a *DataDirError, only when the
// directory itself cannot be read; the supply is then unchanged. It may be
// called once per supply, as --data is given once (ADR-008 D7).
//
// This is the layer's one use of internal/hostfs, and only of its read
// side: OpenDir, and the directory's List and ReadFile.
func (s *Supply) ReadDataDir(path string) error {
	if s.dataDir {
		panic("inputs: ReadDataDir called twice on one supply")
	}
	d, err := hostfs.OpenDir(path)
	if err != nil {
		r := hostfs.ReasonOf(err)
		switch {
		case r == hostfs.NotDirectory:
			r = hostfs.NotRegularFile
		case !r.IsOS():
			r = hostfs.IOError
		}
		return &DataDirError{Reason: string(r)}
	}
	defer d.Close()
	reason := func(err error) string {
		switch r := hostfs.ReasonOf(err); r {
		case hostfs.TooLarge:
			return readTooLarge
		case hostfs.Link:
			return readLink
		case hostfs.NotRegularFile, hostfs.NotDirectory:
			return readNotRegular
		case hostfs.IsDirectory:
			return readIsDir
		case hostfs.NotFound, hostfs.PermissionDenied:
			return string(r)
		}
		// hostfs.Changed and anything else: the file was not the one
		// listed, or the read failed.
		return readIOError
	}
	src := dirSource{
		list: func(name string, max int) ([]dirEntry, string) {
			es, err := d.List(name, max)
			if err != nil {
				return nil, reason(err)
			}
			out := make([]dirEntry, len(es))
			for i, e := range es {
				k := entryOther
				switch e.Kind {
				case hostfs.KindFile:
					k = entryFile
				case hostfs.KindDir:
					k = entryDir
				case hostfs.KindLink:
					k = entryLink
				}
				out[i] = dirEntry{name: e.Name, kind: k, size: e.Size}
			}
			return out, ""
		},
		read: func(name string, limit int64) ([]byte, string) {
			b, err := d.ReadFile(name, limit)
			if err != nil {
				return nil, reason(err)
			}
			return b, ""
		},
	}
	top, r := src.list(".", int(s.lim.Value(limits.CollectionItems)))
	if r != "" {
		return &DataDirError{Reason: r}
	}
	s.dataDir = true
	s.readEntries(src, top)
	return nil
}

// readEntries reads the data directory's top-level entries (ADR-003 D16).
func (s *Supply) readEntries(src dirSource, top []dirEntry) {
	type claim struct {
		entry dirEntry
		ext   string
	}
	byInput := map[string][]claim{}
	for _, e := range top {
		if strings.HasPrefix(e.name, ".") {
			continue
		}
		name, rest, hasDot := strings.Cut(e.name, ".")
		ext := ""
		if hasDot {
			ext = "." + rest
		}
		if _, declared := s.l.byName[name]; !declared {
			l := s.fileLoc(e.name)
			s.add(finding.New(CodeFileUnknown, finding.Subject{Kind: finding.SubjectInput, Name: name}, &l, finding.Params{}))
			continue
		}
		byInput[name] = append(byInput[name], claim{e, ext})
	}
	for _, in := range s.l.inputs {
		claims := byInput[in.Name]
		if len(claims) == 0 {
			continue
		}
		subj := s.l.subject(in.Name)
		fi := &fileInput{}
		s.files[in.Name] = fi
		if len(claims) > 1 {
			// Two entries for one input (D16 item 4): neither is read.
			first := s.fileLoc(claims[0].entry.name)
			for _, c := range claims[1:] {
				l := s.fileLoc(c.entry.name)
				s.add(finding.New(CodeFileDuplicate, subj, &l, finding.Params{},
					finding.WithRelated(finding.Related{Location: first, Note: noteFirstEntry})))
			}
			fi.broken = true
			continue
		}
		c := claims[0]
		fi.file = c.entry.name
		l := s.fileLoc(c.entry.name)
		switch {
		case c.entry.kind == entryLink:
			s.add(finding.New(CodeLink, subj, &l, finding.Params{}))
			fi.broken = true
		case in.Collection && c.entry.kind == entryDir && c.ext == "":
			s.readCollection(src, in, fi)
		case in.Collection && c.entry.kind == entryDir:
			// D15 item 2: the collection's directory is <name>/.
			s.add(finding.New(CodeDirectoryName, subj, &l, finding.Params{}))
			fi.broken = true
		case in.Collection:
			// D15 item 2: a collection is a directory, never a file.
			s.add(finding.New(CodeAsFile, subj, &l, finding.Params{}))
			fi.broken = true
		case c.entry.kind == entryDir:
			s.add(finding.New(CodeNotRegularFile, subj, &l, finding.Params{"kind": finding.String(kindDirectory)}))
			fi.broken = true
		case c.entry.kind == entryOther:
			s.add(finding.New(CodeNotRegularFile, subj, &l, finding.Params{"kind": finding.String(kindSpecial)}))
			fi.broken = true
		default:
			form, ok := formOf(c.ext, in.Forms)
			if !ok {
				s.add(finding.New(CodeExtensionNotAccepted, subj, &l, finding.Params{"accepted": accepted(in.Forms)}))
				fi.broken = true
				break
			}
			b, ok := s.readFile(src, in, c.entry, c.entry.name)
			if !ok {
				fi.broken = true
				break
			}
			p := s.l.reader(in, s.lim, s.values).read(b, form, l, "", []schema.Step{schema.Member(in.Name)})
			s.add(p.findings...)
			if p.value == nil {
				fi.broken = true
				break
			}
			fi.value, fi.partial, fi.raw = p.value, p.partial, b
			fi.broken = len(p.findings) > 0
		}
	}
}

// fileLoc is the location of the data-directory entry at name, a path
// relative to the data directory (ADR-006 D4 item 4): an example's, read
// from the package, is located at its package entry.
func (s *Supply) fileLoc(name string) finding.Location {
	if s.locPrefix != "" {
		return finding.Location{Origin: finding.OriginExample, File: s.locPrefix + name}
	}
	return finding.Location{Origin: finding.OriginDataDir, File: name}
}

// readFile reads one regular file of in, within input-size and what is
// left of inputs-total (ADR-009 D3), reporting why it cannot.
func (s *Supply) readFile(src dirSource, in manifest.Input, e dirEntry, name string) ([]byte, bool) {
	subj := s.l.subject(in.Name)
	l := s.fileLoc(name)
	if e.size > s.lim.Value(limits.InputSize) {
		s.add(s.lim.Exceeded(limits.InputSize, subj, l, e.size, true))
		return nil, false
	}
	if !s.charge(e.size, subj, l) {
		return nil, false
	}
	b, r := src.read(name, s.lim.Value(limits.InputSize))
	switch r {
	case "":
	case readTooLarge:
		s.add(s.lim.Exceeded(limits.InputSize, subj, l, 0, false))
		return nil, false
	case readLink:
		s.add(finding.New(CodeLink, subj, &l, finding.Params{}))
		return nil, false
	case readNotRegular, readIsDir:
		s.add(finding.New(CodeNotRegularFile, subj, &l, finding.Params{"kind": finding.String(kindSpecial)}))
		return nil, false
	default:
		s.add(finding.New(CodeNotReadable, subj, &l, finding.Params{"reason": finding.String(r)}))
		return nil, false
	}
	// The file may have grown since it was listed.
	if extra := int64(len(b)) - e.size; extra > 0 && !s.charge(extra, subj, l) {
		return nil, false
	}
	return b, true
}

// readCollection reads the directory of a collection input (ADR-003 D15).
func (s *Supply) readCollection(src dirSource, in manifest.Input, fi *fileInput) {
	subj := s.l.subject(in.Name)
	dirLoc := s.fileLoc(in.Name)
	fi.collection = true
	entries, r := src.list(in.Name, int(s.lim.Value(limits.CollectionItems)))
	switch r {
	case "":
	case readTooLarge:
		s.add(s.lim.Exceeded(limits.CollectionItems, subj, dirLoc, 0, false))
		fi.broken = true
		return
	default:
		s.add(finding.New(CodeNotReadable, subj, &dirLoc, finding.Params{"reason": finding.String(r)}))
		fi.broken = true
		return
	}
	fi.listed = true
	// The collection's array is a value of its own (ADR-009 Amendment A4
	// item 1).
	if !s.values.Add(1) {
		s.add(s.lim.Exceeded(limits.InputValues, subj, dirLoc, 0, false))
		fi.broken = true
		return
	}
	// An item's name is its file name in NFC; items are ordered by the
	// UTF-8 bytes of those names (D15 items 5 and 6).
	type candidate struct {
		e       dirEntry
		name    string
		refused string // why the name is refused, or ""
	}
	var cands []candidate
	for _, e := range entries {
		if strings.HasPrefix(e.name, ".") {
			continue
		}
		switch {
		case !utf8.ValidString(e.name):
			cands = append(cands, candidate{e, e.name, nameEncoding})
		case strings.ContainsAny(e.name, `\:`):
			// A name hostfs will not resolve, which no file system the
			// package is portable to can hold.
			cands = append(cands, candidate{e, e.name, nameCharacter})
		default:
			cands = append(cands, candidate{e, unitext.NFC(e.name), ""})
		}
	}
	slices.SortFunc(cands, func(a, b candidate) int {
		return cmp.Or(strings.Compare(a.name, b.name), strings.Compare(a.e.name, b.e.name))
	})

	stems := map[string]int{} // each stem's caseless key, to the first item holding it
	for _, c := range cands {
		file := in.Name + "/" + c.e.name
		l := s.fileLoc(file)
		it := &item{name: c.name, stem: strings.TrimSuffix(c.name, path.Ext(c.name)), file: file, index: len(fi.items)}
		fi.items = append(fi.items, it)
		fail := func(f finding.Finding) {
			s.add(f)
			fi.broken = true
		}
		if c.refused != "" {
			fail(finding.New(CodeItemName, subj, &l, finding.Params{"reason": finding.String(c.refused)}))
			continue
		}
		switch c.e.kind {
		case entryLink:
			fail(finding.New(CodeLink, subj, &l, finding.Params{}))
			continue
		case entryDir:
			fail(finding.New(CodeSubdirectory, subj, &l, finding.Params{}))
			continue
		case entryOther:
			fail(finding.New(CodeNotRegularFile, subj, &l, finding.Params{"kind": finding.String(kindSpecial)}))
			continue
		}
		key := unitext.CaselessKey(it.stem)
		if j, dup := stems[key]; dup {
			first := fi.items[j]
			fail(finding.New(CodeStemDuplicate, subj, &l, finding.Params{"case": finding.Bool(first.stem != it.stem)},
				finding.WithRelated(finding.Related{Location: s.fileLoc(first.file), Note: noteCollides})))
			continue
		}
		stems[key] = it.index
		form, ok := formOf(path.Ext(c.name), in.Forms)
		if !ok {
			fail(finding.New(CodeExtensionNotAccepted, subj, &l, finding.Params{"accepted": accepted(in.Forms)}))
			continue
		}
		it.form = form
		b, ok := s.readFile(src, in, c.e, file)
		if !ok {
			fi.broken = true
			continue
		}
		at := []schema.Step{schema.Member(in.Name), schema.Index(it.index)}
		p := s.l.reader(in, s.lim, s.values).read(b, form, l, finding.Pointer(itoa(it.index)), at)
		s.add(p.findings...)
		if p.value == nil {
			fi.broken = true
			continue
		}
		fi.broken = fi.broken || len(p.findings) > 0
		it.value, it.raw, it.ok = p.value, b, true
	}
}
