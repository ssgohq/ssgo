//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

// setPlatformSysProcAttr sets Unix-specific process attributes.
func setPlatformSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// killProcessGroup sends a signal to the process group.
func killProcessGroup(pid int, sig syscall.Signal) error {
	return syscall.Kill(-pid, sig)
}

// signalInterrupt returns the interrupt signal for graceful shutdown.
func signalInterrupt() syscall.Signal {
	return syscall.SIGINT
}

// signalTerminate returns the terminate signal.
func signalTerminate() syscall.Signal {
	return syscall.SIGTERM
}

// signalKill returns the kill signal for force termination.
func signalKill() syscall.Signal {
	return syscall.SIGKILL
}
