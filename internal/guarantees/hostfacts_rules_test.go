// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"slices"
	"strings"
	"testing"
)

// The tests in this file are TestNoHostFacts's rules run over sources that
// break them, one clause at a time, so a clause gone blind fails the build.

// ruleCase is one source file and what a rule must say about it: nothing,
// when want is empty, or a violation containing each string of want.
type ruleCase struct {
	name string
	pkg  string // module-relative package path
	file string
	src  string
	want []string
}

// parseCase parses c's source as a file of the module.
func parseCase(t *testing.T, c ruleCase) *sourceFile {
	t.Helper()
	pkg := modulePath
	if c.pkg != "." {
		pkg += "/" + c.pkg
	}
	file := c.file
	if file == "" {
		file = "x.go"
	}
	f, err := parseSource(pkg, file, []byte(c.src))
	if err != nil {
		t.Fatalf("%s: %v", c.name, err)
	}
	return f
}

// checkRule runs rule over each case's file alone.
func checkRule(t *testing.T, cases []ruleCase, rule func(*sourceFile) []string) {
	t.Helper()
	for _, c := range cases {
		got := rule(parseCase(t, c))
		if len(c.want) == 0 {
			if len(got) > 0 {
				t.Errorf("%s: flagged %q, want nothing", c.name, got)
			}
			continue
		}
		if !slices.ContainsFunc(got, func(v string) bool {
			for _, w := range c.want {
				if !strings.Contains(v, w) {
					return false
				}
			}
			return true
		}) {
			t.Errorf("%s: flagged %q, want a violation containing %q", c.name, got, c.want)
		}
	}
}

