// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"slices"
	"strings"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/liquid/value"
)

// File is one entry of a data directory built in memory: a regular file
// at Path, holding Data, or, when Path ends in "/", a directory. Paths are
// relative to the directory and "/"-separated, and a file's parent
// directories are entries of their own.
type File struct {
	Path string
	Data []byte
}

// ReadFiles reads files as the data directory, as ReadDataDir reads one on
// the host (ADR-003 D16): the sample generator's data, built in memory and
// loaded through the same reader as a supplier's, so what is validated is
// exactly what a data directory holding these files gives (ADR-007 D9 item
// 22). It may be called once per supply, in ReadDataDir's place.
func (s *Supply) ReadFiles(files []File) {
	if s.dataDir {
		panic("inputs: a supply's data directory read twice")
	}
	dirs := map[string][]dirEntry{".": nil}
	data := map[string][]byte{}
	for _, f := range files {
		name, dir := strings.CutSuffix(f.Path, "/")
		parent, base := ".", name
		if i := strings.LastIndexByte(name, '/'); i >= 0 {
			parent, base = name[:i], name[i+1:]
		}
		e := dirEntry{name: base, kind: entryFile, size: int64(len(f.Data))}
		if dir {
			e.kind = entryDir
			if _, ok := dirs[name]; !ok {
				dirs[name] = nil
			}
		} else {
			data[name] = f.Data
		}
		dirs[parent] = append(dirs[parent], e)
	}
	src := dirSource{
		list: func(name string, max int) ([]dirEntry, string) {
			es, ok := dirs[name]
			switch {
			case !ok:
				return nil, readIOError
			case len(es) > max:
				return nil, readTooLarge
			}
			return slices.Clone(es), ""
		},
		read: func(name string, limit int64) ([]byte, string) {
			b, ok := data[name]
			switch {
			case !ok:
				return nil, readIOError
			case int64(len(b)) > limit:
				return nil, readTooLarge
			}
			return b, ""
		},
	}
	top, _ := src.list(".", len(dirs["."]))
	s.dataDir = true
	s.readEntries(src, top)
}

// Fixed assembles the supply's arguments into the inputs object as
// Validate would (ADR-003 D18), and returns each input's value by name,
// before any default is applied and without validating it, with every
// finding the supply's parsing and assembly gave. An input no argument
// supplies, or whose argument did not parse, has no value. These are the
// values a sample generator takes as fixed points (ADR-007 D9 item 2);
// the supply is spent, and Validate may not be called on it.
func (s *Supply) Fixed() (map[string]value.Value, []finding.Finding) {
	if s.validated {
		panic("inputs: Fixed called on a spent supply")
	}
	s.validated = true
	root, states := s.assemble()
	out := map[string]value.Value{}
	for _, in := range s.l.inputs {
		n, ok := root.obj[in.Name]
		if !ok || s.broken[in.Name] || states[in.Name].refused {
			continue
		}
		out[in.Name] = n.value()
	}
	return out, slices.Clone(s.findings)
}
