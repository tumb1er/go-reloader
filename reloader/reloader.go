package reloader

import (
	"context"
	"github.com/tumb1er/go-reloader/reloader/executable"
	"log"
	"os"
	"os/signal"
	"path"
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

	var reloaderContext context.Context
	reloaderContext, r.stopReloader = context.WithCancel(context.Background())

	interrupted := make(chan os.Signal, 1)
	signal.Notify(interrupted, os.Interrupt)

	childExited, stopChild, err := r.startChild(reloaderContext)
	if err != nil {
		r.logger.Fatalf("child start error")
	}

	ticker := time.NewTicker(r.interval)

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
				r.logger.Printf("switching %s", r.cmd.String())
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
			if err := r.checkExecutableError(r.self, r.startSelfUpdate); err != nil {
				return err
			}

			if running && (r.restart || updated) {
				if childExited, stopChild, err = r.startChild(reloaderContext); err != nil {
					return err
				}
			} else {
				r.logger.Print("terminating")
				// prevent multiple reads from closed channel
				childExited = make(chan int)
				r.stopReloader()
			}
		case <-ticker.C:
			// check child and stop it if updated
			r.checkExecutable(r.cmd, stopChild)
			// check self and stop reloader if updated
			r.checkExecutable(r.self, stopChild)
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

// startSelfUpdate starts new process for switching binaries and stops reloader
func (r Reloader) startSelfUpdate() error {
	args := make([]string, 0, len(os.Args))
	args = append(args, "--update", r.self.Path())
	args = append(args, os.Args[1:]...)
	var err error
	var cmd *executable.Executable
	updater := path.Join(r.staging, r.self.String())
	r.logger.Printf("running %s %v", updater, args)
	if cmd, err = executable.NewExecutable(updater, args...); err != nil {
		return err
	}
	if err = cmd.Start(r.stdout, r.stderr); err != nil {
		return err
	}
	r.stopReloader()
	return nil
}

func (r *Reloader) Update(what string, restart bool) error {
	r.logger.Printf("Updating %s %s...", what, r.version)
	var err error
	var cmd *executable.Executable
	if cmd, err = executable.NewExecutable(what, r.args...); err != nil {
		r.logger.Fatalf("self init failed %s", err.Error())
		return err
	}
	r.logger.Printf("switching from %s", r.staging)
	if err = cmd.Switch(r.staging); err != nil {
		r.logger.Fatalf("self switch failed: %s", err.Error())
		return err
	}
	if !restart {
		return nil
	}
	r.logger.Print("restarting")
	if err = cmd.Start(r.stdout, r.stderr); err != nil {
		r.logger.Fatalf("self restart failed: %s", err.Error())
		return err
	}
	return nil
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
