package main

import (
	"errors"
	"github.com/urfave/cli"
	"go-reloader/reloader"
	"log"
	"os"
	"time"
)

func watch(c *cli.Context) error {
	if self, err := os.Executable(); err != nil {
		return err
	} else {
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
		if err := r.SetChild(args[0], args[1:]...); err != nil {
			return err
		}

		if !c.Bool("daemonize") {
			return r.Run()
		} else {
			return r.Daemonize()
		}
	}
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
	}
	app.Action = watch
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
