//go:build !windows

package main

import (
	"os"
	"syscall"
)

func flockExclusive(f *os.File, blocking bool) error {
	flags := syscall.LOCK_EX
	if !blocking {
		flags |= syscall.LOCK_NB
	}
	return syscall.Flock(int(f.Fd()), flags)
}

func flockUnlock(f *os.File) {
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:errcheck
}

func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}

func isProcessAlive(proc *os.Process) bool {
	return proc.Signal(syscall.Signal(0)) == nil
}

func terminateProcess(proc *os.Process) error {
	return proc.Signal(syscall.SIGTERM)
}
