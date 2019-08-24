package reloader

import (
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
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
	logger   *log.Logger
	// mainloop running flag
	running bool
	stderr  io.Writer
	stdout  io.Writer
}

func (r *Reloader) Run() error {
	r.logger.Print("Running...")
	var err error
	if r.self, err = NewExecutable(r.executable); err != nil {
		return err
	}
	r.running = true
	r.logger.Print("starting child...")
	if r.child, err = r.StartChild(); err != nil {
		return err
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	go r.WaitTerm(c)

	for r.running {
		r.Sleep()

		// checking reloader itself
		r.logger.Printf("checking self")
		if latest, err := r.self.Latest(r.staging); err != nil {
			r.logger.Printf("self check error: %v", err)
			return err
		} else {
			if !latest {
				r.logger.Print("reloader updated, exiting")
				return nil
			}
		}

		// checking child cmd update
		r.logger.Printf("checking child")
		if latest, err := r.cmd.Latest(r.staging); err == nil {
			if latest {
				continue
			}
		} else {
			r.logger.Printf("child check error: %v", err)
			return err
		}

		// stopping child process
		r.logger.Printf("cmd updated, stopping...")
		err := r.TerminateChild()
		if err != nil {
			return err
		}

		// switching binaries
		r.logger.Printf("cmd stopped, swithing...")
		if err := r.cmd.Switch(r.staging); err != nil {
			return err
		}

		// restarting child cmd
		r.logger.Print("cmd switched, starting...")
		if r.child, err = r.StartChild(); err != nil {
			return err
		}
		r.logger.Print("cmd started")
	}
	r.logger.Print("exit")
	return nil
}

// TerminateChild stops child process and waits for process exit
func (r *Reloader) TerminateChild() error {
	if err := r.child.Process.Kill(); err != nil {
		return err
	}
	if _, err := r.child.Process.Wait(); err != nil {
		return err
	}
	return nil
}

// Sleep waits for check interval with periodic checks of running flag
func (r *Reloader) Sleep() {
	iterations := int(r.interval.Seconds())
	for i := 0; i < iterations && r.running; i++ {
		time.Sleep(time.Second)
	}
}

// StartChild starts new child process.
func (r *Reloader) StartChild() (*exec.Cmd, error) {
	var err error
	if r.cmd, err = NewExecutable(filepath.Base(r.cmd.Path)); err != nil {
		return nil, err
	}
	child := exec.Command(r.cmd.Path, r.args...)
	child.Stdout = r.stdout
	child.Stderr = r.stderr
	if err := child.Start(); err != nil {
		return nil, err
	}
	return child, nil
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

// WaitTerm is a process termination handler.
func (r *Reloader) WaitTerm(c chan os.Signal) {
	<-c

	r.logger.Print("terminating reloader...")
	if err := r.TerminateChild(); err != nil {
		panic(err)
	}
	r.running = false
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
