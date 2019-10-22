// +build windows

package reloader

import (
	"errors"
	"github.com/judwhite/go-svc/svc"
	"golang.org/x/sys/windows"
	"os/exec"
	"sync"
	"syscall"
)

var (
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	procAllocConsole = kernel32.MustFindProc("AllocConsole")
)

// service is an implementation of svc.service interface for running reloader as Windows service.
type service struct {
	r  *Reloader
	wg sync.WaitGroup
}

// Start starts reloader loop in separate goroutine and adds it to a wait group.
func (s service) Start() error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ret, _, err := procAllocConsole.Call()
		if ret == 0 {
			panic(err)
		}
		if err := s.r.Run(); err != nil {
			panic(err)
		}
	}()
	return nil
}

// Stops marks reloader as not running and terminates reloader child process.
func (s service) Stop() error {
	s.r.stopReloader()
	return nil
}

// Init checks whether current process is started with Service Control Manager.
func (s service) Init(env svc.Environment) error {
	if !env.IsWindowsService() {
		return errors.New("not a windows service")
	}
	return nil
}

// Daemonize makes a Windows service from current process.
func (r *Reloader) Daemonize() error {
	s := service{r: r}
	return svc.Run(s, syscall.SIGTERM, syscall.SIGINT)
}

func (r Reloader) RestartDaemon(name string) error {
	r.logger.Printf("Restaring daemon %s", name)
	cmd := exec.Command("sc", "stop", name)
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr
	if err := cmd.Start(); err != nil {
		r.logger.Fatalf("service stop error: %s", err.Error())
		return err
	}
	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == int(windows.ERROR_SERVICE_NOT_ACTIVE) {
				// skip Service not started error
				return nil
			}
		}
		r.logger.Fatalf("service stop failed: %s", err.Error())
		return err
	}

	cmd = exec.Command("sc", "start", "icm_client")
	cmd.Stdout = r.stdout
	cmd.Stderr = r.stderr
	if err := cmd.Start(); err != nil {
		r.logger.Fatalf("service start error: %s", err.Error())
		return err
	}
	if err := cmd.Wait(); err != nil {
		r.logger.Fatalf("service start failed: %s", err.Error())
		return err
	}
	return nil
}

// SetExecutable is a stub of settings executable bit for a file in tmp directory.
// OS Windows does not need any file attributes to execute any file as exe.
//noinspection GoUnusedParameter,GoUnusedExportedFunction
func SetExecutable(name string) error {
	return nil
}
