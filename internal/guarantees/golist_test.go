// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// modulePath is the module's import path.
const modulePath = "github.com/kottera/metais"

// target is one GOOS and GOARCH the tripwires read the tree under.
type target struct{ goos, goarch string }

func (t target) String() string { return t.goos + "/" + t.goarch }

// targets are the six of the gate (ADR-010 D12 item 2, D15 item 3). ADR-009
// D16 item 3 asks for darwin, linux and windows; both architectures of each
// are read because a dependency's files differ by architecture, and the
// dependency scan's allow-list is exact only over every file the gate builds.
var targets = []target{
	{"darwin", "amd64"}, {"darwin", "arm64"},
	{"linux", "amd64"}, {"linux", "arm64"},
	{"windows", "amd64"}, {"windows", "arm64"},
}

// listedPackage is the part of go list -json's output the tripwires read.
type listedPackage struct {
	ImportPath     string
	Name           string
	Dir            string
	Standard       bool
	Module         *listedModule
	GoFiles        []string
	CgoFiles       []string
	IgnoredGoFiles []string
	TestGoFiles    []string
	XTestGoFiles   []string
	Imports        []string
	TestImports    []string
	XTestImports   []string
	Deps           []string
	Error          *struct{ Err string }
	DepsErrors     []*struct{ Err string }
}

type listedModule struct {
	Path    string
	Version string
	Main    bool
	Dir     string
}

// inModule reports whether p belongs to the module path names.
func (p *listedPackage) inModule(path string) bool {
	return p.Module != nil && p.Module.Path == path
}

// releaseFiles is every file of p a release build compiles.
func (p *listedPackage) releaseFiles() []string {
	return append(append([]string(nil), p.GoFiles...), p.CgoFiles...)
}

var (
	modCacheOnce sync.Once
	modCache     string
	modCacheErr  error
)

// goTool is the go command of the toolchain running this test, so the
// tripwires read the tree with the toolchain that builds it. A test binary
// built with -trimpath does not know its GOROOT; then the go on PATH is used.
func goTool(t testing.TB) string {
	t.Helper()
	exe := "go"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	if root := runtime.GOROOT(); root != "" {
		p := filepath.Join(root, "bin", exe)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	p, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("no go command: %v", err)
	}
	return p
}

// moduleCache is where the toolchain keeps downloaded modules. It is the one
// fact read from the developer's go configuration, by go env GOMODCACHE,
// because without it the isolated go list below would need the network.
func moduleCache(t testing.TB) string {
	t.Helper()
	modCacheOnce.Do(func() {
		out, err := exec.Command(goTool(t), "env", "GOMODCACHE").Output()
		modCache, modCacheErr = strings.TrimSpace(string(out)), err
		if err == nil && modCache == "" {
			modCacheErr = errors.New("go env GOMODCACHE is empty")
		}
	})
	if modCacheErr != nil {
		t.Fatalf("locating the module cache: %v", modCacheErr)
	}
	return modCache
}

// isolatedGoEnv is the whole environment of a go command the tripwires run.
// Nothing is inherited: HOME, the temporary directory, GOPATH and the
// build cache are fresh directories from internal/testenv, GOENV=off ignores the developer's go
// configuration file, GOPROXY=off, GOSUMDB=off and GOTOOLCHAIN=local keep
// the command off the network, and the module cache is the one the
// toolchain already holds, read only (-mod=readonly).
func isolatedGoEnv(t testing.TB, tg target) []string {
	t.Helper()
	home, tmp := testenv.Dir(t), testenv.Dir(t)
	return []string{
		"HOME=" + home,
		"USERPROFILE=" + home,
		"TMPDIR=" + tmp,
		"TMP=" + tmp,
		"TEMP=" + tmp,
		"GOTMPDIR=" + tmp,
		"GOENV=off",
		"GOFLAGS=-mod=readonly",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOTOOLCHAIN=local",
		"GOWORK=off",
		"GOMODCACHE=" + moduleCache(t),
		"GOCACHE=" + testenv.Dir(t),
		"GOPATH=" + testenv.Dir(t),
		"CGO_ENABLED=0",
		"GOOS=" + tg.goos,
		"GOARCH=" + tg.goarch,
	}
}

// runGo runs the go command in dir under tg with the isolated environment
// and returns its standard output.
func runGo(t testing.TB, dir string, tg target, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(goTool(t), args...)
	cmd.Dir = dir
	cmd.Env = isolatedGoEnv(t, tg)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go %s in %s for %s: %v\n%s", strings.Join(args, " "), dir, tg, err, stderr.Bytes())
	}
	return out
}

