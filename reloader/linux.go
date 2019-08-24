// +build linux

package reloader

import (
	"github.com/sevlyar/go-daemon"
	"os"
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

// SetExecutable sets executable bit for a file in tmp directory.
func SetExecutable(tmp *os.File) error {
	return tmp.Chmod(0751)
}
