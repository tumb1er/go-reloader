// +build windows

package reloader

import (
	"errors"
	"github.com/judwhite/go-svc/svc"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
)

var (
	kernel32                     = syscall.MustLoadDLL("kernel32.dll")
	procGenerateConsoleCtrlEvent = kernel32.MustFindProc("GenerateConsoleCtrlEvent")
	procAllocConsole             = kernel32.MustFindProc("AllocConsole")
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
	s.r.running = false
	if err := s.r.TerminateChild(); err != nil {
		s.r.logger.Fatal(err)
		return err
	}
	s.wg.Wait()
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

// SetExecutable is a stub of settings executable bit for a file in tmp directory.
// OS Windows does not need any file attributes to execute any file as exe.
//noinspection GoUnusedParameter
func SetExecutable(tmp *os.File) error {
	return nil
}

func callTaskKill(cmd *exec.Cmd) error {
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if ee.ExitCode() == 128 {
				// process not found - called when reloader is terminated via Ctrl+C in console
				return nil
			}
		}
		return err
	}
	return nil
}

func (r *Reloader) terminateProcess() error {
	ret, _, err := procGenerateConsoleCtrlEvent.Call(syscall.CTRL_BREAK_EVENT, uintptr(r.child.Process.Pid))
	if ret == 0 {
		return err
	}
	return nil
}

func (r *Reloader) terminateProcessTree() error {
	cmd := exec.Command("taskkill", "/f", "/t", "/pid", strconv.Itoa(r.child.Process.Pid))
	return callTaskKill(cmd)
}
