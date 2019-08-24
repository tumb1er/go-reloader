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

type Reloader struct {
	self       *Executable
	cmd        *Executable
	executable string
	child      *exec.Cmd
	args       []string
	staging    string
	interval   time.Duration
	logger     *log.Logger
	running    bool
	stderr     io.Writer
	stdout     io.Writer
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

	for {
		if !r.running {
			r.logger.Print("exit")
			return nil
		}
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
			if err != nil {
				r.logger.Printf("child check error: %v", err)
				return err
			}
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
}

func (r *Reloader) TerminateChild() error {
	if err := r.child.Process.Kill(); err != nil {
		return err
	}
	if _, err := r.child.Process.Wait(); err != nil {
		return err
	}
	return nil
}

func (r *Reloader) Sleep() {
	iterations := int(r.interval.Seconds())
	for i := 0; i < iterations && r.running; i++ {
		time.Sleep(time.Second)
	}
}

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

func (r *Reloader) SetStaging(staging string) error {
	var err error
	if r.staging, err = filepath.Abs(staging); err != nil {
		return err
	}
	return nil
}

func (r *Reloader) SetInterval(interval time.Duration) {
	r.interval = interval
}

func (r *Reloader) SetChild(child string, args ...string) error {
	var err error
	if r.cmd, err = NewExecutable(child); err != nil {
		return err
	}
	r.args = args
	return nil
}

func (r *Reloader) SetLogger(logger *log.Logger) {
	r.logger = logger
}

func (r *Reloader) SetStdout(s io.Writer) {
	r.stdout = s
}

func (r *Reloader) SetStderr(s io.Writer) {
	r.stderr = s
}

func (r *Reloader) WaitTerm(c chan os.Signal) {
	<-c
	if err := r.child.Process.Signal(syscall.SIGTERM); err != nil {
		panic(err)
	}
	if _, err := r.child.Process.Wait(); err != nil {
		panic(err)
	}
	r.logger.Print("terminating reloader...")
	r.running = false
}

func NewReloader(executable string) *Reloader {
	return &Reloader{
		executable: executable,
		staging:    "staging",
		interval:   time.Minute,
		logger:     log.New(os.Stderr, "", 0),
		stdout:     os.Stdout,
		stderr:     os.Stderr,
	}
}
