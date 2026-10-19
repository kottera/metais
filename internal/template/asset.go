// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"bytes"
	"encoding/base64"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/liquid"
	"github.com/kottera/metais/internal/liquid/ast"
)

// AssetDir is the directory of a package's assets (ADR-001 D5). An asset's
// name, as a declaration and the asset tag write it, is its path beneath
// AssetDir.
const AssetDir = "assets/"

// The asset tag's modes (ADR-004 D13 items 2 to 4).
const (
	assetPath     = "path"
	assetContents = "contents"
	assetBase64   = "base64"
)

// The engine codes the asset tag reports at render, which renderFinding
// maps to their template codes (ADR-004 D16 item 1).
const (
	codeAssetNotWritten liquid.Code = "asset_not_written"
	codeAssetAmbiguous  liquid.Code = "asset_ambiguous"
	codeAssetTree       liquid.Code = "asset_tree"
	codeAssetNotText    liquid.Code = "asset_not_text"
)

// Placement is one file a render writes an asset to (ADR-005 D5 item 6):
// the tree it is written in, "" for the output directory and otherwise
// the container's path as Output's In gives it, and its path relative to
// that tree's root.
type Placement struct {
	Tree string
	Path string
}

// assetTag is ADR-004 D13's tag, {% asset "<path>" [,] [as: "<mode>"] %}.
// The dialect a template is parsed in holds one with no renderer, which
// checks its arguments; a Renderer renders through its own, which reads
// the asset from the package and the placement map from the render.
type assetTag struct{ r *Renderer }

func (assetTag) Name() string { return "asset" }

// Check refuses what is wrong whatever the data (ADR-004 D13 item 1): a
// path that is not one string literal, and an argument other than as:
// with the literal path, contents or base64. Each is a syntax finding:
// D16 names no other code for the tag's grammar.
func (assetTag) Check(t *ast.Tag) *liquid.Error {
	if _, ok := assetName(t); !ok {
		return &liquid.Error{Code: liquid.CodeSyntax, Detail: "asset takes the asset's path as one string literal first"}
	}
	for _, a := range t.Args[1:] {
		lit, ok := a.Value.(*ast.Literal)
		if a.Name != "as" || !ok || lit.Kind != ast.LitString || !slices.Contains([]string{assetPath, assetContents, assetBase64}, lit.Text) {
			return &liquid.Error{Code: liquid.CodeSyntax, Detail: `asset takes only as: "path", "contents" or "base64" after its path`}
		}
	}
	return nil
}

// assetName is the path an asset tag names, its first argument, when that
// is a string literal given by position.
func assetName(t *ast.Tag) (string, bool) {
	if len(t.Args) == 0 || t.Args[0].Name != "" {
		return "", false
	}
	lit, ok := t.Args[0].Value.(*ast.Literal)
	if !ok || lit.Kind != ast.LitString {
		return "", false
	}
	return lit.Text, true
}

// assetMode is the mode an asset tag gives, path when it gives none.
func assetMode(t *ast.Tag) string {
	for _, a := range t.Args[1:] {
		if a.Name == "as" {
			return a.Value.(*ast.Literal).Text
		}
	}
	return assetPath
}

// Render writes what the tag's mode names (ADR-004 D13 items 2 to 4),
// never escaped (item 5). The asset exists: its template's parse found it.
func (a assetTag) Render(c liquid.TagContext, t *ast.Tag) *liquid.Error {
	if a.r == nil {
		return &liquid.Error{Code: liquid.CodeInternal, Detail: "the asset tag rendered outside a Renderer"}
	}
	name, _ := assetName(t)
	if mode := assetMode(t); mode != assetPath {
		b, e := a.r.ts.readAsset(name)
		if e != nil {
			return e
		}
		if mode == assetBase64 {
			return c.Write(base64.StdEncoding.AppendEncode(nil, b))
		}
		b = bytes.TrimPrefix(b, []byte("\uFEFF"))
		if !utf8.Valid(b) {
			return &liquid.Error{Code: codeAssetNotText, Args: map[string]string{"asset": name}}
		}
		return c.Write(b)
	}
	var here, elsewhere []Placement
	for _, p := range a.r.assets[name] {
		if p.Tree == a.r.out.In {
			here = append(here, p)
		} else {
			elsewhere = append(elsewhere, p)
		}
	}
	args := map[string]string{"asset": name}
	switch {
	case len(here) == 1:
		return c.Write([]byte(relativePath(a.r.out.Path, here[0].Path)))
	case len(here) > 1:
		return &liquid.Error{Code: codeAssetAmbiguous, Args: args}
	case len(elsewhere) > 0:
		return &liquid.Error{Code: codeAssetTree, Args: args}
	}
	return &liquid.Error{Code: codeAssetNotWritten, Args: args}
}

