//go:build !windows

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type readyChannel struct {
	r *os.File
}

// setupReadinessIPC creates a pipe, configures cmd to inherit the write end as fd 3,
// and returns the read end wrapped in a readyChannel. writeEnd must be closed by the
// caller after cmd.Start().
func setupReadinessIPC(cmd *exec.Cmd) (*readyChannel, *os.File, error) {
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		return nil, nil, fmt.Errorf("creating readiness pipe: %w", err)
	}
	cmd.ExtraFiles = []*os.File{writeEnd}
	cmd.Env = append(os.Environ(), "_CRIT_READY_FD=3")
	return &readyChannel{r: readEnd}, writeEnd, nil
}

func (rc *readyChannel) readPort() (portCh chan int, errCh chan error) {
	portCh = make(chan int, 1)
	errCh = make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(rc.r)
		if !scanner.Scan() {
			errCh <- fmt.Errorf("daemon closed readiness pipe without writing")
			return
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "error:") {
			errCh <- fmt.Errorf("%s", strings.TrimPrefix(line, "error:"))
			return
		}
		port, err := strconv.Atoi(line)
		if err != nil {
			errCh <- fmt.Errorf("daemon wrote invalid port: %q", line)
			return
		}
		portCh <- port
	}()
	return portCh, errCh
}

func (rc *readyChannel) close() {
	rc.r.Close()
}

// openReadyWriter returns the write end of the readiness pipe (fd 3) if this
// process was spawned as a daemon. Returns nil if not running as daemon.
func openReadyWriter() *os.File {
	if os.Getenv("_CRIT_READY_FD") != "3" {
		return nil
	}
	os.Unsetenv("_CRIT_READY_FD")
	return os.NewFile(3, "ready-pipe")
}
