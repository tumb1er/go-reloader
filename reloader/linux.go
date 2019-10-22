// +build linux

package reloader

import (
	"github.com/sevlyar/go-daemon"
	"os"
	"os/exec"
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

func (r Reloader) RestartDaemon(name string) error {
	r.logger.Printf("Restaring daemon %s", name)
	cmd := exec.Command("service", name, "restart")
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr
	if err := cmd.Start(); err != nil {
		r.logger.Fatalf("service restart error: %s", err.Error())
		return err
	}
	if err := cmd.Wait(); err != nil {
		r.logger.Fatalf("service restart failed: %s", err.Error())
		return err
	}
	return nil
}

// SetExecutable sets executable bit for a file in tmp directory.
func SetExecutable(name string) error {
	return os.Chmod(name, 0751)
}
