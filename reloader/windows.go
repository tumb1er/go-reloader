// +build windows

package reloader

import (
	"errors"
	"github.com/judwhite/go-svc/svc"
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

// SetExecutable is a stub of settings executable bit for a file in tmp directory.
// OS Windows does not need any file attributes to execute any file as exe.
//noinspection GoUnusedParameter,GoUnusedExportedFunction
func SetExecutable(name string) error {
	return nil
}
