// +build linux

package reloader
import (
	"github.com/sevlyar/go-daemon"
)

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
		if err := ctx.Release();err != nil {
			panic(err)
		}
	}()

	return r.Run()
}
