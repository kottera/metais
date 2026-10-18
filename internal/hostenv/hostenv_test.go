// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostenv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kottera/metais/internal/testenv"
)

// childEnv marks the re-executed test binary that calls Isolate. Isolate
// clears the environment of the process it runs in, so it only ever runs in
// a child, never in the test binary the other tests share.
const childEnv = "METAIS_HOSTENV_ISOLATE_CHILD"

// observed is what the child saw after Isolate returned.
type observed struct {
	NoColor     bool
	Environ     []string
	LocalIsUTC  bool
	EpochFormat string
}

// TestIsolateChild is the child's half of TestIsolate. It does nothing
// unless the parent started it.
func TestIsolateChild(t *testing.T) {
	if os.Getenv(childEnv) != "1" {
		t.Skip("run only as TestIsolate's child")
	}
	d := Isolate()
	o := observed{
		NoColor:     d.NoColor,
		Environ:     os.Environ(),
		LocalIsUTC:  time.Local == time.UTC,
		EpochFormat: time.Unix(0, 0).Format("2006-01-02T15:04:05 -0700 MST"),
	}
	b, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("\nOBSERVED %s\n", b)
}

func runChild(t *testing.T, env []string) observed {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestIsolateChild$", "-test.count=1")
	cmd.Env = append([]string{childEnv + "=1", "HOME=" + testenv.Dir(t), "TZ=Pacific/Chatham"}, env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("child: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.Bytes(), stderr.Bytes())
	}
	_, line, ok := bytes.Cut(stdout.Bytes(), []byte("\nOBSERVED "))
	if !ok {
		t.Fatalf("child reported nothing; did it run?\nstdout:\n%s", stdout.Bytes())
	}
	line, _, _ = bytes.Cut(line, []byte("\n"))
	var o observed
	if err := json.Unmarshal(line, &o); err != nil {
		t.Fatalf("child report %q: %v", line, err)
	}
	return o
}

func TestIsolate(t *testing.T) {
	// Variables a careless tool might read for colour or for the terminal.
	// None of them may change the result; only NO_COLOR does.
	decoys := []string{"TERM=xterm-256color", "CLICOLOR=0", "CLICOLOR_FORCE=1", "FORCE_COLOR=1", "COLORTERM=truecolor", "NO_COLOUR=1", "no_color=1"}
	tests := []struct {
		name        string
		env         []string
		wantNoColor bool
	}{
		{"NO_COLOR unset", nil, false},
		{"NO_COLOR empty", []string{"NO_COLOR="}, false},
		{"NO_COLOR set", []string{"NO_COLOR=1"}, true},
		{"NO_COLOR set to any text", []string{"NO_COLOR=false"}, true},
		{"only decoys", decoys, false},
		{"NO_COLOR set among decoys", append([]string{"NO_COLOR=yes"}, decoys...), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := runChild(t, tt.env)
			if o.NoColor != tt.wantNoColor {
				t.Errorf("Display.NoColor = %v, want %v", o.NoColor, tt.wantNoColor)
			}
			if len(o.Environ) != 0 {
				t.Errorf("environment after Isolate = %q, want it empty", o.Environ)
			}
			if !o.LocalIsUTC {
				t.Errorf("time.Local after Isolate is not time.UTC")
			}
			if want := "1970-01-01T00:00:00 +0000 UTC"; o.EpochFormat != want {
				t.Errorf("epoch in local time = %q, want %q (TZ was Pacific/Chatham)", o.EpochFormat, want)
			}
		})
	}
}

// TestIsolateOrder pins what the child cannot observe: Isolate's body is
// exactly the NO_COLOR read, then os.Clearenv, then time.Local set to UTC,
// and nothing else in the package reads the environment.
func TestIsolateOrder(t *testing.T) {
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var steps []string
	for _, name := range files {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		if f.Name.Name != "hostenv" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			fd, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}
			inIsolate := fd.Recv == nil && fd.Name.Name == "Isolate"
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CallExpr:
					if sel := selector(n.Fun); sel == "os.Getenv" || sel == "os.LookupEnv" || sel == "os.Environ" || sel == "os.Clearenv" {
						step := sel
						if len(n.Args) == 1 {
							if lit, ok := n.Args[0].(*ast.BasicLit); ok {
								v, _ := strconv.Unquote(lit.Value)
								step += "(" + v + ")"
							}
						}
						if !inIsolate {
							t.Errorf("%s: %s outside Isolate", fset.Position(n.Pos()), step)
						}
						steps = append(steps, step)
					}
				case *ast.AssignStmt:
					for i, lhs := range n.Lhs {
						if selector(lhs) == "time.Local" {
							step := "time.Local=" + selector(n.Rhs[i])
							if !inIsolate {
								t.Errorf("%s: %s outside Isolate", fset.Position(n.Pos()), step)
							}
							steps = append(steps, step)
						}
					}
				}
				return true
			})
			return false
		})
	}
	want := []string{"os.Getenv(NO_COLOR)", "os.Clearenv", "time.Local=time.UTC"}
	if fmt.Sprint(steps) != fmt.Sprint(want) {
		t.Errorf("Isolate's steps = %q, want %q", steps, want)
	}
}

func selector(e ast.Expr) string {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return x.Name + "." + sel.Sel.Name
}

func TestIsTerminal(t *testing.T) {
	f, err := os.Create(filepath.Join(testenv.Dir(t), "stream"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	tests := []struct {
		name   string
		stream any
	}{
		{"regular file", f},
		{"pipe", w},
		{"in-memory writer", &bytes.Buffer{}},
		{"nil", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsTerminal(tt.stream) {
				t.Errorf("IsTerminal(%s) = true, want false", tt.name)
			}
		})
	}
}
