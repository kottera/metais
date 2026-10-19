// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquidspec

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/template"
)

// The protocol's error codes. A Liquid parse error is not one of them: it
// is a compile result. A render error is codeRender, the form liquid-spec's
// adapter at the pinned commit reads as a raised render error; its
// Adapter#render reads only a result's output, so a render error written
// as a result would read as empty output.
const (
	codeJSON           = -32700
	codeInvalidRequest = -32600
	codeNoMethod       = -32601
	codeInvalidParams  = -32602
	codeRender         = -32001
)

// maxLine bounds one message. liquid-spec's largest template and
// environment are far below it.
const maxLine = 64 << 20

// rpcError is a JSON-RPC error object.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// problem is the data of an invalid-params error: the parameter, and
// what is wrong with it.
type problem struct {
	Param   string `json:"param"`
	Message string `json:"message"`
}

// liquidError is a Liquid error as the protocol reports it: its type,
// parse_error or render_error, its message and its 1-based line.
type liquidError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
}

func invalidParams(param, format string, args ...any) *rpcError {
	msg := fmt.Sprintf(format, args...)
	return &rpcError{Code: codeInvalidParams, Message: "Invalid params: " + msg, Data: problem{Param: param, Message: msg}}
}

// response is a JSON-RPC response: a result or an error, never both.
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// server is one connection's state: the templates compiled so far, by id.
type server struct {
	templates map[string]*compiled
	next      int
	quit      bool
}

// newServer returns a connection's state.
func newServer() *server {
	return &server{templates: map[string]*compiled{}}
}

// compiled is a template that parsed: its package of templates, and the
// name of the one the request compiled.
type compiled struct {
	ts   *template.Templates
	name string
}

// serve answers each message read from r on w until r ends or a quit
// arrives. Only a failure to read or write ends it with an error; every
// fault of a message is answered.
func serve(r io.Reader, w io.Writer) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), maxLine)
	bw := bufio.NewWriter(w)
	s := newServer()
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		if resp := s.handle(line); resp != nil {
			b, err := json.Marshal(resp)
			if err != nil {
				return err
			}
			if _, err := bw.Write(append(b, '\n')); err != nil {
				return err
			}
			if err := bw.Flush(); err != nil {
				return err
			}
		}
		if s.quit {
			return nil
		}
	}
	return sc.Err()
}

// handle answers one message, or returns nil for a notification, which
// is never answered.
func (s *server) handle(line []byte) *response {
	var msg map[string]json.RawMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		var syntax *json.SyntaxError
		if errors.As(err, &syntax) || !json.Valid(line) {
			return errorResponse(nil, &rpcError{Code: codeJSON, Message: "Parse error: " + err.Error()})
		}
		return errorResponse(nil, &rpcError{Code: codeInvalidRequest, Message: "Invalid request: a message is a JSON object"})
	}
	id, hasID := msg["id"]
	if hasID && !validID(id) {
		return errorResponse(nil, &rpcError{Code: codeInvalidRequest, Message: "Invalid request: id is a string, a number or null"})
	}
	var version, method string
	if json.Unmarshal(msg["jsonrpc"], &version) != nil || version != "2.0" {
		return errorResponse(id, &rpcError{Code: codeInvalidRequest, Message: `Invalid request: jsonrpc must be "2.0"`})
	}
	if json.Unmarshal(msg["method"], &method) != nil || method == "" {
		return errorResponse(id, &rpcError{Code: codeInvalidRequest, Message: "Invalid request: method is a non-empty string"})
	}
	params := msg["params"]
	var result any
	var rerr *rpcError
	switch method {
	case "initialize":
		result, rerr = s.initialize(params)
	case "compile":
		result, rerr = s.compile(params)
	case "render":
		result, rerr = s.render(params)
	case "quit":
		s.quit = true
		result = struct{}{}
	default:
		rerr = &rpcError{Code: codeNoMethod, Message: "Method not found: " + method}
	}
	if !hasID {
		return nil
	}
	if rerr != nil {
		return errorResponse(id, rerr)
	}
	return &response{JSONRPC: "2.0", ID: id, Result: result}
}

