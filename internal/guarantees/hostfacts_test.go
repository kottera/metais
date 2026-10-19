// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// TestNoHostFacts is the static tripwire of ADR-009 D16 item 3 and ADR-010
// D11 items 9 and 10. It reads the module and the binary's closure as go
// list reports them with release tags, for every target, and applies five
// rules:
//
//   - the import rule: the closure of ./cmd/metais holds no package of
//     forbiddenImports;
//   - the symbol rule: no non-test file of the module refers to a symbol of
//     hostSymbols, or imports a package of hostPackages, except where
//     moduleAllowList names it; reflect is imported only by internal/schema;
//     and only internal/hostfs and internal/hostenv vary by platform;
//   - the directive rule: no Go file of the module carries //go:linkname,
//     imports "C", dot-imports a package or imports unsafe;
//   - the dependency scan: every hit of the same lists in a non-standard
//     package of the closure, or of the closure of any package of the
//     module, matches an entry of dependencyAllowList, and every entry
//     matches a hit;
//   - the canary rule: only canary_<channel>.go files carry a
//     metais_canary_<channel> tag, and no release build compiles one.
//
// Its controls run with it: the os.Getenv fixture under testdata/hostfacts
// and the netleak main package must each be flagged.
func TestNoHostFacts(t *testing.T) {
	root := moduleRootAbs(t)

	t.Run("imports", func(t *testing.T) {
		for _, tg := range targets {
			for _, v := range importViolations(cachedList(t, root, tg, "./cmd/metais")) {
				t.Errorf("%s: %s", tg, v)
			}
		}
	})
	t.Run("symbols", func(t *testing.T) {
		for _, v := range symbolViolations(moduleFiles(t, root), modulePath) {
			t.Error(v)
		}
	})
	t.Run("directives", func(t *testing.T) {
		for _, v := range directiveViolations(walkModuleFiles(t, root, modulePath)) {
			t.Error(v)
		}
	})
	t.Run("canaries", func(t *testing.T) {
		for _, v := range canaryViolations(walkModuleFiles(t, root, modulePath), releasedFiles(t, root)) {
			t.Error(v)
		}
	})
	t.Run("dependencies", func(t *testing.T) {
		hits := dependencyHits(t, root)
		for _, v := range dependencyViolations(hits, dependencyAllowList) {
			t.Error(v)
		}
	})

	// The controls: a tripwire that has gone blind fails here.
	t.Run("control/getenv", func(t *testing.T) {
		fixture := filepath.Join(root, "internal", "guarantees", "testdata", "hostfacts")
		got := symbolViolations(moduleFiles(t, fixture), modulePath)
		if !slices.ContainsFunc(got, func(v string) bool {
			return strings.Contains(v, "internal/leak/getenv.go") && strings.Contains(v, "os.Getenv")
		}) {
			t.Errorf("the symbol rule did not flag testdata/hostfacts/internal/leak/getenv.go's os.Getenv; it reported %q", got)
		}
	})
	t.Run("control/netleak", func(t *testing.T) {
		for _, tg := range targets {
			got := importViolations(cachedList(t, root, tg, "./internal/guarantees/testdata/netleak"))
			if !slices.ContainsFunc(got, func(v string) bool { return strings.HasPrefix(v, `imports "net" `) }) {
				t.Errorf("%s: the import rule did not flag testdata/netleak's import of net; it reported %q", tg, got)
			}
		}
	})
	t.Run("control/dependencies", func(t *testing.T) {
		// The closure holds golang.org/x/sys, whose environment wrappers
		// are hits; a scan that finds none has gone blind.
		if len(dependencyHits(t, root)) == 0 {
			t.Error("the dependency scan found no hit in the closure of ./cmd/metais")
		}
	})
}

// forbiddenImports is ADR-010 D11 item 10's list, which ADR-009 D16 item 3's
// import rule states: no package of it is in the binary's closure.
// net/url and net/netip are parsers that do not import net, and are allowed.
var forbiddenImports = []string{
	"crypto/tls", "log/syslog", "net", "net/http", "net/rpc", "net/smtp",
	"os/exec", "os/signal", "os/user", "plugin", "runtime/cgo", "syscall/js",
}

