// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command netleak is the import rule's control: a main package importing
// net, which the rule must flag (ADR-010 D13 item 4).
package main

import "net"

func main() {
	_, _ = net.Dial("tcp", "127.0.0.1:1")
}
