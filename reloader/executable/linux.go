// +build linux

package executable

import (
	"syscall"
)

// setCmdFlags sets new process group flag
func (e *Executable) setCmdFlags() {
	e.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// terminateProcess sends SIGTERM to child process
func (e *Executable) terminateProcess() error {
	return syscall.Kill(e.cmd.Process.Pid, syscall.SIGTERM)
}

// terminateProcessTree sends SIGTERM to child process tree
func (e *Executable) terminateProcessTree() error {
	return syscall.Kill(-e.cmd.Process.Pid, syscall.SIGTERM)
}
