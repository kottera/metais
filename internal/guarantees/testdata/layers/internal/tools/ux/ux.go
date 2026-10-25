// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command ux keeps only-referrers' exception for the UX runner, and
// only-importers' for its credential route's net and net/http.
package main

import (
	"net"
	"net/http/httputil"
	"os"

	"metais.org/metais/internal/tools/ux/scenario"
)

var _, _ = net.Listen, httputil.NewSingleHostReverseProxy

func main() {
	scenario.Load()
	os.Exit(len(os.Args))
}
