// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"metais.org/metais/internal/testenv"
)

// The tests in this file prove ADR-009 A12's credential route with a fake
// credential and a fake upstream on 127.0.0.1: no test reaches
// api.anthropic.com or starts a session.

// fakeToken is a credential shaped as claude setup-token prints one. It
// is no secret: no upstream accepts it.
const fakeToken = "sk-ant-oat01-metais-ux-fake-credential-0123456789"

// upstreamSeen is what a fake upstream received.
type upstreamSeen struct {
	mu   sync.Mutex
	reqs []*http.Request
}

func (u *upstreamSeen) all() []*http.Request {
	u.mu.Lock()
	defer u.mu.Unlock()
	return slices.Clone(u.reqs)
}

// fakeUpstream answers every request with a streamed body, and keeps
// each request it received.
func fakeUpstream(t *testing.T) (*httptest.Server, *upstreamSeen) {
	t.Helper()
	seen := &upstreamSeen{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.mu.Lock()
		seen.reqs = append(seen.reqs, r.Clone(r.Context()))
		seen.mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "event: message_start\ndata: {}\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv, seen
}

// startTestRoute starts the route to a fake upstream.
func startTestRoute(t *testing.T) (*route, *upstreamSeen) {
	t.Helper()
	up, seen := fakeUpstream(t)
	rt, err := startRoute(fakeToken, up.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rt.close() })
	return rt, seen
}