func TestSymbolRule(t *testing.T) {
	cases := []ruleCase{
		{"Isolate reads NO_COLOR", "internal/hostenv", "", `package hostenv
import ("os"; "time")
func Isolate() bool { v := os.Getenv("NO_COLOR") != ""; os.Clearenv(); time.Local = time.UTC; return v }`, nil},
		{"Isolate reads another variable", "internal/hostenv", "", `package hostenv
import "os"
func Isolate() string { return os.Getenv("HOME") }`, []string{"os.Getenv", "in Isolate"}},
		{"Isolate reads NO_COLOR by a name", "internal/hostenv", "", `package hostenv
import "os"
const v = "NO_COLOR"
func Isolate() string { return os.Getenv(v) }`, []string{"os.Getenv"}},
		{"Isolate looks a variable up", "internal/hostenv", "", `package hostenv
import "os"
func Isolate() bool { _, ok := os.LookupEnv("NO_COLOR"); return ok }`, []string{"os.LookupEnv"}},
		{"hostenv clears the environment outside Isolate", "internal/hostenv", "", `package hostenv
import "os"
func IsTerminal() { os.Clearenv() }`, []string{"os.Clearenv", "in IsTerminal"}},
		{"a method named Isolate is not Isolate", "internal/hostenv", "", `package hostenv
import "os"
type T struct{}
func (*T) Isolate() string { return os.Getenv("NO_COLOR") }`, []string{"os.Getenv", "in T.Isolate"}},
		{"Isolate's symbols elsewhere", "internal/cli", "", `package cli
import "os"
func run() { os.Clearenv() }`, []string{"os.Clearenv"}},
		{"an aliased import", "internal/render", "", `package render
import o "os"
func run() string { return o.Getenv("HOME") }`, []string{"os.Getenv"}},
		{"a package-level initializer", "internal/render", "", `package render
import "os"
var home, _ = os.UserHomeDir()`, []string{"os.UserHomeDir", "outside the allow-list"}},
		{"a function value", "internal/render", "", `package render
import "os"
var env = os.Environ`, []string{"os.Environ"}},
		{"the shell's streams", "cmd/metais", "", `package main
import "os"
func main() { os.Stdout.Write(nil); os.Stderr.Write(nil); _ = os.Stdin; os.Exit(len(os.Args)) }`, nil},
		{"the streams outside the shell", "internal/cli", "", `package cli
import "os"
func run() { os.Stdout.Write(nil) }`, []string{"os.Stdout"}},
		{"the shell reads the environment", "cmd/metais", "", `package main
import "os"
func main() { os.Getenv("HOME") }`, []string{"os.Getenv"}},
		{"hostfs opens a root", "internal/hostfs", "", `package hostfs
import "os"
func open(p string) (*os.Root, error) { if _, err := os.Lstat(p); err != nil { return nil, err }; return os.OpenRoot(p) }`, nil},
		{"hostfs reads the environment", "internal/hostfs", "", `package hostfs
import "os"
func f() string { return os.Getenv("TMPDIR") }`, []string{"os.Getenv"}},
		{"hostfs stages under the temporary directory", "internal/hostfs", "", `package hostfs
import "os"
func f() (string, error) { return os.MkdirTemp("", "x") }`, []string{"os.MkdirTemp"}},
		{"hostfs makes a path absolute", "internal/hostfs", "", `package hostfs
import "path/filepath"
func f(p string) (string, error) { return filepath.Abs(p) }`, []string{"path/filepath.Abs"}},
		{"a file function outside hostfs", "internal/browse", "", `package browse
import "os"
func f() ([]byte, error) { return os.ReadFile("x") }`, []string{"os.ReadFile"}},
		{"a file walk outside hostfs", "internal/browse", "", `package browse
import "path/filepath"
func f() ([]string, error) { return filepath.Glob("*") }`, []string{"path/filepath.Glob"}},
		{"dates parses and converts", "internal/dates", "", `package dates
import "time"
func f(s string) (time.Time, error) { _ = time.Unix(0, 0); _ = time.UnixMilli(0); _ = time.UnixMicro(0); return time.Parse(time.RFC3339, s) }`, nil},
		{"dates uses the local zone", "internal/dates", "", `package dates
import "time"
func f(t time.Time) time.Time { return t.In(time.Local) }`, []string{"time.Local"}},
		{"dates reads the clock", "internal/dates", "", `package dates
import "time"
func f() time.Time { return time.Now() }`, []string{"time.Now"}},
		{"time.Parse outside dates", "internal/manifest", "", `package manifest
import "time"
func f(s string) (time.Time, error) { return time.Parse(time.RFC3339, s) }`, []string{"time.Parse"}},
		{"a timer", "internal/render", "", `package render
import "time"
func f() { <-time.After(0) }`, []string{"time.After", "clock"}},
		{"the host name", "internal/template", "", `package template
import "os"
func f() (string, error) { return os.Hostname() }`, []string{"os.Hostname"}},
		{"the process id", "internal/render", "", `package render
import "os"
func f() int { return os.Getpid() }`, []string{"os.Getpid"}},
		{"a subprocess without os/exec", "internal/render", "", `package render
import "os"
func f() { os.StartProcess("x", nil, nil) }`, []string{"os.StartProcess"}},
		{"the syscall environment", "internal/render", "", `package render
import "syscall"
func f() { syscall.Getenv("HOME") }`, []string{"syscall.Getenv"}},
		{"the platform", "internal/render", "", `package render
import "runtime"
var nl = map[bool]string{true: "\r\n"}[runtime.GOOS == "windows"]`, []string{"runtime.GOOS"}},
		{"the platform in hostfs", "internal/hostfs", "", `package hostfs
import "runtime"
var w = runtime.GOOS == "windows"`, []string{"runtime.GOOS"}},
		{"math/rand/v2", "internal/sample", "", `package sample
import "math/rand/v2"
func f() int { return rand.Int() }`, []string{"imports math/rand/v2", "random"}},
		{"a blank crypto/rand", "internal/sample", "", `package sample
import _ "crypto/rand"`, []string{"imports crypto/rand"}},
		{"math/rand in an allow-listed package", "internal/hostfs", "", `package hostfs
import "math/rand"
var n = rand.Int()`, []string{"imports math/rand"}},
		{"os/user", "internal/author", "", `package author
import "os/user"
func f() { user.Current() }`, []string{"imports os/user"}},
		{"reflect outside the schema adapter", "internal/finding", "", `package finding
import "reflect"
var _ = reflect.TypeOf(0)`, []string{"imports reflect"}},
		{"reflect in the schema adapter", "internal/schema", "", `package schema
import "reflect"
var _ = reflect.TypeOf(0)`, nil},
		{"a platform file by name", "internal/finding", "x_linux.go", `package finding`, []string{"built for one platform"}},
		{"a platform file by architecture", "internal/finding", "x_arm64.go", `package finding`, []string{"built for one platform"}},
		{"a platform file by build line", "internal/cli", "", "//go:build windows\n\npackage cli", []string{"built for one platform"}},
		{"a unix file", "internal/cli", "", "//go:build unix && !darwin\n\npackage cli", []string{"built for one platform"}},
		{"a platform file in hostenv", "internal/hostenv", "x_windows.go", `package hostenv`, nil},
		{"a platform file in hostfs", "internal/hostfs", "", "//go:build linux\n\npackage hostfs", nil},
		{"a non-platform tag", "internal/cli", "", "//go:build go1.26\n\npackage cli", nil},
		{"a name like a platform", "internal/cli", "linux.go", `package cli`, nil},
		{"the release builder runs, reads the environment and writes", "internal/tools/release", "", `package main
import ("os"; "os/exec")
func main() { _ = exec.Command("go"); _ = os.Environ(); _ = os.Getenv("GOOS"); _, _ = os.LookupEnv("GOOS"); _, _ = os.MkdirTemp("", "x"); _ = os.TempDir(); _ = os.WriteFile("x", nil, 0o644); _ = os.RemoveAll("x"); _, _ = os.Lstat("x"); os.Stdout.Write(nil); os.Stderr.Write(nil); _ = os.Stdin; os.Exit(len(os.Args)) }`, nil},
		{"a package beneath the release builder", "internal/tools/release/sub", "", `package sub
import "os"
func f() ([]byte, error) { return os.ReadFile(os.Getenv("X")) }`, nil},
		{"the release builder reads the clock", "internal/tools/release", "", `package main
import "time"
func f() time.Time { return time.Now() }`, []string{"time.Now"}},
		{"the release builder starts a timer", "internal/tools/release/sub", "", `package sub
import "time"
func f() { <-time.After(0) }`, []string{"time.After"}},
		{"the release builder reads the working directory", "internal/tools/release", "", `package main
import "os"
func f() (string, error) { return os.Getwd() }`, []string{"os.Getwd"}},
		{"the release builder makes a path absolute", "internal/tools/release", "", `package main
import "path/filepath"
func f() (string, error) { return filepath.Abs("x") }`, []string{"path/filepath.Abs"}},
		{"the release builder walks with path/filepath", "internal/tools/release", "", `package main
import "path/filepath"
func f() { filepath.WalkDir(".", nil) }`, []string{"path/filepath.WalkDir"}},
		{"the release builder creates a temporary file", "internal/tools/release", "", `package main
import "os"
func f() { os.CreateTemp("", "x") }`, []string{"os.CreateTemp"}},
		{"the release builder reads the host name", "internal/tools/release", "", `package main
import "os"
func f() (string, error) { return os.Hostname() }`, []string{"os.Hostname"}},
		{"the release builder reads the process", "internal/tools/release", "", `package main
import "os"
func f() int { return os.Getpid() + os.Getuid() }`, []string{"os.Getpid"}},
		{"the release builder reads the home directory", "internal/tools/release", "", `package main
import "os"
func f() (string, error) { return os.UserHomeDir() }`, []string{"os.UserHomeDir"}},
		{"the release builder imports os/user", "internal/tools/release", "", `package main
import "os/user"
func f() { user.Current() }`, []string{"imports os/user"}},
		{"the release builder imports a random source", "internal/tools/release", "", `package main
import "crypto/rand"
func f() { rand.Text() }`, []string{"imports crypto/rand"}},
		{"the release builder sets the environment", "internal/tools/release", "", `package main
import "os"
func f() { os.Setenv("X", "y") }`, []string{"os.Setenv"}},
		{"a sibling named like the release builder", "internal/tools/releasex", "", `package main
import "os"
func f() string { return os.Getenv("X") }`, []string{"os.Getenv"}},
		{"the release builder's parent", "internal/tools", "", `package tools
import "os"
func f() string { return os.Getenv("X") }`, []string{"os.Getenv"}},
		{"the UX runner starts sessions, reads the environment and the clock, and writes", "internal/tools/ux", "", `package main
import ("os"; "os/exec"; "time")
func main() { _ = exec.Command("claude"); _ = os.Environ(); _ = time.Now(); _, _ = os.ReadFile("x"); _ = os.WriteFile("x", nil, 0o644); _, _ = os.Create("x"); _ = os.Rename("x", "y"); _ = os.MkdirAll("x", 0o755); _ = os.RemoveAll("x"); _, _ = os.Stat("x"); _, _ = os.Lstat("x"); _ = os.DirFS("x"); os.Stdout.Write(nil); os.Stderr.Write(nil); os.Exit(len(os.Args)) }`, nil},
		{"a package beneath the UX runner", "internal/tools/ux/scenario", "", `package scenario
import "os"
func f() ([]byte, error) { return os.ReadFile("x") }`, []string{"os.ReadFile"}},
		{"the scorer beneath the UX runner reads the clock", "internal/tools/ux/score", "", `package score
import "time"
func f() time.Time { return time.Now() }`, []string{"time.Now"}},
		{"a sibling named like the UX runner", "internal/tools/uxx", "", `package main
import "time"
func f() time.Time { return time.Now() }`, []string{"time.Now"}},
		{"the UX runner's clock outside it", "internal/render", "", `package render
import "time"
func f() time.Time { return time.Now() }`, []string{"time.Now"}},
		{"the UX runner reads one variable", "internal/tools/ux", "", `package main
import "os"
func f() string { return os.Getenv("HOME") }`, []string{"os.Getenv"}},
		{"the UX runner looks a variable up", "internal/tools/ux", "", `package main
import "os"
func f() bool { _, ok := os.LookupEnv("HOME"); return ok }`, []string{"os.LookupEnv"}},
		{"the UX runner sets the environment", "internal/tools/ux", "", `package main
import "os"
func f() { os.Setenv("X", "y") }`, []string{"os.Setenv"}},
		{"the UX runner measures with time.Since", "internal/tools/ux", "", `package main
import "time"
func f(t time.Time) time.Duration { return time.Since(t) }`, []string{"time.Since"}},
		{"the UX runner starts a timer", "internal/tools/ux", "", `package main
import "time"
func f() { <-time.After(0) }`, []string{"time.After"}},
		{"the UX runner uses the local zone", "internal/tools/ux", "", `package main
import "time"
func f(t time.Time) time.Time { return t.In(time.Local) }`, []string{"time.Local"}},
		{"the UX runner reads the working directory", "internal/tools/ux", "", `package main
import "os"
func f() (string, error) { return os.Getwd() }`, []string{"os.Getwd"}},
		{"the UX runner makes a path absolute", "internal/tools/ux", "", `package main
import "path/filepath"
func f() (string, error) { return filepath.Abs("x") }`, []string{"path/filepath.Abs"}},
		{"the UX runner walks with path/filepath", "internal/tools/ux", "", `package main
import "path/filepath"
func f() { filepath.WalkDir(".", nil) }`, []string{"path/filepath.WalkDir"}},
		{"the UX runner opens a file", "internal/tools/ux", "", `package main
import "os"
func f() { os.Open("x"); os.Remove("x") }`, []string{"os.Open"}},
		{"the UX runner uses the temporary directory", "internal/tools/ux", "", `package main
import "os"
func f() (string, error) { _ = os.TempDir(); return os.MkdirTemp("", "x") }`, []string{"os.TempDir"}},
		{"the UX runner reads its standard input", "internal/tools/ux", "", `package main
import "os"
var in = os.Stdin`, []string{"os.Stdin"}},
		{"the UX runner reads the host name", "internal/tools/ux", "", `package main
import "os"
func f() (string, error) { return os.Hostname() }`, []string{"os.Hostname"}},
		{"the UX runner reads the process", "internal/tools/ux", "", `package main
import "os"
func f() (string, error) { _ = os.Getpid(); return os.Executable() }`, []string{"os.Getpid"}},
		{"the UX runner reads the home directory", "internal/tools/ux", "", `package main
import "os"
func f() (string, error) { return os.UserHomeDir() }`, []string{"os.UserHomeDir"}},
		{"the UX runner starts a process without os/exec", "internal/tools/ux", "", `package main
import "os"
func f() { os.StartProcess("x", nil, nil) }`, []string{"os.StartProcess"}},
		{"the UX runner reads the platform", "internal/tools/ux", "", `package main
import "runtime"
var w = runtime.GOOS == "windows"`, []string{"runtime.GOOS"}},
		{"the UX runner imports os/user", "internal/tools/ux", "", `package main
import "os/user"
func f() { user.Current() }`, []string{"imports os/user"}},
		{"the UX runner imports a random source", "internal/tools/ux", "", `package main
import "math/rand/v2"
func f() int { return rand.Int() }`, []string{"imports math/rand/v2"}},
		{"strings.Index is no host fact", "internal/cli", "", `package cli
import ("os"; "strings")
var _ = strings.Index("", ""); var _ os.FileMode`, nil},
	}
	checkRule(t, cases, func(f *sourceFile) []string {
		return symbolViolations([]*sourceFile{f}, modulePath)
	})
}

