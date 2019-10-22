// +build windows

package executable

import (
	"os/exec"
	"strconv"
	"syscall"
)

// setCmdFlags sets new process group flag
func (e *Executable) setCmdFlags() {
	child.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// terminateProcess sends CTRL_BREAK to child process
func (e *Executable) terminateProcess() error {
	ret, _, err := procGenerateConsoleCtrlEvent.Call(syscall.CTRL_BREAK_EVENT, uintptr(e.cmd.Process.Pid))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok {
			if errno == windows.ERROR_INVALID_PARAMETER {
				return nil
			}
		}
		return err
	}
	return nil
}

// terminateProcessTree uses taskkill to force-stop child process tree
func (e *Executable) terminateProcessTree() error {
	cmd := exec.Command("taskkill", "/f", "/t", "/pid", strconv.Itoa(e.cmd.Process.Pid))
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if ee.ExitCode() == 128 {
				// process not found - called when reloader is terminated via Ctrl+C in console
				return nil
			}
		}
		return err
	}
}
