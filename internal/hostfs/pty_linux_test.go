// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs_test

import (
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

// openPTY opens a new pseudo-terminal through /dev/ptmx: unlocked, and
// its slave named by its number, as posix_openpt, unlockpt and ptsname
// do.
func openPTY() (master, slave *os.File, err error) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	fd := int(m.Fd())
	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil {
		m.Close()
		return nil, nil, err
	}
	n, err := unix.IoctlGetInt(fd, unix.TIOCGPTN)
	if err != nil {
		m.Close()
		return nil, nil, err
	}
	s, err := os.OpenFile("/dev/pts/"+strconv.Itoa(n), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		m.Close()
		return nil, nil, err
	}
	return m, s, nil
}

// rawOutput turns off the output processing of the terminal f, so that it
// hands on the bytes written to it: no LF becomes CR LF.
func rawOutput(f *os.File) error {
	t, err := unix.IoctlGetTermios(int(f.Fd()), unix.TCGETS)
	if err != nil {
		return err
	}
	t.Oflag &^= unix.OPOST
	return unix.IoctlSetTermios(int(f.Fd()), unix.TCSETS, t)
}
