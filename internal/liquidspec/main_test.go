// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquidspec

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

var (
	serveFlag   = flag.Bool("serve", false, "serve liquid-spec's driver protocol on standard input and output, and run no test")
	resultsFlag = flag.String("results", "", "liquid-spec's --json result of the pinned selection, for TestDelegatedRun to judge")
)

// TestMain is the program: with -serve the test binary is the server
// liquid-spec runs, and runs no test.
func TestMain(m *testing.M) {
	flag.Parse()
	if *serveFlag {
		if err := serve(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "liquidspec:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}
