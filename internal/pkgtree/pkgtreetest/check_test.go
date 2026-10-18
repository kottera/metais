// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package pkgtreetest

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/pkgtree"
)

// fake is a Tree that answers as the in-memory tree over its files does,
// except where one of its functions says otherwise. Each control of
// TestCheckDetects breaks one clause of the contract with it.
type fake struct {
	mem  pkgtree.Tree
	list func() []string
	stat func(path string) (pkgtree.Info, error)
	open func(path string) (io.ReadCloser, error)
}

func (f *fake) List() []string {
	if f.list != nil {
		return f.list()
	}
	return f.mem.List()
}

func (f *fake) Stat(path string) (pkgtree.Info, error) {
	if f.stat != nil {
		return f.stat(path)
	}
	return f.mem.Stat(path)
}

func (f *fake) Open(path string) (io.ReadCloser, error) {
	if f.open != nil {
		return f.open(path)
	}
	return f.mem.Open(path)
}

// controlFiles are the entries every control's tree is built over.
func controlFiles() map[string][]byte {
	return map[string][]byte{
		"metais.yaml":                 []byte("metais: \"1.0\"\n"),
		"inputs/customer.schema.json": []byte(`{"type":"object"}`),
		"templates/report.md.liquid":  []byte("Hello {{ name }}\n"),
	}
}

// readerFunc is an io.ReadCloser built from a Read function.
type readerFunc func([]byte) (int, error)

func (r readerFunc) Read(b []byte) (int, error) { return r(b) }
func (r readerFunc) Close() error               { return nil }

// failingCloser is a reader of real content whose Close fails.
type failingCloser struct{ io.Reader }

func (failingCloser) Close() error { return errors.New("close failed") }

