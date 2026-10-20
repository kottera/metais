// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// openPTY opens a new pseudo-terminal through /dev/ptmx: granted,
// unlocked, and its slave named by TIOCPTYGNAME, as posix_openpt,
// grantpt, unlockpt and ptsname do.
func openPTY() (master, slave *os.File, err error) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	fd := int(m.Fd())
	var name [128]byte
	for _, req := range []uint{unix.TIOCPTYGRANT, unix.TIOCPTYUNLK, unix.TIOCPTYGNAME} {
		arg := uintptr(0)
		if req == unix.TIOCPTYGNAME {
			arg = uintptr(unsafe.Pointer(&name[0]))
		}
		if _, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(req), arg); e != 0 {
			m.Close()
			return nil, nil, e
		}
	}
	path, _, _ := bytes.Cut(name[:], []byte{0})
	s, err := os.OpenFile(string(path), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		m.Close()
		return nil, nil, err
	}
	return m, s, nil
}