// forbiddenImportPrefixes extends the list below the packages ADR-010 D11
// item 10 names with "and below".
var forbiddenImportPrefixes = []string{"net/http/", "net/rpc/"}

// importViolations applies the import rule to a go list -deps closure.
func importViolations(pkgs []*listedPackage) []string {
	var bad []string
	for _, p := range pkgs {
		if slices.Contains(forbiddenImports, p.ImportPath) ||
			slices.ContainsFunc(forbiddenImportPrefixes, func(pre string) bool { return strings.HasPrefix(p.ImportPath, pre) }) {
			bad = append(bad, fmt.Sprintf("imports %q (ADR-010 D11 item 10), through %s", p.ImportPath, importersOf(pkgs, p.ImportPath)))
		}
	}
	return bad
}

// importersOf names the packages of pkgs that import path directly.
func importersOf(pkgs []*listedPackage, path string) string {
	var by []string
	for _, p := range pkgs {
		if slices.Contains(p.Imports, path) {
			by = append(by, p.ImportPath)
		}
	}
	return strings.Join(by, ", ")
}

// Host facts, as ADR-009 D12's table names its channels.
const (
	factEnv      = "environment variables"
	factFiles    = "files"
	factCwd      = "working directory"
	factHome     = "home, config and cache directories"
	factTemp     = "temporary directory"
	factUser     = "user and groups"
	factHostname = "host name"
	factProcess  = "process"
	factClock    = "clock"
	factZone     = "time zone"
	factRandom   = "random numbers"
	factExec     = "subprocesses"
	factPlatform = "platform"
	factStreams  = "process arguments, streams and exit"
)

// hostSymbols is the symbol rule's list: ADR-009 D12's tripwire symbols,
// ADR-010 D11 items 8 and 9's additions, and every os file function, each
// with the host fact it reads.
//
// Three groups go beyond the names the ADRs spell out, each closing a
// channel D12 already closes by another name: the environment's setters
// (os.Setenv, os.Unsetenv and their syscall kin) beside its readers;
// os.StartProcess, os.FindProcess and the syscall process starters, which
// run a subprocess without os/exec; and the file functions of io/ioutil and
// path/filepath, which reach the file system without naming an os file
// function. runtime.GOARCH joins runtime.GOOS: behaviour does not vary by
// architecture either.
var hostSymbols = map[string]string{
	"os.Getenv": factEnv, "os.LookupEnv": factEnv, "os.Environ": factEnv,
	"os.ExpandEnv": factEnv, "os.Expand": factEnv, "os.Clearenv": factEnv,
	"os.Setenv": factEnv, "os.Unsetenv": factEnv,
	"syscall.Getenv": factEnv, "syscall.Environ": factEnv, "syscall.Setenv": factEnv,
	"syscall.Unsetenv": factEnv, "syscall.Clearenv": factEnv,

	"os.Chmod": factFiles, "os.Chown": factFiles, "os.Chtimes": factFiles,
	"os.CopyFS": factFiles, "os.Create": factFiles, "os.DirFS": factFiles,
	"os.Lchown": factFiles, "os.Link": factFiles, "os.Lstat": factFiles,
	"os.Mkdir": factFiles, "os.MkdirAll": factFiles, "os.NewFile": factFiles,
	"os.Open": factFiles, "os.OpenFile": factFiles, "os.OpenInRoot": factFiles,
	"os.OpenRoot": factFiles, "os.Pipe": factFiles, "os.ReadDir": factFiles,
	"os.ReadFile": factFiles, "os.Readlink": factFiles, "os.Remove": factFiles,
	"os.RemoveAll": factFiles, "os.Rename": factFiles, "os.Stat": factFiles,
	"os.Symlink": factFiles, "os.Truncate": factFiles, "os.WriteFile": factFiles,
	"io/ioutil.ReadFile": factFiles, "io/ioutil.WriteFile": factFiles,
	"io/ioutil.ReadDir": factFiles, "path/filepath.Walk": factFiles,
	"path/filepath.WalkDir": factFiles, "path/filepath.Glob": factFiles,

	"os.Getwd": factCwd, "os.Chdir": factCwd,
	"path/filepath.Abs": factCwd, "path/filepath.EvalSymlinks": factCwd,

	"os.UserHomeDir": factHome, "os.UserConfigDir": factHome, "os.UserCacheDir": factHome,

	"os.TempDir": factTemp, "os.CreateTemp": factTemp, "os.MkdirTemp": factTemp,
	"io/ioutil.TempDir": factTemp, "io/ioutil.TempFile": factTemp,

	"os.Getuid": factUser, "os.Geteuid": factUser, "os.Getgid": factUser,
	"os.Getegid": factUser, "os.Getgroups": factUser,

	"os.Hostname": factHostname,

	"os.Getpid": factProcess, "os.Getppid": factProcess, "os.Executable": factProcess,

	"os.StartProcess": factExec, "os.FindProcess": factExec,
	"syscall.Exec": factExec, "syscall.ForkExec": factExec, "syscall.StartProcess": factExec,

	"time.Now": factClock, "time.Since": factClock, "time.Until": factClock,
	"time.NewTimer": factClock, "time.NewTicker": factClock, "time.After": factClock,
	"time.AfterFunc": factClock, "time.Tick": factClock,

	"time.Local": factZone, "time.LoadLocation": factZone, "time.Parse": factZone,
	"time.Unix": factZone, "time.UnixMilli": factZone, "time.UnixMicro": factZone,

	"runtime.GOOS": factPlatform, "runtime.GOARCH": factPlatform,

	"os.Args": factStreams, "os.Stdin": factStreams, "os.Stdout": factStreams,
	"os.Stderr": factStreams, "os.Exit": factStreams,
}