// TestCheckDetects is Check's control: a tree that breaks one clause of the
// contract must be caught, naming that clause, so a Check gone blind fails
// the build. The in-memory tree passing Check means nothing without it.
func TestCheckDetects(t *testing.T) {
	notExist := func(op, path string) error {
		return &pkgtree.PathError{Op: op, Path: path, Err: pkgtree.ErrNotExist}
	}
	tests := []struct {
		name  string
		build func(f *fake)
		want  func(files map[string][]byte) // adjusts what Check is told to expect
		catch string
	}{
		{
			name:  "List out of order",
			build: func(f *fake) { f.list = func() []string { l := f.mem.List(); slices.Reverse(l); return l } },
			catch: "is not before",
		},
		{
			name:  "List names an entry twice",
			build: func(f *fake) { f.list = func() []string { l := f.mem.List(); return append(l, l[len(l)-1]) } },
			catch: "listed twice",
		},
		{
			name:  "List leaves an entry out",
			build: func(f *fake) { f.list = func() []string { l := f.mem.List(); return l[1:] } },
			catch: "is not listed",
		},
		{
			name:  "List names an entry that should not be there",
			want:  func(files map[string][]byte) { delete(files, "metais.yaml") },
			catch: `"metais.yaml" is listed but is not a wanted entry`,
		},
		{
			name: "List shares its slice",
			build: func(f *fake) {
				shared := f.mem.List()
				f.list = func() []string { return shared }
			},
			catch: "changed what List returns next",
		},
		{
			name: "List names a directory",
			build: func(f *fake) {
				f.list = func() []string { return append([]string{"inputs/"}, f.mem.List()...) }
			},
			catch: `"inputs/" ends with /`,
		},
		{
			name: "List names a file that is also a directory",
			build: func(f *fake) {
				f.list = func() []string { return append([]string{"inputs"}, f.mem.List()...) }
			},
			catch: `"inputs" is both an entry and a directory`,
		},
		{
			name: "Stat has the wrong size",
			build: func(f *fake) {
				f.stat = func(p string) (pkgtree.Info, error) {
					info, err := f.mem.Stat(p)
					info.Size++
					return info, err
				}
			},
			catch: ".Size = ",
		},
		{
			name: "Stat has the wrong path",
			build: func(f *fake) {
				f.stat = func(p string) (pkgtree.Info, error) {
					info, err := f.mem.Stat(p)
					info.Path = strings.ToUpper(info.Path)
					return info, err
				}
			},
			catch: ".Path = ",
		},
		{
			name: "Stat fails on an entry",
			build: func(f *fake) {
				f.stat = func(p string) (pkgtree.Info, error) { return pkgtree.Info{}, notExist("stat", p) }
			},
			catch: `Stat("metais.yaml"): stat "metais.yaml": no such entry`,
		},
		{
			name: "Open yields other bytes",
			build: func(f *fake) {
				f.open = func(p string) (io.ReadCloser, error) {
					if _, err := f.mem.Stat(p); err != nil {
						return nil, notExist("open", p)
					}
					info, _ := f.mem.Stat(p)
					return io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("?"), int(info.Size)))), nil
				}
			},
			catch: "bytes that are not the entry's",
		},
		{
			name: "Open's reader never ends",
			build: func(f *fake) {
				f.open = func(p string) (io.ReadCloser, error) {
					if _, err := f.mem.Stat(p); err != nil {
						return nil, notExist("open", p)
					}
					return readerFunc(func(b []byte) (int, error) { b[0] = 'x'; return 1, nil }), nil
				}
			},
			catch: "yields more than the entry's",
		},
		{
			name: "Open's reader stalls without EOF",
			build: func(f *fake) {
				f.open = func(p string) (io.ReadCloser, error) {
					if _, err := f.mem.Stat(p); err != nil {
						return nil, notExist("open", p)
					}
					return readerFunc(func([]byte) (int, error) { return 0, nil }), nil
				}
			},
			catch: "does not reach EOF",
		},
		{
			name: "Open returns one shared reader",
			build: func(f *fake) {
				readers := map[string]io.ReadCloser{}
				f.open = func(p string) (io.ReadCloser, error) {
					if r, ok := readers[p]; ok {
						return r, nil
					}
					r, err := f.mem.Open(p)
					if err == nil {
						readers[p] = r
					}
					return r, err
				}
			},
			catch: "a second reader, read while the first was open",
		},
		{
			name: "Open's reader fails to close",
			build: func(f *fake) {
				f.open = func(p string) (io.ReadCloser, error) {
					r, err := f.mem.Open(p)
					if err != nil {
						return nil, err
					}
					return failingCloser{r}, nil
				}
			},
			catch: "Close: close failed",
		},
		{
			name: "lookup folds case",
			build: func(f *fake) {
				byFold := func(p string) string {
					for _, e := range f.mem.List() {
						if strings.EqualFold(e, p) {
							return e
						}
					}
					return p
				}
				f.stat = func(p string) (pkgtree.Info, error) {
					info, err := f.mem.Stat(byFold(p))
					info.Path = p
					return info, err
				}
				f.open = func(p string) (io.ReadCloser, error) { return f.mem.Open(byFold(p)) }
			},
			catch: `Stat("METAIS.YAML"), not an entry, succeeded`,
		},
		{
			name: "lookup cleans the path",
			build: func(f *fake) {
				clean := func(p string) string { return strings.TrimPrefix(strings.TrimSuffix(p, "/"), "/") }
				f.open = func(p string) (io.ReadCloser, error) { return f.mem.Open(clean(p)) }
			},
			catch: `Open("/metais.yaml"), not an entry, returned a reader`,
		},
		{
			name: "a directory prefix opens",
			build: func(f *fake) {
				f.open = func(p string) (io.ReadCloser, error) {
					if p == "inputs" {
						return io.NopCloser(strings.NewReader("")), nil
					}
					return f.mem.Open(p)
				}
			},
			catch: `Open("inputs"), not an entry, returned a reader`,
		},
		{
			name: "refusal does not wrap ErrNotExist",
			build: func(f *fake) {
				f.stat = func(p string) (pkgtree.Info, error) {
					info, err := f.mem.Stat(p)
					if err != nil {
						return info, &pkgtree.PathError{Op: "stat", Path: p, Err: errors.New("gone")}
					}
					return info, nil
				}
			},
			catch: "which does not wrap pkgtree.ErrNotExist",
		},
		{
			name: "refusal is not a PathError",
			build: func(f *fake) {
				f.open = func(p string) (io.ReadCloser, error) {
					r, err := f.mem.Open(p)
					if err != nil {
						return nil, pkgtree.ErrNotExist
					}
					return r, nil
				}
			},
			catch: "which is not a *pkgtree.PathError",
		},
		{
			name: "refusal names the wrong method",
			build: func(f *fake) {
				f.stat = func(p string) (pkgtree.Info, error) {
					info, err := f.mem.Stat(p)
					if err != nil {
						return info, notExist("open", p)
					}
					return info, nil
				}
			},
			catch: `returned a *pkgtree.PathError for "open" of`,
		},
		{
			name: "refusal carries an Info",
			build: func(f *fake) {
				f.stat = func(p string) (pkgtree.Info, error) {
					info, err := f.mem.Stat(p)
					if err != nil {
						return pkgtree.Info{Path: p}, err
					}
					return info, nil
				}
			},
			catch: "returned an Info beside its error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := controlFiles()
			mem, err := New(files)
			if err != nil {
				t.Fatal(err)
			}
			f := &fake{mem: mem}
			if tt.build != nil {
				tt.build(f)
			}
			if tt.want != nil {
				tt.want(files)
			}
			err = Check(f, files)
			if err == nil {
				t.Fatalf("Check passed a tree whose %s", tt.name)
			}
			if !strings.Contains(err.Error(), tt.catch) {
				t.Errorf("Check caught it as:\n%s\nwant a problem saying %q", err, tt.catch)
			}
		})
	}
}

// TestCheckPassesFake checks the controls' fake answers as the in-memory
// tree does when nothing is broken, so each control's red is the breakage
// alone.
func TestCheckPassesFake(t *testing.T) {
	files := controlFiles()
	mem, err := New(files)
	if err != nil {
		t.Fatal(err)
	}
	if err := Check(&fake{mem: mem}, files); err != nil {
		t.Error(err)
	}
}
