//go:build windows

package main

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func flockExclusive(f *os.File, blocking bool) error {
	var flags uint32 = windows.LOCKFILE_EXCLUSIVE_LOCK
	if !blocking {
		flags |= windows.LOCKFILE_FAIL_IMMEDIATELY
	}
	ol := new(windows.Overlapped)
	return windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, ol)
}

func flockUnlock(f *os.File) {
	ol := new(windows.Overlapped)
	windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, ol) //nolint:errcheck
}

func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW,
		HideWindow:    true,
	}
}

func isProcessAlive(proc *os.Process) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(proc.Pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}

func terminateProcess(proc *os.Process) error {
	return proc.Kill()
}
