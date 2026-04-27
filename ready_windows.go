//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type readyChannel struct {
	path string
}

// setupReadinessIPC creates a temp file for the child to write its port into.
// writeEnd is always nil on Windows (child opens the file directly via env var).
func setupReadinessIPC(cmd *exec.Cmd) (*readyChannel, *os.File, error) {
	f, err := os.CreateTemp("", "crit-ready-*.txt")
	if err != nil {
		return nil, nil, fmt.Errorf("creating readiness file: %w", err)
	}
	path := f.Name()
	f.Close()
	// Truncate so child can detect when content appears.
	os.Truncate(path, 0)
	cmd.Env = append(os.Environ(), "_CRIT_READY_FILE="+path)
	return &readyChannel{path: path}, nil, nil
}

func (rc *readyChannel) readPort() (portCh chan int, errCh chan error) {
	portCh = make(chan int, 1)
	errCh = make(chan error, 1)
	go func() {
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			data, err := os.ReadFile(rc.path)
			if err == nil && len(data) > 0 {
				line := strings.TrimSpace(string(data))
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
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		errCh <- fmt.Errorf("daemon did not write readiness file within 15 seconds")
	}()
	return portCh, errCh
}

func (rc *readyChannel) close() {
	os.Remove(rc.path)
}

// openReadyWriter returns a file for writing the port/error to the readiness channel.
// Returns nil if not running as daemon.
func openReadyWriter() *os.File {
	path := os.Getenv("_CRIT_READY_FILE")
	if path == "" {
		return nil
	}
	os.Unsetenv("_CRIT_READY_FILE")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil
	}
	return f
}
