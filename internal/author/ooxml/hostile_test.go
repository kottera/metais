// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package ooxml

import (
	"fmt"
	"strings"
)

// fill repeats unit between head and tail until the whole is about size
// bytes.
func fill(size int, head, unit, tail string) string {
	n := max(1, (size-len(head)-len(tail))/len(unit))
	return head + strings.Repeat(unit, n) + tail
}

const hostileRoot = `<w:document xmlns:w="` + nsW + `" xmlns:a="` + nsA + `"><w:body>`

// hostileCases are templates of about size bytes built to make a scan that is not linear in
// its input's length show it: deep nesting, end tags that match nothing or
// pop the whole stack, a construct spanning every text element, a finding
// per text element, causes between pieces of nested containers,
// declarations by the thousand, constructs in one start tag, references
// and unclosed markup. Each is a function of its name and size, so a child
// process rebuilds the one it runs.
var hostileCases = map[string]func(size int) []Template{
	"deep-nesting": func(size int) []Template {
		n := size / 11
		return one(hostileRoot + strings.Repeat("<w:p><w:t>", n/2) + strings.Repeat("</w:t></w:p>", n/2))
	},
	"unmatched-end-tags": func(size int) []Template {
		n := size / 12
		return one(hostileRoot + strings.Repeat("<w:r>", n) + strings.Repeat("</w:x>", n))
	},
	"stack-popped-again-and-again": func(size int) []Template {
		return one(fill(size, hostileRoot+"<w:p>", strings.Repeat("<w:r>", 64)+"</w:r>"+"</w:p><w:p>", ""))
	},
	"one-construct-every-element": func(size int) []Template {
		return one(fill(size, hostileRoot+"<w:p><w:r><w:t>{{ a", `</w:t></w:r><w:proofErr/><w:r><w:t>&amp;b`, "}}</w:t></w:r></w:p>"))
	},
	"a-construct-per-element": func(size int) []Template {
		return one(fill(size, hostileRoot+"<w:p>", `<w:r><w:t>{{ a</w:t></w:r><w:r><w:t>b }}</w:t></w:r>`, "</w:p>"))
	},
	"a-delimiter-per-element": func(size int) []Template {
		return one(fill(size, hostileRoot+"<w:p>", `<w:t>{</w:t>`, "</w:p>"))
	},
	"nested-containers-with-causes": func(size int) []Template {
		n := size / 64
		return one(hostileRoot + strings.Repeat(`<w:p><w:t>{</w:t><w:proofErr/><w:ins>`, n) + strings.Repeat(`<w:t>{</w:t></w:ins></w:p>`, n))
	},
	"declarations": func(size int) []Template {
		var b strings.Builder
		b.WriteString(hostileRoot + "<w:p")
		for i := 0; b.Len() < size/2; i++ {
			fmt.Fprintf(&b, ` xmlns:p%d="%s"`, i, nsW)
		}
		b.WriteString(">")
		return one(fill(size, b.String(), `<w:r xmlns:w="`+nsW+`" xmlns="`+nsA+`"><t>{</t></w:r>`, ""))
	},
	"constructs-in-a-start-tag": func(size int) []Template {
		return one(fill(size, hostileRoot+"<w:t ", `{{ ">" }}{% if a > b %}`, ">"))
	},
	"references": func(size int) []Template {
		return one(fill(size, hostileRoot+"<w:p><w:t>{{ ", `&&#x10FFFF&#99999999999;&lt`, "</w:t><w:t>}}</w:t></w:p>"))
	},
	"lone-angles": func(size int) []Template {
		return one(fill(size, hostileRoot+"<w:p><w:t>{{", `< <1<=<-`, "</w:t><w:t>}}</w:t></w:p>"))
	},
	"unclosed-comments": func(size int) []Template {
		return one(fill(size, hostileRoot, `<!--<?<![CDATA[<!x `, ""))
	},
	"unclosed-attributes": func(size int) []Template {
		return one(fill(size, hostileRoot+`<w:t a="`, `{{ x }}'<w:t b='`, ""))
	},
	"bad-outputs": func(size int) []Template {
		return one(fill(size, hostileRoot+"<w:p><w:t>", `{{ }</w:t><w:t>`, "</w:t></w:p>"))
	},
}

// hostileGraphs are packages built to make the propagation of bindings
// between templates show work beyond its bound of the prefixes in use times
// the render edges: a cycle of templates each rendering two others, and a
// chain of templates each resolving, through its inheritance, one of the
// many prefixes its root declares.
var hostileGraphs = map[string]func() []Template{
	"free-prefixes-down-a-chain": func() []Template {
		const n = 2000
		var root strings.Builder
		root.WriteString(`<w:document xmlns:w="` + nsW + `"`)
		for i := range n {
			fmt.Fprintf(&root, ` xmlns:p%d="%s"`, i, nsW)
		}
		root.WriteString(`>{% render 't0.xml' %}</w:document>`)
		ts := []Template{{Path: "document.xml", Text: []byte(root.String())}}
		for i := range n {
			ts = append(ts, Template{
				Path: fmt.Sprintf("t%d.xml", i),
				Text: fmt.Appendf(nil, `<p%d:p><p%d:t>{{ a</p%d:t><p%d:t>b }}</p%d:t></p%d:p>{%% render 't%d.xml' %%}`, i, i, i, i, i, i, (i+1)%n),
			})
		}
		return ts
	},
	"render-chain": func() []Template {
		const n = 2000
		ts := make([]Template, n)
		for i := range ts {
			ts[i] = Template{
				Path: fmt.Sprintf("t%d.xml", i),
				Text: fmt.Appendf(nil, `<w:p xmlns:w="%s" xmlns:p%d="%s">{%% render 't%d.xml' %%}{%% render 't%d.xml' %%}<w:t>{{ a</w:t><w:t>b }}</w:t><p%d:t/><q%d:t/></w:p>`, nsW, i, nsA, (i+1)%n, (i*7+3)%n, i, i),
			}
		}
		return ts
	},
}

func one(src string) []Template { return []Template{{Path: "document.xml", Text: []byte(src)}} }
