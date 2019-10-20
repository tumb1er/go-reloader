// +build linux

package reloader

import (
	"github.com/sevlyar/go-daemon"
	"os"
	"os/exec"
	"syscall"
)

// Daemonize detaches console application from terminal, making reloader a daemon.
func (r *Reloader) Daemonize() error {
	ctx := daemon.Context{}
	d, err := ctx.Reborn()
	if err != nil {
		return err
	}
	if d != nil {
		return nil
	}
	defer func() {
		if err := ctx.Release(); err != nil {
			panic(err)
		}
	}()

	return r.Run()
}

// StartChild starts new child process.
func (r *Reloader) StartChild() (*exec.Cmd, error) {
	var err error
	if r.cmd, err = NewExecutable(r.cmd.Path); err != nil {
		return nil, err
	}
	child := exec.Command(r.cmd.Path, r.args...)
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	child.Stdout = r.stdout
	child.Stderr = r.stderr
	if err := child.Start(); err != nil {
		return nil, err
	}
	return child, nil
}

// SetExecutable sets executable bit for a file in tmp directory.
func SetExecutable(name string) error {
	return os.Chmod(name, 0751)
}

func (r *Reloader) terminateProcess() error {
	return syscall.Kill(r.child.Process.Pid, syscall.SIGTERM)
}

func (r *Reloader) terminateProcessTree() error {
	return syscall.Kill(-r.child.Process.Pid, syscall.SIGTERM)
}