// sessionRequest is a request as a nested claude makes one: the
// placeholder as its bearer token, to the address it was handed.
func sessionRequest(t *testing.T, base, method, path string, edit func(*http.Request)) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, base+path, strings.NewReader(`{"model":"claude-haiku-5-5"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+placeholderToken)
	if edit != nil {
		edit(req)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

// TestRouteRewritesTheCredential: a session's request reaches the
// upstream with the credential as its bearer token, the OAuth beta value
// merged into the session's own, and nothing of the session's
// credential: no placeholder and no x-api-key (A12 item 1).
func TestRouteRewritesTheCredential(t *testing.T) {
	rt, seen := startTestRoute(t)
	resp := sessionRequest(t, rt.url(), http.MethodPost, "/v1/messages?beta=true", func(r *http.Request) {
		r.Header.Set("x-api-key", "sk-ant-api03-the-session-s-own")
		r.Header.Add("anthropic-beta", "claude-code-20250219, interleaved-thinking-2025-05-14")
		r.Header.Add("anthropic-beta", "fine-grained-tool-streaming-2025-05-14")
	})
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "message_start") {
		t.Fatalf("status %d, body %q", resp.StatusCode, body)
	}
	reqs := seen.all()
	if len(reqs) != 1 {
		t.Fatalf("the upstream received %d requests, want 1", len(reqs))
	}
	got := reqs[0]
	if v := got.Header.Values("Authorization"); !slices.Equal(v, []string{"Bearer " + fakeToken}) {
		t.Errorf("Authorization %q, want the credential's bearer token alone", v)
	}
	if v := got.Header.Values("X-Api-Key"); len(v) > 0 {
		t.Errorf("x-api-key %q forwarded", v)
	}
	want := []string{"claude-code-20250219", "interleaved-thinking-2025-05-14", "fine-grained-tool-streaming-2025-05-14", oauthBeta}
	if v := got.Header.Values("Anthropic-Beta"); len(v) != 1 || !slices.Equal(strings.Split(v[0], ","), want) {
		t.Errorf("anthropic-beta %q, want %q merged", v, want)
	}
	if got.URL.Path != "/v1/messages" || got.URL.RawQuery != "beta=true" || got.Method != http.MethodPost {
		t.Errorf("forwarded %s %s?%s", got.Method, got.URL.Path, got.URL.RawQuery)
	}
	for k, v := range got.Header {
		if strings.Contains(strings.Join(v, ","), placeholderToken) {
			t.Errorf("the upstream received the placeholder in %s", k)
		}
	}
}

// TestRouteBetaNotDoubled: a session that already sends the OAuth beta
// value, or sends none, gets it once.
func TestRouteBetaNotDoubled(t *testing.T) {
	rt, seen := startTestRoute(t)
	sessionRequest(t, rt.url(), http.MethodPost, "/v1/messages", func(r *http.Request) {
		r.Header.Set("anthropic-beta", oauthBeta+",claude-code-20250219")
	})
	sessionRequest(t, rt.url(), http.MethodGet, "/v1/models", nil)
	reqs := seen.all()
	if len(reqs) != 2 {
		t.Fatalf("the upstream received %d requests, want 2", len(reqs))
	}
	for i, want := range []string{oauthBeta + ",claude-code-20250219", oauthBeta} {
		if v := reqs[i].Header.Values("Anthropic-Beta"); !slices.Equal(v, []string{want}) {
			t.Errorf("request %d: anthropic-beta %q, want %q", i, v, want)
		}
	}
}

// TestRouteRefuses: every request A12 item 4 refuses gets 403, and none
// reaches the upstream.
func TestRouteRefuses(t *testing.T) {
	rt, seen := startTestRoute(t)
	for _, c := range []struct {
		name, method, path string
		edit               func(*http.Request)
	}{
		{"no credential", http.MethodPost, "/v1/messages", func(r *http.Request) { r.Header.Del("Authorization") }},
		{"a credential not the placeholder", http.MethodPost, "/v1/messages", func(r *http.Request) { r.Header.Set("Authorization", "Bearer sk-ant-oat01-other") }},
		{"the placeholder as an API key", http.MethodPost, "/v1/messages", func(r *http.Request) {
			r.Header.Del("Authorization")
			r.Header.Set("x-api-key", placeholderToken)
		}},
		{"two credentials", http.MethodPost, "/v1/messages", func(r *http.Request) { r.Header.Add("Authorization", "Bearer "+placeholderToken) }},
		{"a Host not the route's", http.MethodPost, "/v1/messages", func(r *http.Request) { r.Host = "rebound.example:80" }},
		{"a preflight", http.MethodOptions, "/v1/messages", nil},
		{"a method not GET or POST", http.MethodDelete, "/v1/files/f", nil},
		{"a path outside /v1/", http.MethodPost, "/api/oauth/profile", nil},
		{"/v1 alone", http.MethodGet, "/v1", nil},
		{"a path leaving /v1/", http.MethodGet, "/v1/../api/hello", nil},
		{"an escaped path leaving /v1/", http.MethodGet, "/v1/%2e%2e/api/hello", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			resp := sessionRequest(t, rt.url(), c.method, c.path, c.edit)
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("status %d, want 403", resp.StatusCode)
			}
		})
	}
	if n := len(seen.all()); n != 0 {
		t.Errorf("the upstream received %d refused requests", n)
	}
}

// TestRouteListensOnLoopback: the route's address is 127.0.0.1's.
func TestRouteListensOnLoopback(t *testing.T) {
	rt, _ := startTestRoute(t)
	if !strings.HasPrefix(rt.url(), "http://127.0.0.1:") {
		t.Errorf("the route is at %s", rt.url())
	}
}

// pipePath is a pipe holding data, by the path a shell's process
// substitution gives one, as -credential-file is given it.
func pipePath(t *testing.T, data string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("-credential-file reads a pipe by its /dev/fd path, which Windows does not have")
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	go func() {
		io.WriteString(w, data)
		w.Close()
	}()
	return fmt.Sprintf("/dev/fd/%d", r.Fd())
}

func TestReadCredential(t *testing.T) {
	for _, in := range []string{fakeToken, fakeToken + "\n", fakeToken + "\r\n"} {
		got, err := readCredential(pipePath(t, in))
		if err != nil || got != fakeToken {
			t.Errorf("%q: %q, %v", in, got, err)
		}
	}
}

// TestReadCredentialRefuses: every credential A12 item 2 refuses is
// refused, and no refusal prints what it read.
func TestReadCredentialRefuses(t *testing.T) {
	dir := testenv.Dir(t)
	regular := filepath.Join(dir, "token")
	if err := os.WriteFile(regular, []byte(fakeToken), 0o600); err != nil {
		t.Fatal(err)
	}
	const secret = "sk-ant-oat01-must-never-be-printed"
	for _, c := range []struct {
		name, path, want string
	}{
		{"a regular file", regular, "is not a pipe"},
		{"a directory", dir, "is not a pipe"},
		{"nothing", filepath.Join(dir, "absent"), "no such file"},
		{"empty", pipePath(t, ""), "is empty"},
		{"a newline alone", pipePath(t, "\n"), "is empty"},
		{"longer than 1 KiB", pipePath(t, "sk-ant-oat01-"+strings.Repeat("a", credentialMax)), "longer than 1 KiB"},
		{"two lines", pipePath(t, secret+"\n"+secret+"\n"), "not one line of printable ASCII"},
		{"a space", pipePath(t, "sk-ant-oat01- "+secret), "not one line of printable ASCII"},
		{"a control byte", pipePath(t, secret+"\x00"), "not one line of printable ASCII"},
		{"beyond ASCII", pipePath(t, secret+"é"), "not one line of printable ASCII"},
		{"an API key", pipePath(t, "sk-ant-api03-"+secret[len("sk-ant-oat01-"):]), "not a subscription OAuth token"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := readCredential(c.path)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("%v, want %q", err, c.want)
			}
			if strings.Contains(err.Error(), "must-never-be-printed") {
				t.Errorf("the refusal prints the credential: %v", err)
			}
		})
	}
}

// TestCLIRefusesCredentialRoute: -credential-file is refused beside a
// route of the environment's, and with -scorecard or -scrub, before
// anything is read.
func TestCLIRefusesCredentialRoute(t *testing.T) {
	out := filepath.Join(testenv.Dir(t), "set")
	for _, c := range []struct {
		name    string
		args    []string
		environ []string
		code    int
		want    string
	}{
		{"beside ANTHROPIC_BASE_URL", []string{"-out", out, "-credential-file", "/dev/null"},
			[]string{"PATH=/usr/bin:/bin", "ANTHROPIC_BASE_URL=http://127.0.0.1:9"}, 2, "ANTHROPIC_BASE_URL"},
		{"with -scorecard", []string{"-out", out, "-scorecard", "-credential-file", "/dev/null"},
			[]string{"PATH=/usr/bin:/bin"}, 2, "start no session"},
		{"with -scrub", []string{"-out", out, "-scrub", out + "-scrubbed", "-credential-file", "/dev/null"},
			[]string{"PATH=/usr/bin:/bin"}, 2, "start no session"},
		{"not a pipe", []string{"-out", out, "-credential-file", "/dev/null"},
			[]string{"PATH=/usr/bin:/bin"}, 1, "is not a pipe"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := cli(c.args, &stdout, &stderr, c.environ); code != c.code || !strings.Contains(stderr.String(), c.want) {
				t.Errorf("exit %d, stderr %q; want %d and %q", code, stderr.String(), c.code, c.want)
			}
		})
	}
}

// credentialSession is a fake session that does what a nested claude
// does with its environment: it calls the address it was handed with
// the credential it was handed, and writes its transcript.
func credentialSession(t *testing.T, f *fakeSet) func(inv invocation) string {
	return func(inv invocation) string {
		base, _ := lookup(inv.env, "ANTHROPIC_BASE_URL")
		tok, _ := lookup(inv.env, "ANTHROPIC_AUTH_TOKEN")
		req, err := http.NewRequest(http.MethodPost, base+"/v1/messages", strings.NewReader("{}"))
		if err != nil {
			t.Error(err)
			return ""
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return ""
		}
		resp.Body.Close()
		return initLine(session{work: inv.dir}, "2.1.299", "[]", `["debug"]`) + "\n" + resultLine + "\n"
	}
}

// TestRouteNeverHandsTheCredential is A12's headline property: with the
// route on, a set's sessions reach the upstream with the credential, and
// no session's environment, arguments, record, transcript or any file of
// its run holds it.
func TestRouteNeverHandsTheCredential(t *testing.T) {
	up, seen := fakeUpstream(t)
	f := newFakeSet(t, "C4", "I1")
	f.c.harness.(fakeClaude).baseURL = ""
	f.c.credential, f.c.upstream = fakeToken, up.URL
	f.session = credentialSession(t, f)
	if err := f.c.run(); err != nil {
		t.Fatalf("run: %v\n%s", err, f.out)
	}
	if len(f.sessions) != 4 {
		t.Fatalf("%d sessions, want 4", len(f.sessions))
	}
	for _, inv := range f.sessions {
		if base, _ := lookup(inv.env, "ANTHROPIC_BASE_URL"); !strings.HasPrefix(base, "http://127.0.0.1:") || base == up.URL {
			t.Errorf("session %s was handed %q, not the route's own address", inv.dir, base)
		}
		if strings.Contains(strings.Join(slices.Concat(inv.args, inv.env, []string{inv.path, inv.dir}), "\n"), fakeToken) {
			t.Errorf("session %s was handed the credential", inv.dir)
		}
	}
	reqs := seen.all()
	if len(reqs) != 4 {
		t.Fatalf("the upstream received %d requests, want one per session", len(reqs))
	}
	for _, r := range reqs {
		if r.Header.Get("Authorization") != "Bearer "+fakeToken {
			t.Errorf("the upstream received Authorization %q", r.Header.Get("Authorization"))
		}
	}
	err := fs.WalkDir(os.DirFS(f.c.out), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(filepath.Join(f.c.out, p))
		if err == nil && bytes.Contains(b, []byte(fakeToken)) {
			t.Errorf("%s holds the credential", p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(f.out.String(), fakeToken) {
		t.Error("the runner printed the credential")
	}
	for _, run := range []string{"C4-what-it-needs", "I1-install-and-render"} {
		r := f.record(t, run, "haiku")
		if i := slices.IndexFunc(r.Checks, func(c check) bool { return c.Name == checkCredentialNotSeen }); i < 0 || !r.Checks[i].Pass || !r.Valid {
			t.Errorf("%s: checks %+v", run, r.Checks)
		}
	}
}

// TestCredentialNotSeen: a session that writes the credential anywhere in
// its run is recorded invalid, and the set stops naming the file but not
// the credential.
func TestCredentialNotSeen(t *testing.T) {
	up, _ := fakeUpstream(t)
	for _, where := range []string{"transcript", "work", "home"} {
		t.Run(where, func(t *testing.T) {
			f := newFakeSet(t, "C4")
			f.c.models = []string{"haiku"}
			f.c.credential, f.c.upstream = fakeToken, up.URL
			f.session = func(inv invocation) string {
				t := initLine(session{work: inv.dir}, "2.1.299", "[]", `["debug"]`) + "\n" + resultLine + "\n"
				switch where {
				case "transcript":
					t += `{"type":"user","note":"` + fakeToken + `"}` + "\n"
				case "work":
					os.WriteFile(filepath.Join(inv.dir, "leak.txt"), []byte("x"+fakeToken+"x"), 0o644)
				case "home":
					home, _ := lookup(inv.env, "HOME")
					os.MkdirAll(filepath.Join(home, ".config"), 0o755)
					os.WriteFile(filepath.Join(home, ".config", "leak"), []byte(fakeToken), 0o644)
				}
				return t
			}
			err := f.c.run()
			if err == nil || !strings.Contains(err.Error(), checkCredentialNotSeen) || strings.Contains(err.Error(), fakeToken) {
				t.Fatalf("run: %v", err)
			}
			r := f.record(t, "C4-what-it-needs", "haiku")
			i := slices.IndexFunc(r.Checks, func(c check) bool { return c.Name == checkCredentialNotSeen })
			if r.Valid || i < 0 || r.Checks[i].Pass || strings.Contains(r.Checks[i].Detail, fakeToken) {
				t.Errorf("valid %v, checks %+v", r.Valid, r.Checks)
			}
		})
	}
}

// TestNoRouteNoCheck: without the route, a run makes no
// credential-not-seen check, so its checks are what they were.
func TestNoRouteNoCheck(t *testing.T) {
	f := newFakeSet(t, "C4")
	f.c.models = []string{"haiku"}
	if err := f.c.run(); err != nil {
		t.Fatal(err)
	}
	if got := names(f.record(t, "C4-what-it-needs", "haiku").Checks); slices.Contains(got, checkCredentialNotSeen) {
		t.Errorf("checks %q", got)
	}
}