// goList runs go list -e -deps -json over patterns in dir under tg and
// returns every package, failing t on any package that could not be loaded.
func goList(t testing.TB, dir string, tg target, patterns ...string) []*listedPackage {
	t.Helper()
	out := runGo(t, dir, tg, append([]string{"list", "-e", "-deps", "-json"}, patterns...)...)
	var pkgs []*listedPackage
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		p := new(listedPackage)
		err := dec.Decode(p)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decoding go list output: %v", err)
		}
		if p.Error != nil {
			t.Fatalf("go list %s for %s: %s: %s", strings.Join(patterns, " "), tg, p.ImportPath, p.Error.Err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs
}

type listKey struct {
	dir     string
	tg      target
	pattern string
}

var (
	listMu    sync.Mutex
	listCache = map[listKey][]*listedPackage{}
)

// cachedList is goList over one pattern, run once per test binary for each
// directory, target and pattern; the tree does not change while tests run.
func cachedList(t testing.TB, dir string, tg target, pattern string) []*listedPackage {
	t.Helper()
	k := listKey{dir, tg, pattern}
	listMu.Lock()
	defer listMu.Unlock()
	if pkgs, ok := listCache[k]; ok {
		return pkgs
	}
	pkgs := goList(t, dir, tg, pattern)
	listCache[k] = pkgs
	return pkgs
}

// sourceFile is one parsed Go file of a listed package.
type sourceFile struct {
	pkg  string // the package's import path
	name string // the file's base name
	fset *token.FileSet
	file *ast.File
}

// parseFile parses name in dir, comments included.
func parseFile(pkg, dir, name string) (*sourceFile, error) {
	src, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}
	return parseSource(pkg, name, src)
}

// parseSource parses src as the file name of package pkg.
func parseSource(pkg, name string, src []byte) (*sourceFile, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	return &sourceFile{pkg: pkg, name: name, fset: fset, file: f}, nil
}

// at is the file-relative position of n, for messages.
func (f *sourceFile) at(n ast.Node) string {
	p := f.fset.Position(n.Pos())
	return fmt.Sprintf("%s/%s:%d", f.pkg, f.name, p.Line)
}

// importName is the name an import spec binds in its file: its explicit
// name, or the path's last element, skipping a major-version element
// (math/rand/v2 binds rand).
func importName(spec *ast.ImportSpec) (path, name string) {
	path, _ = strconv.Unquote(spec.Path.Value)
	if spec.Name != nil {
		return path, spec.Name.Name
	}
	elems := strings.Split(path, "/")
	name = elems[len(elems)-1]
	if len(elems) > 1 && len(name) > 1 && name[0] == 'v' && strings.Trim(name[1:], "0123456789") == "" {
		name = elems[len(elems)-2]
	}
	return path, name
}

// reference is one selector pkg.Name in a file, where pkg is a name the
// file's imports bind, together with the top-level function it sits in.
type reference struct {
	path string        // the imported package's path
	name string        // the selected name
	fn   string        // the enclosing top-level function, "" outside any
	call *ast.CallExpr // the call it is the function of, if any
	sel  *ast.SelectorExpr
}

// symbol is the reference as the tripwire lists name it: path.Name.
func (r reference) symbol() string { return r.path + "." + r.name }

// references returns every selector in f whose left side is a name one of
// f's imports binds. It is syntactic, as ADR-009 PR13 rules: a local
// variable that shadows an import name is read as the import, which can
// only flag more.
func (f *sourceFile) references() []reference {
	names := map[string]string{}
	for _, spec := range f.file.Imports {
		path, name := importName(spec)
		if name != "_" && name != "." {
			names[name] = path
		}
	}
	var refs []reference
	for _, decl := range f.file.Decls {
		fn := ""
		if d, ok := decl.(*ast.FuncDecl); ok {
			fn = d.Name.Name
			if d.Recv != nil && len(d.Recv.List) > 0 {
				fn = recvName(d.Recv.List[0].Type) + "." + fn
			}
		}
		calls := map[*ast.SelectorExpr]*ast.CallExpr{}
		ast.Inspect(decl, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if s, ok := ast.Unparen(n.Fun).(*ast.SelectorExpr); ok {
					calls[s] = n
				}
			case *ast.SelectorExpr:
				if id, ok := n.X.(*ast.Ident); ok {
					if path, ok := names[id.Name]; ok {
						refs = append(refs, reference{path: path, name: n.Sel.Name, fn: fn, call: calls[n], sel: n})
					}
				}
			}
			return true
		})
	}
	return refs
}

// recvName is the type name of a method receiver.
func recvName(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.StarExpr:
		return recvName(e.X)
	case *ast.IndexExpr:
		return recvName(e.X)
	case *ast.IndexListExpr:
		return recvName(e.X)
	case *ast.Ident:
		return e.Name
	}
	return "?"
}

// moduleRootAbs is the module root as an absolute path, for go commands.
func moduleRootAbs(t testing.TB) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, moduleRoot))
}

// moduleFiles is every release file of every package of ./... in the
// module at root, over every target, parsed once per file. A file a target
// excludes is read under the targets that include it.
func moduleFiles(t testing.TB, root string) []*sourceFile {
	t.Helper()
	seen := map[string]bool{}
	var files []*sourceFile
	for _, tg := range targets {
		for _, p := range cachedList(t, root, tg, "./...") {
			if p.Standard || p.Module == nil || !p.Module.Main {
				continue
			}
			for _, name := range p.releaseFiles() {
				key := filepath.Join(p.Dir, name)
				if seen[key] {
					continue
				}
				seen[key] = true
				f, err := parseFile(p.ImportPath, p.Dir, name)
				if err != nil {
					t.Fatal(err)
				}
				files = append(files, f)
			}
		}
	}
	return files
}