// relativePath is the path of to relative to the directory of from, both
// paths from the root of one tree, with / separators: ../ for each of
// from's directories to has not in common, then the rest of to.
func relativePath(from, to string) string {
	dir := strings.Split(from, "/")
	dir = dir[:len(dir)-1]
	segs := strings.Split(to, "/")
	n := 0
	for n < len(dir) && n < len(segs)-1 && dir[n] == segs[n] {
		n++
	}
	return strings.Repeat("../", len(dir)-n) + strings.Join(segs[n:], "/")
}

// assetExists reports whether name names an asset of the package: a name
// confined beneath assets/, as a partial's is beneath templates/ (ADR-004
// D13 item 1, D8 item 2), matched byte for byte to an entry.
func (ts *Templates) assetExists(name string) bool {
	if !confined(name) {
		return false
	}
	_, err := ts.tree.Stat(AssetDir + name)
	return err == nil || canaryHasAsset(name)
}

// readAsset reads the asset name, which assetExists found, from the
// package.
func (ts *Templates) readAsset(name string) ([]byte, *liquid.Error) {
	info, err := ts.tree.Stat(AssetDir + name)
	if err != nil {
		if b, ok := canaryReadAsset(name); ok {
			return b, nil
		}
		return nil, &liquid.Error{Code: liquid.CodeInternal, Detail: "an asset tag names an asset its template's parse did not find: " + name}
	}
	b, err := readAll(ts.tree, info.Path, info.Size)
	if err != nil {
		panic("template: reading " + info.Path + ": " + err.Error())
	}
	return b, nil
}

// canaryAsset, when set, is where an asset the package lacks is read from
// instead. Only a canary build sets it (ADR-009 D16 item 2): the home
// canary falls back to the home directory, which the SC7 harness must
// catch. The tool's own build never does, and reads an asset only from the
// package's tree.
var canaryAsset func(name string) ([]byte, bool)

func canaryReadAsset(name string) ([]byte, bool) {
	if canaryAsset == nil {
		return nil, false
	}
	return canaryAsset(name)
}

func canaryHasAsset(name string) bool {
	_, ok := canaryReadAsset(name)
	return ok
}

// confined reports whether name is confined beneath a layout directory: no
// leading "/", no "\", no "." or ".." segment and no empty segment
// (ADR-004 D8 item 2).
func confined(name string) bool {
	if name == "" || strings.ContainsRune(name, '\\') {
		return false
	}
	return !slices.ContainsFunc(strings.Split(name, "/"), func(s string) bool { return s == "" || s == "." || s == ".." })
}

// assetTags returns every asset tag of a template whose path is a literal
// (ADR-004 D13 item 1): from its syntax tree when it parsed, and otherwise
// from its constructs before stop, where the parse stopped, as renderTags
// does, so that a missing asset is reported with the template's other
// parse findings in one pass (D16 item 4).
func assetTags(t *liquid.Template, text []byte, stop int) []renderTag {
	var out []renderTag
	if t != nil {
		ast.Walk(t.Root, func(n ast.Node) bool {
			if tag, ok := n.(*ast.Tag); ok && tag.Name == "asset" {
				if name, ok := assetName(tag); ok {
					out = append(out, renderTag{name, tag.Span})
				}
			}
			return true
		})
		return out
	}
	for _, c := range liquid.Constructs(string(text[:min(stop, len(text))])) {
		if c.Tag != "asset" || c.Span.Start >= stop {
			continue
		}
		if name, ok := literalName(c.Markup); ok {
			out = append(out, renderTag{name, c.Span})
		}
	}
	return out
}

// assetCodes maps the asset tag's render errors to their codes (ADR-004
// D16 item 1).
var assetCodes = map[liquid.Code]finding.Code{
	codeAssetNotWritten: CodeAssetNotWritten,
	codeAssetAmbiguous:  CodeAssetAmbiguous,
	codeAssetTree:       CodeAssetTree,
	codeAssetNotText:    CodeAssetNotText,
}
