package reloader

import (
	"context"
	"github.com/tumb1er/go-reloader/reloader/executable"
	"log"
	"os"
	"os/signal"
	"time"
)

// Reloader watches, updates and restarts an executable.
type Reloader struct {
	Config
	// link to reloader binary itself
	self *executable.Executable
	// link to child executable binary
	cmd          *executable.Executable
	stopReloader context.CancelFunc
}

// startChild starts new child process and returns a channel that is closed when
// child process exits. When context is done, child process is terminated.
func (r *Reloader) startChild(ctx context.Context) (<-chan int, context.CancelFunc, error) {
	childContext, stopChild := context.WithCancel(ctx)
	var err error
	r.logger.Print("starting child")
	// initializing child process
	if r.cmd, err = executable.NewExecutable(r.child, r.args...); err != nil {
		r.logger.Fatalf("child init failed %s", err.Error())
		return nil, nil, err
	}

	if err := r.cmd.Start(r.stdout, r.stderr); err != nil {
		r.logger.Fatalf("child start failed: %s", err.Error())
		return nil, nil, err
	}
	r.logger.Print("child started")

	// start child process waiter
	ch := make(chan int)
	go func() {
		defer close(ch)
		r.logger.Print("waiting for child exit")
		if exitCode, err := r.cmd.Wait(); err != nil {
			r.logger.Fatalf("terminate wait: %e", err)
		} else {
			r.logger.Printf("child exited with exit code %d", exitCode)
		}
	}()

	// start context handler
	go func() {
		<-childContext.Done()
		r.logger.Print("terminating child")
		if err := r.cmd.Terminate(r.tree); err != nil {
			r.logger.Fatalf("terminate child: %s", err.Error())
		}
	}()

	return ch, stopChild, nil
}

func (r *Reloader) initSelf() error {
	var self string
	var err error
	if self, err = os.Executable(); err != nil {
		return err
	}
	if r.self, err = executable.NewExecutable(self); err != nil {
		return err
	}
	return nil
}

func (r *Reloader) Run() error {
	r.logger.Printf("Running %s...", r.version)
	if err := r.initSelf(); err != nil {
		return err
	}

	reloaderContext, stopReloader := context.WithCancel(context.Background())
	r.stopReloader = stopReloader

	interrupted := make(chan os.Signal, 1)
	signal.Notify(interrupted, os.Interrupt)

	childExited, stopChild, err := r.startChild(reloaderContext)
	if err != nil {
		r.logger.Fatalf("child start error")
	}

	ticker := time.NewTicker(time.Second * 5)

	running := true
	for {
		select {
		case <-reloaderContext.Done():
			r.logger.Print("reloader context done")
			return nil
		case <-interrupted:
			r.logger.Print("received interrupt signal")
			running = false
			stopChild()
		case <-childExited:
			r.logger.Print("child exited, checking for updates")
			var latest bool
			if latest, err = r.cmd.Latest(r.staging); err != nil {
				r.logger.Fatalf("child check error: %s", err.Error())
				return err
			}
			if !latest {
				r.logger.Print("child updated, switching")
				if err := r.cmd.Switch(r.staging); err != nil {
					r.logger.Fatalf("switch binary error: %s", err.Error())
					return err
				}
			}
			if running && (r.restart || !latest) {
				r.logger.Print("restarting child")
				childExited, stopChild, err = r.startChild(reloaderContext)
				if err != nil {
					r.logger.Fatalf("child start error")
				}
			} else {
				r.logger.Print("child exited, terminating")
				// prevent multiple reads from closed channel
				childExited = make(chan int)
				stopReloader()
			}
		case <-ticker.C:
			r.logger.Print("checking child")
			if latest, err := r.cmd.Latest(r.staging); err != nil {
				r.logger.Fatalf("child check error: %e", err)
			} else {
				if !latest {
					r.logger.Print("child updated, terminating")
					stopChild()
				}
			}
		}

	}
}

// NewReloader returns a new Reloader instance with default configuration.
func NewReloader(version string) *Reloader {
	return &Reloader{
		Config: Config{
			version:  version,
			staging:  "staging",
			interval: time.Minute,
			logger:   log.New(os.Stderr, "", log.LstdFlags),
			stdout:   os.Stdout,
			stderr:   os.Stderr,
		},
	}
}