func errorResponse(id json.RawMessage, e *rpcError) *response {
	if id == nil {
		id = json.RawMessage("null")
	}
	return &response{JSONRPC: "2.0", ID: id, Error: e}
}

// validID reports whether id is a JSON string, number or null.
func validID(id json.RawMessage) bool {
	var v any
	if json.Unmarshal(id, &v) != nil {
		return false
	}
	switch v.(type) {
	case nil, string, float64:
		return true
	}
	return false
}

// decodeParams decodes params, which may be absent, into v.
func decodeParams(params json.RawMessage, v any) *rpcError {
	if len(params) == 0 || string(params) == "null" {
		return nil
	}
	if err := json.Unmarshal(params, v); err != nil {
		return invalidParams("params", "%v", err)
	}
	return nil
}

// initializeResult is initialize's answer. The dialect is the gem's
// 5.14.0 (ADR-004 D1); features is empty, because the adapter's
// missing_features selects what runs.
type initializeResult struct {
	Version        string   `json:"version"`
	Implementation string   `json:"implementation"`
	LiquidVersion  string   `json:"liquid_version"`
	Features       []string `json:"features"`
}

func (s *server) initialize(params json.RawMessage) (any, *rpcError) {
	var p struct {
		Version *string `json:"version"`
	}
	if e := decodeParams(params, &p); e != nil {
		return nil, e
	}
	if p.Version != nil && *p.Version != "1.0" {
		return nil, invalidParams("version", "the protocol version is 1.0, not %q", *p.Version)
	}
	return initializeResult{Version: "1.0", Implementation: "metais", LiquidVersion: "5.14.0", Features: []string{}}, nil
}

// compileResult is compile's answer: the template's id, or a null id and
// the parse error.
type compileResult struct {
	TemplateID *string      `json:"template_id"`
	Error      *liquidError `json:"error,omitempty"`
}

func (s *server) compile(params json.RawMessage) (any, *rpcError) {
	var p struct {
		Template *string `json:"template"`
		Options  *struct {
			ErrorMode   *string `json:"error_mode"`
			LineNumbers *bool   `json:"line_numbers"`
		} `json:"options"`
		Filesystem map[string]string `json:"filesystem"`
	}
	if e := decodeParams(params, &p); e != nil {
		return nil, e
	}
	if p.Template == nil {
		return nil, invalidParams("template", "missing required parameter 'template' in compile request")
	}
	if p.Options != nil && p.Options.ErrorMode != nil && *p.Options.ErrorMode != "strict2" {
		return nil, invalidParams("options.error_mode", "the dialect parses only as strict2 (ADR-004 D1), not %q", *p.Options.ErrorMode)
	}
	tree, name := newTree(p.Filesystem, *p.Template)
	ts := template.NewTemplates(tree, limits.Defaults(), nil)
	for _, n := range ts.Reach(name) {
		if _, fs := ts.Parse(n); len(fs) > 0 {
			return compileResult{Error: parseError(fs[0])}, nil
		}
	}
	s.next++
	id := "tmpl_" + strconv.Itoa(s.next)
	s.templates[id] = &compiled{ts: ts, name: name}
	return compileResult{TemplateID: &id}, nil
}

// parseError is the protocol's parse error for f, the first finding of a
// template's parse.
func parseError(f finding.Finding) *liquidError {
	e := &liquidError{Type: "parse_error", Message: string(f.Code()) + ": " + f.Message()}
	if l, ok := f.Location(); ok {
		e.Line = l.Line
	}
	return e
}

// renderResult is render's answer: the output, and no errors, since a
// render error is raised (MTS-TPL-047).
type renderResult struct {
	Output string `json:"output"`
	Errors []any  `json:"errors"`
}

