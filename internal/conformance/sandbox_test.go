// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
)

// The sentinel token and the names it carries, by source (ADR-011 D12).
// The token has only letters, so it survives every escaping a tool might
// apply; each source appends its own name.
const sentinel = "MTSSENTINEL"

// env names an environment of ADR-011 D12.
type env string

const (
	envA env = "A" // the sentinel environment
	envB env = "B" // the plain environment
)

// sandboxSeq numbers sandboxes across the run.
var sandboxSeq atomic.Int64

// sandbox is one case's fresh directory (ADR-011 D11 item 4): home, tmp,
// bin and work, with work the working directory of every step.
type sandbox struct {
	root string
	env  env
	vars map[string]string // the step environment, exactly
	home string
	tmp  string
	work string
}

// newSandbox makes a sandbox below base for e. Environment A's sandbox and
// its home, temporary and bin directories carry the sentinel in their
// paths, so an absolute path the tool prints is caught by the scan.
func newSandbox(base string, e env) (*sandbox, error) {
	n := sandboxSeq.Add(1)
	sb := &sandbox{env: e}
	if e == envA {
		sb.root = filepath.Join(base, fmt.Sprintf("%sPATH-%d", sentinel, n))
		sb.home = filepath.Join(sb.root, "home", sentinel+"HOME")
		sb.tmp = filepath.Join(sb.root, "tmp", sentinel+"TMP")
	} else {
		sb.root = filepath.Join(base, fmt.Sprintf("plain-%d", n))
		sb.home = filepath.Join(sb.root, "home")
		sb.tmp = filepath.Join(sb.root, "tmp")
	}
	sb.work = filepath.Join(sb.root, "work")
	bin := filepath.Join(sb.root, "bin")
	if e == envA {
		bin = filepath.Join(bin, sentinel+"BIN")
	}
	for _, d := range []string{sb.home, sb.tmp, bin, sb.work} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	if e == envB {
		sb.vars = map[string]string{
			"HOME": sb.home, "TMPDIR": sb.tmp, "PATH": bin,
			"TZ": "UTC", "LC_ALL": "C",
		}
		return sb, nil
	}
	sb.vars = map[string]string{
		"HOME":            sb.home,
		"TMPDIR":          sb.tmp,
		"PATH":            bin,
		"PWD":             sb.work,
		"USER":            sentinel + "USER",
		"LOGNAME":         sentinel + "USER",
		"HOSTNAME":        sentinel + "HOST",
		"XDG_CONFIG_HOME": filepath.Join(sb.home, ".config"),
		"XDG_DATA_HOME":   filepath.Join(sb.home, ".local", "share"),
		"XDG_CACHE_HOME":  filepath.Join(sb.home, ".cache"),
		"TZ":              "Pacific/Chatham",
		"LANG":            "tr_TR.UTF-8",
		"LC_ALL":          "tr_TR.UTF-8",
		"NO_COLOR":        sentinel + "NOCOLOR",
		"TERM":            sentinel + "TERM",
		"GODEBUG":         "zipinsecurepath=1,tarinsecurepath=1",
		"METAIS_SENTINEL": sentinel + "ENV",
	}
	for _, x := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"} {
		dir := filepath.Join(sb.vars[x], "metais")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, "SENTINEL.txt"), []byte(sentinel+"XDG\n"), 0o644); err != nil {
			return nil, err
		}
	}
	return sb, nil
}

// environ is the sandbox's environment as a process receives it, sorted.
func (sb *sandbox) environ() []string {
	var out []string
	for k, v := range sb.vars {
		out = append(out, k+"="+v)
	}
	slices.Sort(out)
	return out
}

// differingVars names the variables whose values differ between two
// environments, which an A/B difference reports (ADR-011 D12 item 4).
func differingVars(a, b map[string]string) []string {
	var names []string
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			names = append(names, k)
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			names = append(names, k)
		}
	}
	slices.Sort(names)
	return names
}

