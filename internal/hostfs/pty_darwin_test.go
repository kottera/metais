// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs_test

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// openPTY opens a new pseudo-terminal through /dev/ptmx, granted and
// unlocked as grantpt and unlockpt do. Its slave is /dev/ttys and the
// master's minor device number in three digits, the name ptsname gives.
func openPTY() (master, slave *os.File, err error) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	fd := int(m.Fd())
	var st unix.Stat_t
	for _, step := range []func() error{
		func() error { return unix.IoctlSetInt(fd, unix.TIOCPTYGRANT, 0) },
		func() error { return unix.IoctlSetInt(fd, unix.TIOCPTYUNLK, 0) },
		func() error { return unix.Fstat(fd, &st) },
	} {
		if err := step(); err != nil {
			m.Close()
			return nil, nil, err
		}
	}
	s, err := os.OpenFile(fmt.Sprintf("/dev/ttys%03d", unix.Minor(uint64(st.Rdev))), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		m.Close()
		return nil, nil, err
	}
	return m, s, nil
}

// rawOutput turns off the output processing of the terminal f, so that it
// hands on the bytes written to it: no LF becomes CR LF.
func rawOutput(f *os.File) error {
	t, err := unix.IoctlGetTermios(int(f.Fd()), unix.TIOCGETA)
	if err != nil {
		return err
	}
	t.Oflag &^= unix.OPOST
	return unix.IoctlSetTermios(int(f.Fd()), unix.TIOCSETA, t)
}
