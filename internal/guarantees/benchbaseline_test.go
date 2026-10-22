// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"metais.org/metais/internal/testenv"
)

// sandboxLine is the line a benchmark baseline carries to say it is not a
// measurement of a budget (docs/dev/benchmarks.md, AGENTS.md).
const sandboxLine = "# sandbox figure, not a budget"

// baselineFile is where a benchmarked package commits its baseline,
// relative to the package's directory.
var baselineFile = filepath.Join("testdata", "bench", "baseline.txt")

// benchmarks returns the benchmarks declared in the Go test files of each
// package directory below root, by the directory's slash-separated path
// relative to root. It visits the directories go test does, so it skips
// testdata and every directory whose name begins with a dot or an
// underscore.
func benchmarks(root string) (map[string][]string, error) {
	out := map[string][]string{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); p != root && (n == "testdata" || strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && isBenchmark(fn) {
				rel, err := filepath.Rel(root, filepath.Dir(p))
				if err != nil {
					return err
				}
				dir := filepath.ToSlash(rel)
				out[dir] = append(out[dir], fn.Name.Name)
			}
		}
		return nil
	})
	for _, names := range out {
		slices.Sort(names)
	}
	return out, err
}

// isBenchmark reports whether fn is a benchmark go test runs: named
// Benchmark, or Benchmark followed by a character that is not a lower-case
// letter, and taking one *testing.B.
func isBenchmark(fn *ast.FuncDecl) bool {
	rest, ok := strings.CutPrefix(fn.Name.Name, "Benchmark")
	if !ok {
		return false
	}
	if r, _ := utf8.DecodeRuneInString(rest); rest != "" && unicode.IsLower(r) {
		return false
	}
	params := fn.Type.Params.List
	if len(params) != 1 || len(params[0].Names) > 1 {
		return false
	}
	star, ok := params[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "B"
}

// missingBaselines returns what each benchmarked package below root lacks
// of its baseline: the file, its sandbox line, or a row of a benchmark it
// declares.
func missingBaselines(root string) ([]string, error) {
	benches, err := benchmarks(root)
	if err != nil {
		return nil, err
	}
	var bad []string
	for _, dir := range sortedKeys(benches) {
		file := filepath.Join(root, filepath.FromSlash(dir), baselineFile)
		b, err := os.ReadFile(file)
		if err != nil {
			bad = append(bad, fmt.Sprintf("%s declares %s and has no %s: %v", dir, strings.Join(benches[dir], ", "), filepath.ToSlash(baselineFile), err))
			continue
		}
		lines := map[string]bool{}
		var rows []string
		sc := bufio.NewScanner(bytes.NewReader(b))
		for sc.Scan() {
			lines[sc.Text()] = true
			rows = append(rows, sc.Text())
		}
		if !lines[sandboxLine] {
			bad = append(bad, fmt.Sprintf("%s/%s has no line %q", dir, filepath.ToSlash(baselineFile), sandboxLine))
		}
		for _, name := range benches[dir] {
			row := regexp.MustCompile(`^` + regexp.QuoteMeta(name) + `(/\S*)?(-\d+)?\s`)
			if !slices.ContainsFunc(rows, row.MatchString) {
				bad = append(bad, fmt.Sprintf("%s/%s has no row of %s", dir, filepath.ToSlash(baselineFile), name))
			}
		}
	}
	return bad, nil
}

// sortedKeys returns m's keys in order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// TestBenchBaselines holds every benchmarked package to its committed
// baseline (AGENTS.md; docs/dev/benchmarks.md): the file, the line that
// says it is a sandbox figure, and a row of every benchmark the package
// declares.
func TestBenchBaselines(t *testing.T) {
	benches, err := benchmarks(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(benches) == 0 {
		t.Fatal("no package below the module root declares a benchmark")
	}
	bad, err := missingBaselines(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range bad {
		t.Error(b)
	}
}

// TestBenchBaselinesControl runs the check over a tree with each omission
// planted, so a check gone blind fails the build.
func TestBenchBaselinesControl(t *testing.T) {
	root := testenv.Dir(t)
	const head = licenceHeader + "package p\n\nimport \"testing\"\n\n"
	const row = "\t       1\t1000 ns/op\n"
	good := "# baselines\n" + sandboxLine + "\ngoos: darwin\n"
	files := map[string]string{
		// Every benchmark has its row, with or without a CPU suffix and
		// by its sub-benchmarks alone; helpers are not benchmarks.
		"good/b_test.go": head + "func BenchmarkOne(b *testing.B) {}\nfunc BenchmarkTwo(b *testing.B) {}\nfunc Benchmark(b *testing.B) {}\n" +
			"func Benchmarkhelper(b *testing.B) {}\nfunc BenchmarkArgs(b *testing.B, n int) {}\nfunc (x T) BenchmarkMethod(b *testing.B) {}\nfunc BenchmarkT(t *testing.T) {}\n",
		"good/" + filepath.ToSlash(baselineFile):   good + "BenchmarkOne-12" + row + "BenchmarkTwo/sub-8" + row + "Benchmark" + row,
		"nofile/b_test.go":                         head + "func BenchmarkOne(b *testing.B) {}\n",
		"noline/b_test.go":                         head + "func BenchmarkOne(b *testing.B) {}\n",
		"noline/" + filepath.ToSlash(baselineFile): "# sandbox figure\nBenchmarkOne" + row,
		"norow/b_test.go":                          head + "func BenchmarkOne(b *testing.B) {}\nfunc BenchmarkOneMore(b *testing.B) {}\n",
		"norow/" + filepath.ToSlash(baselineFile):  good + "BenchmarkOneMore-12" + row + "# BenchmarkOne" + row,
		"nobench/x_test.go":                        head + "func TestX(t *testing.T) {}\n",
		"sub/testdata/b_test.go":                   head + "func BenchmarkSkipped(b *testing.B) {}\n",
		".hidden/b_test.go":                        head + "func BenchmarkSkipped(b *testing.B) {}\n",
	}
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := missingBaselines(root)
	if err != nil {
		t.Fatal(err)
	}
	var short []string
	for _, g := range got {
		dir, rest, _ := strings.Cut(g, "/")
		dir, _, _ = strings.Cut(dir, " ")
		switch {
		case strings.Contains(rest, "has no line"):
			short = append(short, dir+": sandbox line")
		case strings.Contains(rest, "has no row of "):
			short = append(short, dir+": row "+rest[strings.LastIndex(rest, " ")+1:])
		default:
			short = append(short, dir+": file")
		}
	}
	want := []string{"nofile: file", "noline: sandbox line", "norow: row BenchmarkOne"}
	if !slices.Equal(short, want) {
		t.Errorf("missingBaselines reported\n\t%q\nwant %q", got, want)
	}
}