// plantDecoys writes environment A's decoys (ADR-011 D12 item 2):
// SENTINEL.txt in the work directory and the home directory, and for every
// file directly inside a data tree the case's steps name, a file of the
// same name in each, unless the work directory already holds one.
func (sb *sandbox) plantDecoys(v *vector, c *vcase) error {
	names := []string{"SENTINEL.txt"}
	for _, st := range c.Steps {
		if st.args.Data == "" {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(v.dir, filepath.FromSlash(st.args.Data)))
		if err != nil {
			continue // a data tree an earlier step writes has no decoys
		}
		for _, e := range entries {
			if e.Type().IsRegular() {
				names = append(names, e.Name())
			}
		}
	}
	for _, n := range names {
		for _, d := range []struct{ dir, token string }{{sb.work, sentinel + "CWD"}, {sb.home, sentinel + "HOME"}} {
			p := filepath.Join(d.dir, n)
			if _, err := os.Lstat(p); err == nil {
				continue
			}
			if err := os.WriteFile(p, []byte(d.token+"\n"), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// applySetup prepares the work directory for case c of v (ADR-011 D9 item
// 4): the setup tree's contents, the empty directories, the links and the
// named pipes, then the package and every tree or file the steps'
// arguments name, each at the same relative path. expected/ is never
// copied.
func (sb *sandbox) applySetup(v *vector, c *vcase) error {
	if s := c.Setup; s != nil {
		if s.Copy != nil && *s.Copy != "" {
			if err := copyTree(filepath.Join(v.dir, filepath.FromSlash(*s.Copy)), sb.work); err != nil {
				return fmt.Errorf("setup copy: %w", err)
			}
		}
		for _, d := range s.Dirs {
			if err := os.MkdirAll(filepath.Join(sb.work, filepath.FromSlash(d)), 0o755); err != nil {
				return fmt.Errorf("setup dirs: %w", err)
			}
		}
		for _, l := range s.Symlinks {
			p := filepath.Join(sb.work, filepath.FromSlash(l.Path))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(filepath.FromSlash(l.Target), p); err != nil {
				return fmt.Errorf("setup symlinks: %w", err)
			}
		}
		for _, f := range s.Fifos {
			p := filepath.Join(sb.work, filepath.FromSlash(f))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			if err := mkfifo(p); err != nil {
				return fmt.Errorf("setup fifos: %w", err)
			}
		}
	}
	for _, rel := range inputPaths(c) {
		src := filepath.Join(v.dir, filepath.FromSlash(rel))
		fi, err := os.Lstat(src)
		if err != nil {
			continue // written by an earlier step
		}
		dst := filepath.Join(sb.work, filepath.FromSlash(rel))
		// The setup may have made part of a tree already — an empty
		// directory or a link inside the package — and the tree's files
		// are copied around it; anything else the setup put there stands.
		if got, err := os.Lstat(dst); err == nil && !(fi.IsDir() && got.IsDir()) {
			continue
		}
		if fi.IsDir() {
			err = copyTree(src, dst)
		} else {
			err = copyFile(src, dst)
		}
		if err != nil {
			return fmt.Errorf("copy %s: %w", rel, err)
		}
	}
	return nil
}

// inputPaths are the vector paths c's steps read: the package or source,
// the data tree and the inputs file.
func inputPaths(c *vcase) []string {
	var paths []string
	for _, st := range c.Steps {
		for _, p := range []string{st.args.Package, st.args.Source, st.args.Data, st.args.InputsJSON} {
			if p != "" && !slices.Contains(paths, p) {
				paths = append(paths, p)
			}
		}
	}
	return paths
}

// copyTree copies the regular files and directories below src into dst.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type().IsRegular():
			return copyFile(p, target)
		case d.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		return fmt.Errorf("%s: not a regular file, directory or link", rel)
	})
}

// copyFile copies one regular file, keeping its execute bits.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	fi, err := in.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644|fi.Mode().Perm()&0o111)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// node is one snapshot entry (ADR-011 D11 item 4): path, type, size,
// SHA-256, mode and link target.
type node struct {
	kind   string // file, dir, link or other
	size   int64
	digest string
	mode   fs.FileMode
	target string
}

// snapshot records home, tmp and work below the sandbox's root, keyed by
// slash paths relative to the root.
func (sb *sandbox) snapshot() (map[string]node, error) {
	snap := map[string]node{}
	for _, top := range []string{"home", "tmp", "work"} {
		dir := filepath.Join(sb.root, top)
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(sb.root, p)
			if err != nil {
				return err
			}
			fi, err := d.Info()
			if err != nil {
				return err
			}
			n := node{mode: fi.Mode()}
			switch {
			case d.IsDir():
				n.kind = "dir"
			case d.Type().IsRegular():
				n.kind, n.size = "file", fi.Size()
				if n.digest, err = fileDigest(p); err != nil {
					return err
				}
			case d.Type()&fs.ModeSymlink != 0:
				n.kind = "link"
				if n.target, err = os.Readlink(p); err != nil {
					return err
				}
			default:
				n.kind = "other"
			}
			snap[filepath.ToSlash(rel)] = n
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return snap, nil
}

// fileDigest is the hexadecimal SHA-256 of the file at p.
func fileDigest(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// changesOutside lists every path created, changed or removed between two
// snapshots that lies outside the declared trees and the excluded paths,
// all given relative to the work directory (ADR-011 D10 item 4).
func changesOutside(before, after map[string]node, trees, excluded []string) []string {
	inside := func(p string) bool {
		rel, ok := strings.CutPrefix(p, "work/")
		if !ok {
			return false
		}
		for _, t := range append(slices.Clone(trees), excluded...) {
			t = path.Clean(t)
			if t == "." || rel == t || strings.HasPrefix(rel, t+"/") {
				return true
			}
		}
		return false
	}
	var changed []string
	for p, a := range after {
		if inside(p) {
			continue
		}
		b, ok := before[p]
		switch {
		case !ok:
			changed = append(changed, p+" was created")
		case a != b:
			changed = append(changed, p+" was changed")
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok && !inside(p) {
			changed = append(changed, p+" was removed")
		}
	}
	slices.Sort(changed)
	return changed
}

// probe reports whether the file system at dir offers what a vector
// requires (ADR-011 D9 item 6), and why not when it does not.
func probe(dir, req string) (bool, string) {
	d, err := os.MkdirTemp(dir, "probe-")
	if err != nil {
		return false, err.Error()
	}
	defer os.RemoveAll(d)
	switch req {
	case "symlinks":
		if err := os.Symlink("target", filepath.Join(d, "link")); err != nil {
			return false, "cannot create a symbolic link: " + err.Error()
		}
		return true, ""
	case "fifos":
		if err := mkfifo(filepath.Join(d, "fifo")); err != nil {
			return false, "cannot create a named pipe: " + err.Error()
		}
		return true, ""
	case "case-sensitive-fs":
		if err := os.WriteFile(filepath.Join(d, "a"), nil, 0o644); err != nil {
			return false, err.Error()
		}
		if _, err := os.Lstat(filepath.Join(d, "A")); err == nil {
			return false, "the file system folds case"
		} else if !errors.Is(err, fs.ErrNotExist) {
			return false, err.Error()
		}
		return true, ""
	}
	return false, "unknown requirement " + req
}