func TestDirectiveRule(t *testing.T) {
	cases := []ruleCase{
		{"clean", "internal/cli", "", `package cli
import "strings"
var _ = strings.Index`, nil},
		{"linkname", "internal/cli", "", `package cli
import _ "unsafe"
//go:linkname now runtime.nanotime
func now() int64`, []string{"//go:linkname"}},
		{"cgo", "internal/cli", "", `package cli
import "C"`, []string{`imports "C"`}},
		{"unsafe", "internal/hostfs", "", `package hostfs
import "unsafe"
var _ = unsafe.Sizeof(0)`, []string{"imports unsafe"}},
		{"a dot import", "internal/cli", "", `package cli
import . "strings"
var _ = Index`, []string{"dot-imports strings"}},
		{"a dot import in a test", "internal/cli", "x_test.go", `package cli
import . "os"
var _ = Getenv`, []string{"dot-imports os"}},
	}
	checkRule(t, cases, func(f *sourceFile) []string {
		return directiveViolations([]*sourceFile{f})
	})
}

func TestCanaryRule(t *testing.T) {
	cases := []ruleCase{
		{"a canary", "internal/template", "canary_env.go", "//go:build metais_canary_env\n\npackage template", nil},
		{"a release file", "internal/template", "env.go", "package template", nil},
		{"a canary tag on another file", "internal/template", "env.go", "//go:build metais_canary_env\n\npackage template",
			[]string{"env.go", "is not named canary_<channel>.go"}},
		{"a canary tag on a test file", "internal/template", "canary_env_test.go", "//go:build metais_canary_env\n\npackage template",
			[]string{"is not named canary_<channel>.go"}},
		{"a canary without its tag", "internal/template", "canary_env.go", "package template",
			[]string{"must carry the one canary tag metais_canary_env"}},
		{"a canary with another channel's tag", "internal/template", "canary_env.go", "//go:build metais_canary_tz\n\npackage template",
			[]string{"metais_canary_env"}},
		{"a canary with two tags", "internal/template", "canary_env.go", "//go:build metais_canary_env || metais_canary_tz\n\npackage template",
			[]string{"metais_canary_env"}},
	}
	checkRule(t, cases, func(f *sourceFile) []string {
		return canaryViolations([]*sourceFile{f}, nil)
	})

	// A canary whose build line still lets it into a release build is
	// caught by the released file list.
	got := canaryViolations(nil, []string{"internal/template/env.go", "internal/template/canary_env.go"})
	if len(got) != 1 || !strings.Contains(got[0], "internal/template/canary_env.go: a canary is compiled into a release build") {
		t.Errorf("released canary: flagged %q", got)
	}
}

