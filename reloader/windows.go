// +build windows

package reloader

import (
	"errors"
	"github.com/judwhite/go-svc/svc"
	"sync"
	"syscall"
)

type Service struct {
	r *Reloader
	wg sync.WaitGroup
}

func (s Service) Start() error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.r.Run(); err != nil {
			panic(err)
		}
	}()
	return nil
}

func (s Service) Stop() error {
	s.r.running = false
	if err := s.r.TerminateChild(); err != nil {
		s.r.logger.Fatal(err)
		return err
	}
	s.wg.Wait()
	return nil
}

func (s Service) Init(env svc.Environment) error {
	if !env.IsWindowsService() {
		return errors.New("not a windows service")
	}
	return nil
}


func (r *Reloader) Daemonize() error {
	s := Service{r: r}
	return svc.Run(s, syscall.SIGTERM, syscall.SIGINT)
}

