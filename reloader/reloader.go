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
			r.logger.Print("exit")
			return nil
		case <-interrupted:
			r.logger.Print("received interrupt signal")
			running = false
			stopChild()
		case <-childExited:
			r.logger.Print("child exited")
			updated := false
			// check child and raise updated flag if child binary updated
			if err := r.checkExecutableError(r.cmd, func() error {
				if err := r.cmd.Switch(r.staging); err != nil {
					r.logger.Fatalf("switch binary error: %s", err.Error())
					return err
				}
				updated = true
				return nil
			}); err != nil {
				return err
			}

			// check self and lower running flag if self binary updated
			r.checkExecutable(r.self, func() {
				running = false
			})

			if running && (r.restart || updated) {
				if childExited, stopChild, err = r.startChild(reloaderContext); err != nil {
					return err
				}
			} else {
				r.logger.Print("terminating")
				// prevent multiple reads from closed channel
				childExited = make(chan int)
				stopReloader()
			}
		case <-ticker.C:
			// check child and stop it if updated
			r.checkExecutable(r.cmd, stopChild)
			// check self and stop reloader if updated
			r.checkExecutable(r.self, stopReloader)
		}
	}
}

// checkExecutableError checks executable for update and runs callback if update is found
func (r Reloader) checkExecutableError(cmd *executable.Executable, onUpdate func() error) error {
	what := cmd.String()
	r.logger.Printf("checking %s", what)
	if latest, err := cmd.Latest(r.staging); err != nil {
		r.logger.Fatalf("%s check error: %s", what, err.Error())
		return err
	} else {
		if !latest {
			r.logger.Printf("%s updated", what)
			return onUpdate()
		}
	}
	return nil
}

// checkExecutable is a helper for checkExecutableError that accepts function not returning error
func (r Reloader) checkExecutable(cmd *executable.Executable, onUpdate func()) {
	if err := r.checkExecutableError(cmd, func() error {
		onUpdate()
		return nil
	}); err != nil {
		panic(err)
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