func (s *server) render(params json.RawMessage) (any, *rpcError) {
	var p struct {
		TemplateID  *string         `json:"template_id"`
		Environment json.RawMessage `json:"environment"`
		Options     *struct {
			StrictErrors   *bool           `json:"strict_errors"`
			ResourceLimits json.RawMessage `json:"resource_limits"`
			Registers      json.RawMessage `json:"registers"`
		} `json:"options"`
	}
	if e := decodeParams(params, &p); e != nil {
		return nil, e
	}
	if p.TemplateID == nil {
		return nil, invalidParams("template_id", "missing required parameter 'template_id' in render request")
	}
	c, ok := s.templates[*p.TemplateID]
	if !ok {
		return nil, invalidParams("template_id", "unknown template_id %q; call compile first", *p.TemplateID)
	}
	if o := p.Options; o != nil {
		if o.StrictErrors != nil && !*o.StrictErrors {
			return nil, invalidParams("options.strict_errors", "the dialect writes no error into its output (MTS-TPL-047); every render error is raised")
		}
		if len(o.ResourceLimits) > 0 && string(o.ResourceLimits) != "null" {
			return nil, invalidParams("options.resource_limits", "a render's limits are ADR-009's own, and the dialect takes no render score")
		}
		if len(o.Registers) > 0 && string(o.Registers) != "null" && !isObject(o.Registers) {
			return nil, invalidParams("options.registers", "registers is an object")
		}
	}
	globals, err := environment(p.Environment)
	if err != nil {
		return nil, invalidParams("environment", "%v", err)
	}
	t, _ := c.ts.Parse(c.name)
	lim := limits.Defaults()
	env := &liquid.Env{
		Dialect:  template.Dialect(),
		Globals:  globals,
		Partials: c.ts.Partials(),
		Meter: liquid.NewMeter(liquid.MeterLimits{
			RenderWork:     lim.Value(limits.RenderWork),
			OutputSize:     lim.Value(limits.OutputSize),
			AssignBytes:    lim.Value(limits.AssignBytes),
			LoopIterations: lim.Value(limits.LoopIterations),
		}),
		Limits: liquid.RenderLimits{
			PartialDepth: int(lim.Value(limits.PartialDepth)),
			ValueSize:    lim.Value(limits.ValueSize),
			ArraySize:    lim.Value(limits.ArraySize),
		},
	}
	var out bytes.Buffer
	if e := liquid.Render(&out, t, env); e != nil {
		return nil, &rpcError{Code: codeRender, Message: "render error", Data: liquidError{Type: "render_error", Message: e.Error(), Line: e.Line}}
	}
	return renderResult{Output: out.String(), Errors: []any{}}, nil
}

// isObject reports whether raw is a JSON object.
func isObject(raw json.RawMessage) bool {
	var m map[string]json.RawMessage
	return json.Unmarshal(raw, &m) == nil && m != nil
}

// environment reads a render's environment as the globals: an object of
// the data model, its numbers kept as written and bound as the data's
// are (liquid.Data). An absent or null environment binds nothing.
func environment(raw json.RawMessage) (value.Object, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return value.Object{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return value.Object{}, err
	}
	o, ok := liquid.Data(v).(value.Object)
	if !ok {
		return value.Object{}, errors.New("the environment is an object")
	}
	return o, nil
}

// decodeValue reads the next JSON value of dec as a value of the data
// model.
func decodeValue(dec *json.Decoder) (value.Value, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch tok := tok.(type) {
	case nil:
		return value.Null{}, nil
	case bool:
		return value.Bool(tok), nil
	case json.Number:
		return value.ParseNumber(string(tok))
	case string:
		return value.NewString(tok)
	case json.Delim:
		switch tok {
		case '[':
			var elems []value.Value
			for dec.More() {
				e, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				elems = append(elems, e)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return value.NewArray(elems...)
		case '{':
			var ms []value.Member
			for dec.More() {
				k, err := dec.Token()
				if err != nil {
					return nil, err
				}
				v, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				ms = append(ms, value.Member{Name: k.(string), Value: v})
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return value.NewObject(ms...)
		}
	}
	return nil, fmt.Errorf("unexpected token %v", tok)
}
