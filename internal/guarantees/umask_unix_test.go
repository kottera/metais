// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package guarantees

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// The names the test binary answers to as a stub: the umask stub, and the
// open-file stub, which runs a program with few file descriptors.
const (
	umaskStubName   = "metais-umask-stub"
	closedUmaskStub = "metais-umask-777-stub"
	nofileStubName  = "metais-nofile-stub"
)

// stub is the stubs: started under umaskStubName, it sets a umask of 077,
// and under nofileStubName an open-file limit of nofileLimit, and then
// executes os.Args[1] with the arguments after it, which so start under
// that setting; it returns only when the binary is not a stub.
func stub() bool {
	switch filepath.Base(os.Args[0]) {
	case umaskStubName:
		syscall.Umask(0o077)
	case closedUmaskStub:
		syscall.Umask(0o777)
	case nofileStubName:
		lim := syscall.Rlimit{Cur: nofileLimit, Max: nofileLimit}
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
			os.Stderr.WriteString("open-file stub: " + err.Error() + "\n")
			os.Exit(2)
		}
	default:
		return false
	}
	err := syscall.Exec(os.Args[1], os.Args[1:], os.Environ())
	os.Stderr.WriteString("umask stub: " + err.Error() + "\n")
	os.Exit(2)
	return true
}

// umaskCommand is the command that runs bin with args under a umask of
// 077: this test binary, linked as the stub beside bin.
func umaskCommand(bin string, args []string) *exec.Cmd {
	return stubCommand(umaskStubName, bin, args)
}

// closedUmaskCommand is the command that runs bin with args under a umask
// of 777, which grants nobody anything, and whether this platform has one.
func closedUmaskCommand(bin string, args []string) (*exec.Cmd, bool) {
	return stubCommand(closedUmaskStub, bin, args), true
}

// nofileCommand is the command that runs bin with args under an open-file
// limit of nofileLimit, and whether this platform can set one.
func nofileCommand(bin string, args []string) (*exec.Cmd, bool) {
	return stubCommand(nofileStubName, bin, args), true
}

// stubCommand is this test binary, linked beside bin as the stub name,
// running bin with args.
func stubCommand(name, bin string, args []string) *exec.Cmd {
	stubPath := filepath.Join(filepath.Dir(bin), name)
	if _, err := os.Lstat(stubPath); err != nil {
		self, err := os.Executable()
		if err == nil {
			err = os.Symlink(self, stubPath)
		}
		if err != nil {
			panic(err)
		}
	}
	return exec.Command(stubPath, append([]string{bin}, args...)...)
}