func TestImportRule(t *testing.T) {
	pkgs := func(paths ...string) []*listedPackage {
		var ps []*listedPackage
		for _, p := range paths {
			ps = append(ps, &listedPackage{ImportPath: p, Standard: true})
		}
		ps = append(ps, &listedPackage{ImportPath: modulePath + "/cmd/metais", Imports: paths})
		return ps
	}
	if got := importViolations(pkgs("os", "net/url", "net/netip", "strings", "crypto/sha256")); len(got) > 0 {
		t.Errorf("a clean closure: flagged %q", got)
	}
	for _, bad := range append(slices.Clone(forbiddenImports), "net/http/httptrace", "net/rpc/jsonrpc") {
		got := importViolations(pkgs("os", bad))
		if len(got) != 1 || !strings.HasPrefix(got[0], `imports "`+bad+`" `) || !strings.HasSuffix(got[0], "cmd/metais") {
			t.Errorf("a closure holding %s: flagged %q", bad, got)
		}
	}
}

func TestDependencyScan(t *testing.T) {
	f, err := parseSource("example.com/dep", "env.go", []byte(`package dep
import ("math/rand"; o "os"; . "time"; "strings")
func f() { o.Getenv("X"); o.Getenv("Y"); o.Stdout.Write(nil); _ = strings.Index; _ = Second; rand.Int() }`))
	if err != nil {
		t.Fatal(err)
	}
	got := fileHits(f)
	slices.SortFunc(got, func(a, b dependencyHit) int { return strings.Compare(a.symbol, b.symbol) })
	got = slices.Compact(got)
	want := []dependencyHit{
		{"example.com/dep", "env.go", "math/rand"},
		{"example.com/dep", "env.go", "os.Getenv"},
		{"example.com/dep", "env.go", "os.Stdout"},
		{"example.com/dep", "env.go", "time.*"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("fileHits = %q, want %q", got, want)
	}

	hits := []dependencyHit{
		{"example.com/dep", "env.go", "os.Getenv"},
		{"example.com/dep", "clock.go", "time.Now"},
	}
	exact := []dependencyAllowance{
		{"example.com/dep", "env.go", "os.Getenv", "a debug path METAIS never calls"},
		{"example.com/dep", "clock.go", "time.Now", "a debug path METAIS never calls"},
	}
	if got := dependencyViolations(hits, exact); len(got) > 0 {
		t.Errorf("an exact list: flagged %q", got)
	}
	cases := []struct {
		name string
		hits []dependencyHit
		list []dependencyAllowance
		want string
	}{
		{"a hit with no entry", hits, exact[:1], "example.com/dep/clock.go: refers to time.Now, which no allow-list entry names"},
		{"an entry with no hit", hits[:1], exact, "allow-list entry example.com/dep clock.go time.Now: matches no hit"},
		{"an entry for another file", hits[:1], []dependencyAllowance{{"example.com/dep", "other.go", "os.Getenv", "r"}}, "matches no hit"},
		{"an entry with no reason", hits[:1], []dependencyAllowance{{"example.com/dep", "env.go", "os.Getenv", " "}}, "must give the import path, file, symbol and reason"},
		{"an entry with no file", hits[:1], []dependencyAllowance{{"example.com/dep", "", "os.Getenv", "r"}}, "must give the import path, file, symbol and reason"},
		{"an entry listed twice", hits[:1], []dependencyAllowance{exact[0], exact[0]}, "listed twice"},
		{"a random source", []dependencyHit{{"example.com/dep", "r.go", "crypto/rand"}},
			[]dependencyAllowance{{"example.com/dep", "r.go", "crypto/rand", "r"}}, "a random source is refused with no exception"},
	}
	for _, c := range cases {
		got := dependencyViolations(c.hits, c.list)
		if !slices.ContainsFunc(got, func(v string) bool { return strings.Contains(v, c.want) }) {
			t.Errorf("%s: flagged %q, want %q", c.name, got, c.want)
		}
	}
}

func TestModuleAllowListIsR2143(t *testing.T) {
	// RULINGS-v1 R2-143 and ADR-009 D16 item 3: these four, each with its
	// reason; ADR-009 Amendment A2 item 6: and the release builder's one
	// entry; Amendment A8 item 5: and the UX runner's one entry, for its
	// own package and none beneath it; and no other.
	var got []string
	for _, a := range moduleAllowList {
		got = append(got, a.pkg+"."+a.fn)
		if strings.TrimSpace(a.reason) == "" {
			t.Errorf("allow-list entry %s has no reason", a.pkg)
		}
		if a.beneath != (a.pkg == "internal/tools/release") {
			t.Errorf("allow-list entry %s: beneath is %v; only A2's entry covers the packages beneath it", a.pkg, a.beneath)
		}
	}
	want := []string{"internal/hostenv.Isolate", "internal/hostfs.", "cmd/metais.", "internal/dates.", "internal/tools/release.", "internal/tools/ux."}
	if !slices.Equal(got, want) {
		t.Errorf("the module allow-list is %q, want R2-143's, A2's and A8's %q", got, want)
	}

	// A2 item 1's symbols, spelled out, so the entry cannot grow with
	// hostSymbols: the environment reads, every os file function, os.TempDir
	// and os.MkdirTemp, and a main's arguments, streams and exit.
	builder := []string{
		"os.Environ", "os.Getenv", "os.LookupEnv",
		"os.Chmod", "os.Chown", "os.Chtimes", "os.CopyFS", "os.Create", "os.DirFS",
		"os.Lchown", "os.Link", "os.Lstat", "os.Mkdir", "os.MkdirAll", "os.NewFile",
		"os.Open", "os.OpenFile", "os.OpenInRoot", "os.OpenRoot", "os.Pipe", "os.ReadDir",
		"os.ReadFile", "os.Readlink", "os.Remove", "os.RemoveAll", "os.Rename", "os.Stat",
		"os.Symlink", "os.Truncate", "os.WriteFile",
		"os.TempDir", "os.MkdirTemp",
		"os.Args", "os.Stdin", "os.Stdout", "os.Stderr", "os.Exit",
	}
	slices.Sort(builder)
	for _, a := range moduleAllowList {
		if a.pkg != "internal/tools/release" {
			continue
		}
		syms := slices.Sorted(slices.Values(a.symbols))
		if !slices.Equal(syms, builder) || a.fn != "" || a.onlyArg != nil {
			t.Errorf("A2's entry allows %q (function %q, arguments %v), want exactly %q", syms, a.fn, a.onlyArg, builder)
		}
	}

	// A8 item 1's symbols, spelled out: the symbols the runner uses, and no
	// other. os/exec, the item's first entry, is no symbol of hostSymbols.
	runner := []string{
		"os.Environ",
		"os.Create", "os.DirFS", "os.Lstat", "os.MkdirAll", "os.ReadFile",
		"os.RemoveAll", "os.Rename", "os.Stat", "os.WriteFile",
		"time.Now",
		"os.Args", "os.Stdout", "os.Stderr", "os.Exit",
	}
	slices.Sort(runner)
	found := false
	for _, a := range moduleAllowList {
		if a.pkg != "internal/tools/ux" {
			continue
		}
		found = true
		syms := slices.Sorted(slices.Values(a.symbols))
		if !slices.Equal(syms, runner) || a.fn != "" || a.onlyArg != nil || a.beneath {
			t.Errorf("A8's entry allows %q (function %q, arguments %v, beneath %v), want exactly %q for its own package", syms, a.fn, a.onlyArg, a.beneath, runner)
		}
	}
	if !found {
		t.Error("the module allow-list has no entry for A8's UX runner")
	}
}