// hostPackages are packages the symbol rule refuses whole: to import one is
// to use it. ADR-009 D12 refuses the random sources "with no exception",
// so no allow-list entry, in the module or in a dependency, can name one.
var hostPackages = map[string]string{
	"crypto/rand":  factRandom,
	"math/rand":    factRandom,
	"math/rand/v2": factRandom,
	"os/user":      factUser,
}

// reflectPackage is internal/schema, whose adapter is the one place
// ADR-010 D11 item 9 and ADR-009 D16 item 3 let module code use reflect.
const reflectPackage = "internal/schema"

// platformPackages are the only module packages with files built for one
// platform: ADR-010 D11 item 9 makes build-tagged files the one sanctioned
// variation, and only there.
var platformPackages = []string{"internal/hostenv", "internal/hostfs"}

// fileFunctions are the symbols of factFiles, which internal/hostfs may use.
func fileFunctions() []string {
	var syms []string
	for s, fact := range hostSymbols {
		if fact == factFiles {
			syms = append(syms, s)
		}
	}
	slices.Sort(syms)
	return syms
}

// moduleAllowance lets one package, or one top-level function of it, refer
// to the symbols it names.
type moduleAllowance struct {
	pkg     string            // module-relative package path
	fn      string            // the one function, or "" for the whole package
	symbols []string          // what it may refer to
	onlyArg map[string]string // a symbol it may call only with this string literal
	reason  string
}

// moduleAllowList is RULINGS-v1 R2-143's allow-list, exactly: ADR-009 D16
// item 3's four entries, each with its reason. No entry is added to make
// the tripwire pass; a package that cannot pass without one is a finding.
// internal/hostfs and internal/dates are named before they exist, so the
// seams they fill are fixed before their code lands.
var moduleAllowList = []moduleAllowance{
	{
		pkg:     "internal/hostenv",
		fn:      "Isolate",
		symbols: []string{"os.Getenv", "os.Clearenv", "time.Local"},
		onlyArg: map[string]string{"os.Getenv": "NO_COLOR"},
		reason: "Isolate runs first in main: it reads NO_COLOR, the one display setting, " +
			"then empties the environment and sets time.Local to UTC (ADR-009 D14 item 1, ADR-010 D11 item 8)",
	},
	{
		pkg:     "internal/hostfs",
		symbols: fileFunctions(),
		reason: "the one package that touches the host's files, through os.Root, " +
			"with paths and errors rewritten (ADR-009 D14 item 3, D15; ADR-010 D11 item 3)",
	},
	{
		pkg:     "cmd/metais",
		symbols: []string{"os.Args", "os.Stdin", "os.Stdout", "os.Stderr", "os.Exit"},
		reason: "the shell alone touches the process; everything else receives the arguments " +
			"and streams, so the tool runs in-process under test (ADR-009 D14 item 4, ADR-010 D2)",
	},
	{
		pkg:     "internal/dates",
		symbols: []string{"time.Parse", "time.Unix", "time.UnixMilli", "time.UnixMicro"},
		reason: "parses --now with time.Parse into a fixed offset and does the date filter's " +
			"arithmetic with time.Unix and its kin, never with time.Local (ADR-004 D11, ADR-010 D11 item 5)",
	},
}

