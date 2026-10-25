// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// The credential route of ADR-009 A12: on a host whose environment gives
// the runner no ANTHROPIC_BASE_URL, -credential-file hands it a
// subscription OAuth token through a pipe, and the runner serves the
// nested sessions a loopback proxy that authenticates upstream with it.
// Each session is still handed only R-UX-7's placeholder and the proxy's
// address, so no session can read the token.
const (
	// upstreamURL is the route's one upstream, which no flag or variable
	// changes.
	upstreamURL = "https://api.anthropic.com"
	// oauthBeta is the anthropic-beta value a request authenticated with
	// a subscription OAuth token carries.
	oauthBeta = "oauth-2025-04-20"
	// credentialMax is the longest credential read, in bytes.
	credentialMax = 1024
	// tokenPrefix begins a subscription OAuth token as claude setup-token
	// prints one.
	tokenPrefix = "sk-ant-oat"
)

// readCredential reads the token from path, which must be a pipe, such
// as a shell's process substitution makes, so the token is never written
// to a file or set in an environment. No refusal prints what it read.
func readCredential(p string) (string, error) {
	fi, err := os.Stat(p)
	if err != nil {
		return "", fmt.Errorf("-credential-file: %w", err)
	}
	if fi.Mode()&fs.ModeNamedPipe == 0 {
		return "", fmt.Errorf("-credential-file: %s is not a pipe: give the token by a pipe, such as <(security find-generic-password -s metais-ux -w), so it is never written to a file (ADR-009 A12)", p)
	}
	f, err := os.DirFS(filepath.Dir(p)).Open(filepath.Base(p))
	if err != nil {
		return "", fmt.Errorf("-credential-file: %w", err)
	}
	b, err := io.ReadAll(io.LimitReader(f, credentialMax+1))
	f.Close()
	if err != nil {
		return "", fmt.Errorf("-credential-file: %w", err)
	}
	tok := strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r")
	switch {
	case len(b) > credentialMax:
		return "", errors.New("-credential-file: the credential is longer than 1 KiB")
	case tok == "":
		return "", errors.New("-credential-file: the credential is empty")
	case strings.ContainsFunc(tok, func(r rune) bool { return r <= ' ' || r > '~' }):
		return "", errors.New("-credential-file: the credential is not one line of printable ASCII with no space")
	case !strings.HasPrefix(tok, tokenPrefix):
		return "", fmt.Errorf("-credential-file: the credential is not a subscription OAuth token as claude setup-token prints it (%s…)", tokenPrefix)
	}
	return tok, nil
}

// route is the loopback proxy: it forwards a session's request to the
// upstream with the token in place of the session's credential.
type route struct {
	token    string
	upstream *url.URL
	host     string // the address it listens on, host:port
	proxy    *httputil.ReverseProxy
	srv      *http.Server
}

// startRoute listens on 127.0.0.1, on a port the system picks, and
// serves the route until close.
func startRoute(token, upstream string) (*route, error) {
	u, err := url.Parse(upstream)
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("the credential route: %w", err)
	}
	r := &route{token: token, upstream: u, host: ln.Addr().String()}
	r.proxy = &httputil.ReverseProxy{
		Rewrite: r.rewrite,
		// A response is streamed as it comes, as the session reads it.
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "the credential route could not reach the upstream", http.StatusBadGateway)
		},
	}
	r.srv = &http.Server{Handler: r, ReadHeaderTimeout: time.Minute}
	go r.srv.Serve(ln)
	return r, nil
}

// url is the address a session is handed as ANTHROPIC_BASE_URL.
func (r *route) url() string { return "http://" + r.host }

func (r *route) close() error { return r.srv.Close() }

// ServeHTTP forwards a request only from a session as the runner started
// it, to the API's own paths, and refuses anything else with 403: a
// credential not the placeholder's, a Host not the route's own, which a
// name rebound to 127.0.0.1 would send, and any method but GET and POST,
// so a browser's preflight is refused too.
func (r *route) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	p := req.URL.Path
	why := ""
	switch {
	case req.Host != r.host:
		why = "its Host is not the route's address"
	case req.Method != http.MethodGet && req.Method != http.MethodPost:
		why = "its method is not GET or POST"
	case !strings.HasPrefix(p, "/v1/") || path.Clean(p) != p || req.URL.RawPath != "":
		why = "its path is not one of the API's, under /v1/"
	case !slices.Equal(req.Header.Values("Authorization"), []string{"Bearer " + placeholderToken}):
		why = "its credential is not the runner's placeholder"
	}
	if why != "" {
		http.Error(w, "the credential route refuses the request: "+why, http.StatusForbidden)
		return
	}
	r.proxy.ServeHTTP(w, req)
}

// rewrite sends a request to the upstream with the token as its bearer
// credential, and the OAuth beta value merged into the session's own; the
// session's Authorization and x-api-key are dropped.
func (r *route) rewrite(pr *httputil.ProxyRequest) {
	pr.SetURL(r.upstream)
	h := pr.Out.Header
	h.Del("X-Api-Key")
	h.Set("Authorization", "Bearer "+r.token)
	h.Set("Anthropic-Beta", mergeBeta(h.Values("Anthropic-Beta")))
}

// mergeBeta is the anthropic-beta values, each header's comma-separated
// list in order, with oauthBeta added unless one holds it, as one value.
func mergeBeta(values []string) string {
	var betas []string
	for _, v := range values {
		for b := range strings.SplitSeq(v, ",") {
			if b = strings.TrimSpace(b); b != "" && !slices.Contains(betas, b) {
				betas = append(betas, b)
			}
		}
	}
	if !slices.Contains(betas, oauthBeta) {
		betas = append(betas, oauthBeta)
	}
	return strings.Join(betas, ",")
}

// credentialNotSeen: no file of the run, its transcript and its
// standard error among them, holds the token. A session that wrote it
// somewhere, or a harness that recorded it, fails the run, and the set
// stops so the file is removed before anything is published.
func credentialNotSeen(runDir, token string) check {
	var seen []string
	err := fs.WalkDir(os.DirFS(runDir), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		b, err := os.ReadFile(filepath.Join(runDir, filepath.FromSlash(p)))
		if err != nil {
			return err
		}
		if bytes.Contains(b, []byte(token)) {
			seen = append(seen, p)
		}
		return nil
	})
	switch {
	case err != nil:
		return fail(checkCredentialNotSeen, "reading the run: %v", err)
	case len(seen) > 0:
		return fail(checkCredentialNotSeen, "the run holds the credential in %s", strings.Join(seen, ", "))
	}
	return pass(checkCredentialNotSeen)
}
