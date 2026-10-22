// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"metais.org/metais/internal/testenv"
)

// modulePath is the module's import path.
const modulePath = "metais.org/metais"

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

// hostEnviron is the environment this test binary started with, taken
// before any test points HOME at a temporary directory: go env reads the
// host's caches from it.
var hostEnviron = os.Environ()

// goHost is the host's toolchain and its caches (testenv.GoHost), read
// once by the go command of the toolchain running this test. A test binary
// built with -trimpath does not know its GOROOT; then the go on PATH asks.
var goHost = sync.OnceValues(func() (testenv.GoHost, error) {
	exe := "go"
	if root := runtime.GOROOT(); root != "" {
		exe = filepath.Join(root, "bin", "go")
	}
	cmd := exec.Command(exe, testenv.GoHostQuery...)
	cmd.Env = append(slices.Clone(hostEnviron), "GOTOOLCHAIN=local")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return testenv.GoHost{}, fmt.Errorf("go %s: %v\n%s", strings.Join(testenv.GoHostQuery, " "), err, stderr.Bytes())
	}
	return testenv.ParseGoHost(hostEnviron, out)
})

// goCommand is the go command with args, in the whole environment
// testenv.GoEnv gives over a fresh directory: it takes nothing from the
// host but its toolchain and caches, and never reaches the network.
func goCommand(t testing.TB, args ...string) *exec.Cmd {
	t.Helper()
	h, err := goHost()
	if err != nil {
		t.Fatalf("locating the host's toolchain: %v", err)
	}
	cmd := exec.Command(h.Go(), args...)
	cmd.Env = goEnv(t)
	return cmd
}

// goEnv is testenv.GoEnv's environment over a fresh directory, for a go
// command a test runs directly or through a script.
func goEnv(t testing.TB) []string {
	t.Helper()
	h, err := goHost()
	if err != nil {
		t.Fatalf("locating the host's toolchain: %v", err)
	}
	return testenv.GoEnv(h, testenv.Dir(t))
}

// runGo runs the go command in dir under tg, cgo off, and returns its
// standard output.
func runGo(t testing.TB, dir string, tg target, args ...string) []byte {
	t.Helper()
	cmd := goCommand(t, args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Env, "GOOS="+tg.goos, "GOARCH="+tg.goarch)
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
