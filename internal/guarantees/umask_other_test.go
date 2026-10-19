// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build !unix

package guarantees

import "os/exec"

// stub reports that this binary is never the umask stub: a process mask
// is POSIX's, and the determinism detector runs on darwin and linux.
func stub() bool { return false }

// umaskCommand runs bin with args, with no umask to set.
func umaskCommand(bin string, args []string) *exec.Cmd { return exec.Command(bin, args...) }

// nofileCommand reports that no open-file limit can be set here.
func nofileCommand(bin string, args []string) (*exec.Cmd, bool) { return nil, false }

// closedUmaskCommand reports that no umask can be set here.
func closedUmaskCommand(bin string, args []string) (*exec.Cmd, bool) { return nil, false }