// allowed reports whether ref, in package pkg (module-relative), is one an
// allow-list entry lets through.
func allowed(pkg string, ref reference) bool {
	for _, a := range moduleAllowList {
		if a.pkg != pkg || (a.fn != "" && a.fn != ref.fn) || !slices.Contains(a.symbols, ref.symbol()) {
			continue
		}
		if want, ok := a.onlyArg[ref.symbol()]; ok {
			if ref.call == nil || len(ref.call.Args) != 1 {
				return false
			}
			lit, ok := ref.call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return false
			}
			if s, err := strconv.Unquote(lit.Value); err != nil || s != want {
				return false
			}
		}
		return true
	}
	return false
}

// relPackage is a module package's path relative to module, or "" when it
// is not in module.
func relPackage(module, pkg string) string {
	if pkg == module {
		return "."
	}
	if rest, ok := strings.CutPrefix(pkg, module+"/"); ok {
		return rest
	}
	return ""
}

// symbolViolations applies the symbol rule to the non-test files of the
// module module.
func symbolViolations(files []*sourceFile, module string) []string {
	var bad []string
	for _, f := range files {
		pkg := relPackage(module, f.pkg)
		for _, spec := range f.file.Imports {
			path, _ := importName(spec)
			if fact, ok := hostPackages[path]; ok {
				bad = append(bad, fmt.Sprintf("%s: imports %s (%s)", f.at(spec), path, fact))
			}
			if path == "reflect" && pkg != reflectPackage {
				bad = append(bad, fmt.Sprintf("%s: imports reflect outside %s (ADR-010 D11 item 9)", f.at(spec), reflectPackage))
			}
		}
		for _, ref := range f.references() {
			fact, ok := hostSymbols[ref.symbol()]
			if !ok || allowed(pkg, ref) {
				continue
			}
			where := "outside the allow-list"
			if ref.fn != "" {
				where = "in " + ref.fn + ", outside the allow-list"
			}
			bad = append(bad, fmt.Sprintf("%s: refers to %s (%s) %s", f.at(ref.sel), ref.symbol(), fact, where))
		}
		if platformFile(f) && !slices.Contains(platformPackages, pkg) {
			bad = append(bad, fmt.Sprintf("%s/%s: is built for one platform outside %s (ADR-010 D11 item 9)",
				f.pkg, f.name, strings.Join(platformPackages, " and ")))
		}
	}
	return bad
}

// Platform names, as go/build reads them in file names and build lines.
var (
	knownOS = []string{
		"aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js",
		"linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows", "zos",
	}
	knownArch = []string{
		"386", "amd64", "arm", "arm64", "loong64", "mips", "mipsle", "mips64", "mips64le",
		"ppc64", "ppc64le", "riscv64", "s390x", "wasm",
	}
)

// platformFile reports whether f is built for some platforms only: its name
// ends in _GOOS, _GOARCH or both, or its build line names a GOOS, a GOARCH
// or unix.
func platformFile(f *sourceFile) bool {
	stem := strings.TrimSuffix(strings.TrimSuffix(f.name, ".go"), "_test")
	parts := strings.Split(stem, "_")
	if n := len(parts); n > 1 {
		last := parts[n-1]
		if slices.Contains(knownOS, last) || slices.Contains(knownArch, last) {
			return true
		}
	}
	expr := buildConstraint(f.file)
	if expr == nil {
		return false
	}
	platform := false
	expr.Eval(func(tag string) bool {
		if tag == "unix" || slices.Contains(knownOS, tag) || slices.Contains(knownArch, tag) {
			platform = true
		}
		return false
	})
	return platform
}

