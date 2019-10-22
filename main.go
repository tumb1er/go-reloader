package main

import (
	"errors"
	"github.com/tumb1er/go-reloader/reloader"
	"github.com/tumb1er/go-reloader/reloader/executable"
	"github.com/urfave/cli"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var Version = "0.1.0"

func watch(c *cli.Context) error {
	var err error
	var child string
	r := reloader.NewReloader(c.App.Version)
	if logfile := c.String("log"); logfile != "" {
		if l, err := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
			return err
		} else {
			defer executable.CloseFile(l)
			r.SetLogger(log.New(l, "", log.LstdFlags))
		}
	}
	if stdout := c.String("stdout"); stdout != "" {
		if w, err := os.OpenFile(stdout, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
			return err
		} else {
			defer executable.CloseFile(w)
			r.SetStdout(w)
		}
	}
	if stderr := c.String("stderr"); stderr != "" {
		if w, err := os.OpenFile(stderr, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
			return err
		} else {
			defer executable.CloseFile(w)
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
			if err := os.RemoveAll(filepath.Dir(child)); err != nil {
				panic(err)
			}
		}()
	}
	if c.Bool("tree") {
		r.SetTerminateTree(true)
	}
	if c.Bool("restart") {
		r.SetRestart(true)
	}

	r.SetChild(child, args[1:]...)

	service := c.String("service")
	update := c.String("update")
	if service == "" {
		if update != "" {
			return r.Update(update, true)
		} else {
			return r.Run()
		}
	} else {
		if update != "" {
			if err := r.Update(update, false); err != nil {
				return err
			}
			return r.RestartDaemon(service)
		}
		return r.Daemonize()
	}
}

func copyToTemp(child string) (string, error) {
	basename := filepath.Base(child)
	dir, err := ioutil.TempDir("", strings.Split(basename, ".")[0])
	r, err := os.Open(child)
	if err != nil {
		return "", err
	}
	defer executable.CloseFile(r)

	dst := filepath.Join(dir, basename)
	w, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, 0751)
	if err != nil {
		return "", err
	}
	defer executable.CloseFile(w)
	if _, err := io.Copy(w, r); err != nil {
		return "", err
	}
	if err := reloader.SetExecutable(dst); err != nil {
		return "", err
	}
	return dst, nil
}

func main() {
	app := cli.NewApp()
	app.Name = "reloader"
	app.Usage = "reloads an executable after an update"
	app.Version = Version
	app.ArgsUsage = "<cmd> [<arg>...]"
	app.UsageText = "reloader [options...] <cmd> [<arg>...]"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "update",
			Usage: "perform update of executable and exit",
		},
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
		cli.StringFlag{
			Name:  "service",
			Usage: "daemon/service name",
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
		cli.BoolFlag{
			Name:  "tree",
			Usage: "terminate child process and it's process tree",
		},
		cli.BoolFlag{
			Name:  "restart",
			Usage: "restart child process after exit",
		},
	}
	app.Action = watch
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
