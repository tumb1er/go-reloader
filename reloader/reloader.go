package reloader

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"
)

// Reloader watches, updates and restarts an executable.
type Reloader struct {
	// link to reloader binary itself
	self *Executable
	// link to child executable binary
	cmd *Executable
	// path to reloader binary
	executable string
	// child process handler
	child *exec.Cmd
	// child process args
	args []string
	// path to staging directory
	staging string
	// update check interval
	interval time.Duration
	// terminate process tree flag
	tree bool
	// child auto restart flag
	restart      bool
	logger       *log.Logger
	stderr       io.Writer
	stdout       io.Writer
	stopReloader context.CancelFunc
}

// startChild starts new child process and returns a channel that is closed when
// child process exits. When context is done, child process is terminated.
func (r *Reloader) startChild(ctx context.Context) (<-chan int, context.CancelFunc, error) {
	childContext, stopChild := context.WithCancel(ctx)
	var err error
	r.logger.Print("starting child")
	// starting child process
	if r.child, err = r.StartChild(); err != nil {
		r.logger.Fatalf("exec child failed: %s", err.Error())
		return nil, nil, err
	}
	r.logger.Print("child started")

	// start child process waiter
	ch := make(chan int)
	go func() {
		defer close(ch)
		r.logger.Print("waiting for child exit")
		if _, err := r.child.Process.Wait(); err != nil {
			r.logger.Fatalf("terminate wait: %e", err)
		} else {
			r.logger.Print("child exited")
		}
	}()

	// start context handler
	go func() {
		<-childContext.Done()
		r.logger.Print("terminating child")
		if err := r.TerminateChild(); err != nil {
			r.logger.Fatalf("terminate child: %s", err.Error())
		}
	}()

	return ch, stopChild, nil
}

func (r *Reloader) Run() error {
	r.logger.Print("Running...")
	if executable, err := NewExecutable(r.executable); err != nil {
		return err
	} else {
		r.self = executable
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
	updateAvailable := false
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
			if running && updateAvailable {
				r.logger.Print("switching binary")
				if err := r.cmd.Switch(r.staging); err != nil {
					r.logger.Fatalf("switch binary error: %s", err.Error())
					return err
				}
			}
			if running && (r.restart || updateAvailable) {
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
			r.logger.Print("checking self")
			if latest, err := r.self.Latest(r.staging); err != nil {
				r.logger.Fatalf("self check error: %s", err.Error())
			} else if !latest {
				r.logger.Print("self updated, exiting")
				running = false
				stopChild()
				continue
			}

			r.logger.Print("checking child")
			if latest, err := r.cmd.Latest(r.staging); err != nil {
				r.logger.Fatalf("child check error: %e", err)
			} else {
				updateAvailable = !latest
				if !latest {
					r.logger.Print("child updated, terminating")
					stopChild()
				}
			}
		}

	}
}

// TerminateChild stops child process and waits for process exit
func (r *Reloader) TerminateChild() error {
	var killer func() error
	if !r.tree {
		killer = r.terminateProcess
	} else {
		killer = r.terminateProcessTree
	}
	if err := killer(); err != nil {
		r.logger.Printf("terminate process: %e", err)
		return err
	}
	return nil
}

// SetStaging configures updates directory path.
func (r *Reloader) SetStaging(staging string) error {
	var err error
	if r.staging, err = filepath.Abs(staging); err != nil {
		return err
	}
	return nil
}

// SetInterval configures update check interval.
func (r *Reloader) SetInterval(interval time.Duration) {
	r.interval = interval
}

// SetTerminateTree configures terminate process tree flag.
func (r *Reloader) SetTerminateTree(tree bool) {
	r.tree = tree
}

// SetChild configures child cmd and arguments.
func (r *Reloader) SetChild(child string, args ...string) error {
	var err error
	if r.cmd, err = NewExecutable(child); err != nil {
		return err
	}
	r.args = args
	return nil
}

// SetLogger configures reloader logger.
func (r *Reloader) SetLogger(logger *log.Logger) {
	r.logger = logger
}

// SetStdout configures child process stdout redirection.
func (r *Reloader) SetStdout(s io.Writer) {
	r.stdout = s
}

// SetStderr configures child process stderr redirection.
func (r *Reloader) SetStderr(s io.Writer) {
	r.stderr = s
}

func (r *Reloader) SetRestart(restart bool) {
	r.restart = restart
}

// NewReloader returns a new Reloader instance with default configuration.
func NewReloader(executable string) *Reloader {
	return &Reloader{
		executable: executable,
		staging:    "staging",
		interval:   time.Minute,
		logger:     log.New(os.Stderr, "", log.LstdFlags),
		stdout:     os.Stdout,
		stderr:     os.Stderr,
	}
}