// buildConstraint is f's //go:build expression, or nil.
func buildConstraint(f *ast.File) constraint.Expr {
	for _, g := range f.Comments {
		if g.Pos() >= f.Package {
			break
		}
		for _, c := range g.List {
			if constraint.IsGoBuild(c.Text) {
				if expr, err := constraint.Parse(c.Text); err == nil {
					return expr
				}
			}
		}
	}
	return nil
}

// buildTags is every tag f's build line names.
func buildTags(f *ast.File) []string {
	expr := buildConstraint(f)
	if expr == nil {
		return nil
	}
	var tags []string
	expr.Eval(func(tag string) bool {
		tags = append(tags, tag)
		return false
	})
	return tags
}

// walkModuleFiles parses every Go file below root, test files included and
// whatever its build line, as module code: every directory but testdata,
// those whose names begin with a dot or an underscore, which the go command
// ignores too, and those holding another module.
func walkModuleFiles(t testing.TB, root, module string) []*sourceFile {
	t.Helper()
	var files []*sourceFile
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			name := d.Name()
			if name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		pkg := module
		if rel != "." {
			pkg += "/" + filepath.ToSlash(rel)
		}
		f, err := parseFile(pkg, filepath.Dir(path), d.Name())
		if err != nil {
			return err
		}
		files = append(files, f)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// directiveViolations applies the directive rule. No allow-list entry
// holds unsafe, so it is refused wherever it is imported.
func directiveViolations(files []*sourceFile) []string {
	var bad []string
	for _, f := range files {
		for _, g := range f.file.Comments {
			for _, c := range g.List {
				if strings.HasPrefix(c.Text, "//go:linkname") {
					bad = append(bad, fmt.Sprintf("%s: carries //go:linkname", f.at(c)))
				}
			}
		}
		for _, spec := range f.file.Imports {
			path, name := importName(spec)
			switch {
			case path == "C":
				bad = append(bad, fmt.Sprintf(`%s: imports "C"`, f.at(spec)))
			case path == "unsafe":
				bad = append(bad, fmt.Sprintf("%s: imports unsafe", f.at(spec)))
			}
			if name == "." {
				bad = append(bad, fmt.Sprintf("%s: dot-imports %s", f.at(spec), path))
			}
		}
	}
	return bad
}

// canaryFile matches the one name a canary may have, canary_<channel>.go.
var canaryFile = regexp.MustCompile(`^canary_([a-z]+)\.go$`)

const canaryTagPrefix = "metais_canary_"

// releasedFiles is every file a release build compiles, over every target,
// as module-relative slash paths: the files of ./... and of ./cmd/metais's
// closure in the module.
func releasedFiles(t testing.TB, root string) []string {
	t.Helper()
	seen := map[string]bool{}
	for _, tg := range targets {
		for _, pattern := range []string{"./...", "./cmd/metais"} {
			for _, p := range cachedList(t, root, tg, pattern) {
				if p.Module == nil || !p.Module.Main {
					continue
				}
				for _, name := range p.releaseFiles() {
					seen[relPackage(p.Module.Path, p.ImportPath)+"/"+name] = true
				}
			}
		}
	}
	return slices.Sorted(func(yield func(string) bool) {
		for k := range seen {
			if !yield(k) {
				return
			}
		}
	})
}

// canaryViolations applies the canary rule to every file of the module and
// the list of released files.
func canaryViolations(files []*sourceFile, released []string) []string {
	var bad []string
	for _, f := range files {
		var tags []string
		for _, tag := range buildTags(f.file) {
			if strings.HasPrefix(tag, canaryTagPrefix) {
				tags = append(tags, tag)
			}
		}
		m := canaryFile.FindStringSubmatch(f.name)
		switch {
		case m == nil && len(tags) > 0:
			bad = append(bad, fmt.Sprintf("%s/%s: carries %s but is not named canary_<channel>.go", f.pkg, f.name, strings.Join(tags, ", ")))
		case m != nil && !slices.Equal(tags, []string{canaryTagPrefix + m[1]}):
			bad = append(bad, fmt.Sprintf("%s/%s: must carry the one canary tag %s%s, and carries %q", f.pkg, f.name, canaryTagPrefix, m[1], tags))
		}
	}
	for _, path := range released {
		if canaryFile.MatchString(filepath.Base(path)) {
			bad = append(bad, fmt.Sprintf("%s: a canary is compiled into a release build", path))
		}
	}
	return bad
}

// dependencyHit is one symbol of the lists in one file of a dependency.
type dependencyHit struct {
	importPath, file, symbol string
}

// dependencyAllowance is one entry of the dependency scan's allow-list.
type dependencyAllowance struct {
	importPath, file, symbol, reason string
}

// dependencyHits runs the dependency scan over the closure of ./cmd/metais,
// and of every package of the module, for every target: each symbol of
// hostSymbols referred to, and each package of hostPackages imported, in a
// release file of a non-standard package outside the module. A library
// built ahead of its wiring is scanned with the modules it links before
// the binary links them.
func dependencyHits(t testing.TB, root string) []dependencyHit {
	t.Helper()
	seen := map[dependencyHit]bool{}
	parsed := map[string]bool{}
	for _, tg := range targets {
		for _, p := range slices.Concat(cachedList(t, root, tg, "./cmd/metais"), cachedList(t, root, tg, "./...")) {
			if p.Standard || p.Module == nil || p.Module.Main {
				continue
			}
			for _, name := range p.releaseFiles() {
				key := filepath.Join(p.Dir, name)
				if parsed[key] {
					continue
				}
				parsed[key] = true
				f, err := parseFile(p.ImportPath, p.Dir, name)
				if err != nil {
					t.Fatal(err)
				}
				for _, h := range fileHits(f) {
					seen[h] = true
				}
			}
		}
	}
	hits := make([]dependencyHit, 0, len(seen))
	for h := range seen {
		hits = append(hits, h)
	}
	slices.SortFunc(hits, func(a, b dependencyHit) int {
		return strings.Compare(a.importPath+" "+a.file+" "+a.symbol, b.importPath+" "+b.file+" "+b.symbol)
	})
	return hits
}

// fileHits is the dependency scan of one file.
func fileHits(f *sourceFile) []dependencyHit {
	var hits []dependencyHit
	hostPaths := map[string]bool{}
	for s := range hostSymbols {
		hostPaths[s[:strings.LastIndex(s, ".")]] = true
	}
	for _, spec := range f.file.Imports {
		path, name := importName(spec)
		if _, ok := hostPackages[path]; ok {
			hits = append(hits, dependencyHit{f.pkg, f.name, path})
		}
		if name == "." && hostPaths[path] {
			hits = append(hits, dependencyHit{f.pkg, f.name, path + ".*"})
		}
	}
	for _, ref := range f.references() {
		if _, ok := hostSymbols[ref.symbol()]; ok {
			hits = append(hits, dependencyHit{f.pkg, f.name, ref.symbol()})
		}
	}
	return hits
}

// dependencyViolations matches hits against the allow-list both ways: a
// hit no entry matches fails, and so does an entry no hit matches, so the
// list stays exact. An entry missing a field, naming a random source or
// repeated also fails.
func dependencyViolations(hits []dependencyHit, list []dependencyAllowance) []string {
	var bad []string
	entries := map[dependencyHit]bool{}
	for _, a := range list {
		key := dependencyHit{a.importPath, a.file, a.symbol}
		switch {
		case a.importPath == "" || a.file == "" || a.symbol == "" || strings.TrimSpace(a.reason) == "":
			bad = append(bad, fmt.Sprintf("allow-list entry %+v: must give the import path, file, symbol and reason", a))
			continue
		case hostPackages[a.symbol] == factRandom:
			bad = append(bad, fmt.Sprintf("allow-list entry %s %s %s: a random source is refused with no exception (ADR-009 D12)", a.importPath, a.file, a.symbol))
			continue
		case entries[key]:
			bad = append(bad, fmt.Sprintf("allow-list entry %s %s %s: listed twice", a.importPath, a.file, a.symbol))
			continue
		}
		entries[key] = true
	}
	found := map[dependencyHit]bool{}
	for _, h := range hits {
		found[h] = true
		if !entries[h] {
			bad = append(bad, fmt.Sprintf("%s/%s: refers to %s, which no allow-list entry names; add one with the reason it cannot reach a template, an output or a message, or stop using the dependency", h.importPath, h.file, h.symbol))
		}
	}
	for _, a := range list {
		if key := (dependencyHit{a.importPath, a.file, a.symbol}); entries[key] && !found[key] {
			bad = append(bad, fmt.Sprintf("allow-list entry %s %s %s: matches no hit; remove it", a.importPath, a.file, a.symbol))
		}
	}
	return bad
}

// dependencyAllowList is the dependency scan's allow-list (ADR-009 D16
// item 3): one entry per hit, each with the reason it cannot reach a
// template, an output or a message.
//
// The reasons rest on what METAIS calls: internal/hostenv is the one
// importer of golang.org/x/term and calls only term.IsTerminal (ADR-010 D2
// item 2, D11 item 8), and only on unix, where it calls
// unix.IoctlGetTermios; internal/hostfs is the one importer of
// golang.org/x/sys (ADR-010 D11 item 3), and calls only unix.Renameat2 on
// linux and unix.RenameatxNp on darwin, for the no-replace rename; neither
// module is in the windows closure (TestWindowsLinksNoXModules). A change
// to what the module calls in either re-reads these reasons.
var dependencyAllowList = []dependencyAllowance{
	{"golang.org/x/sys/unix", "env_unix.go", "syscall.Clearenv", reasonUnixEnv},
	{"golang.org/x/sys/unix", "env_unix.go", "syscall.Environ", reasonUnixEnv},
	{"golang.org/x/sys/unix", "env_unix.go", "syscall.Getenv", reasonUnixEnv},
	{"golang.org/x/sys/unix", "env_unix.go", "syscall.Setenv", reasonUnixEnv},
	{"golang.org/x/sys/unix", "env_unix.go", "syscall.Unsetenv", reasonUnixEnv},
	{"golang.org/x/sys/unix", "sockcmsg_unix_other.go", "runtime.GOARCH", reasonSocketLayout},
	{"golang.org/x/sys/unix", "sockcmsg_unix_other.go", "runtime.GOOS", reasonSocketLayout},
	{"golang.org/x/sys/unix", "syscall_bsd.go", "runtime.GOOS", reasonSocketLayout},
	{"golang.org/x/sys/unix", "syscall_unix.go", "syscall.Exec",
		"the body of unix.Exec, which replaces the process image; METAIS never calls it, " +
			"and term.IsTerminal does not"},
	{"golang.org/x/sys/unix", "sysvshm_linux.go", "runtime.GOARCH",
		"selects the System V shared-memory layout of the target architecture, a constant of the " +
			"build, inside unix.SysvShmAttach and its kin, which METAIS never calls"},
	{"golang.org/x/term", "terminal.go", "runtime.GOOS",
		"chooses the line ending inside readPasswordLine, reached only from term.ReadPassword and " +
			"Terminal.ReadPassword, which METAIS never calls"},
}

// Reasons shared by several entries of dependencyAllowList.
const (
	reasonUnixEnv = "the bodies of unix.Clearenv, unix.Environ, unix.Getenv, unix.Setenv and unix.Unsetenv, " +
		"which METAIS never calls, and term.IsTerminal does not; hostenv.Isolate has emptied the " +
		"environment before any code could"
	reasonSocketLayout = "selects the target's socket-address and control-message layout, a constant of the " +
		"build, inside the socket functions, which METAIS never calls: the import rule keeps net out " +
		"of the binary and nothing in it opens a socket"
)
