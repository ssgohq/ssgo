//go:build windows

package runner

import (
	"os"
	"os/exec"
	"syscall"
)

// setPlatformSysProcAttr sets Windows-specific process attributes.
func setPlatformSysProcAttr(cmd *exec.Cmd) {
	// On Windows, we don't use process groups the same way
	// CREATE_NEW_PROCESS_GROUP allows sending CTRL_BREAK_EVENT
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// killProcessGroup kills a process on Windows.
// Windows doesn't have Unix-style process groups, so we kill the process directly.
func killProcessGroup(pid int, sig syscall.Signal) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}

// signalInterrupt returns a placeholder signal (not used on Windows).
func signalInterrupt() syscall.Signal {
	return syscall.Signal(0)
}

// signalTerminate returns a placeholder signal (not used on Windows).
func signalTerminate() syscall.Signal {
	return syscall.Signal(0)
}

// signalKill returns a placeholder signal (not used on Windows).
func signalKill() syscall.Signal {
	return syscall.Signal(0)
}
