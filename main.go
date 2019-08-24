package main

import (
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"go-reloader/reloader"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func watch(c *cli.Context) error {
	var err error
	var self, child string
	if self, err = os.Executable(); err != nil {
		return err
	}

	r := reloader.NewReloader(self)
	if logfile := c.String("log"); logfile != "" {
		if l, err := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
			return err
		} else {
			defer reloader.CloseFile(l)
			r.SetLogger(log.New(l, "", log.LstdFlags))
		}
	}
	if stdout := c.String("stdout"); stdout != "" {
		if w, err := os.OpenFile(stdout, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
			return err
		} else {
			defer reloader.CloseFile(w)
			r.SetStdout(w)
		}
	}
	if stderr := c.String("stderr"); stderr != "" {
		if w, err := os.OpenFile(stderr, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
			return err
		} else {
			defer reloader.CloseFile(w)
			r.SetStderr(w)
		}
	}
	r.SetInterval(c.Duration("interval"))
	if err := r.SetStaging(c.String("staging")); err != nil {
		return err
	}
	args := c.Args()
	if len(args) == 0 {
		return errors.New("no child executable passed")
	}
	if child, err = filepath.Abs(args[0]); err != nil {
		return err
	}

	if c.Bool("tmp") {
		// Copy child executable to temporary file
		if child, err = copyToTemp(child); err != nil {
			return err
		}
		defer func() {
			if err := os.Remove(child); err != nil {
				panic(err)
			}
		}()
	}

	if err := r.SetChild(child, args[1:]...); err != nil {
		return err
	}

	if !c.Bool("daemonize") {
		return r.Run()
	} else {
		return r.Daemonize()
	}
}

func copyToTemp(child string) (string, error) {
	parts := strings.SplitN(filepath.Base(child), ".", 2)
	fmt.Println(1)
	basename := parts[0]
	if len(parts) > 1 {
		basename = fmt.Sprintf("%s.*.%s", parts[0], parts[1])
	} else {
		basename = fmt.Sprintf("%s.*", parts[0])
	}
	tmp, err := ioutil.TempFile("", basename)
	if err != nil {
		return "", err
	}
	defer reloader.CloseFile(tmp)
	fmt.Println(2)
	r, err := os.Open(child)
	if err != nil {
		return "", err
	}
	defer reloader.CloseFile(r)
	fmt.Println(3)
	if _, err := io.Copy(tmp, r); err != nil {
		return "", err
	}
	if err := reloader.SetExecutable(tmp); err != nil {
		return "", err
	}
	fmt.Println(4)
	child = tmp.Name()
	fmt.Println(child)
	return child, nil
}

func main() {
	app := cli.NewApp()
	app.Name = "reloader"
	app.Usage = "reloads an executable after an update"
	app.HideVersion = true
	app.ArgsUsage = "<cmd> [<arg>...]"
	app.UsageText = "reloader [options...] <cmd> [<arg>...]"
	app.Flags = []cli.Flag{
		cli.DurationFlag{
			Name:  "interval",
			Value: time.Minute,
			Usage: "update check interval",
		},
		cli.StringFlag{
			Name:  "staging",
			Value: "staging",
			Usage: "staging directory path",
		},
		cli.BoolFlag{
			Name:  "daemonize",
			Usage: "start as daemon/service",
		},
		cli.StringFlag{
			Name:  "log",
			Value: "",
			Usage: "reloader log file",
		},
		cli.StringFlag{
			Name:  "stdout",
			Value: "",
			Usage: "child process stdout file",
		},
		cli.StringFlag{
			Name:  "stderr",
			Value: "",
			Usage: "child process stderr file",
		},
		cli.BoolFlag{
			Name:  "tmp",
			Usage: "copy executable binary to temporary directory before start",
		},
	}
	app.Action = watch
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
