go-reloader
===========

Go-reloader is a command-line tool to restart running application or service after downloading updates.

After downloading a software update, you need to apply it. What if target application is still running? 
On Windows, you even can't replace executable binary before it exits. So, you need an instrument for replacing 
binary and it shouldn't be called from current process.

Go-reloader proposes to run separate executable to watch for update and automatically restart a child process, which is 
your real application. It supports both console applications, Windows services and Linux daemons.

Usage
-----

```shell script
$> reloader
  # interval of periodic file update checks
  --interval 1s
  # daemon/service mode
  --daemonize
  # reloader log file
  --log /tmp/reloader.log
  # child process stdout and stderr redirection
  --stdout /tmp/child.out.log
  --stderr /tmp/child.err.log
  # downloaded updates location
  --staging /tmp/updates
  # child executable and it's args
  ./sleep arg
```

Defaults
--------

* `interval` - 1 minute
* `log` - logs are written to stderr
* `stdout/stderr` - child output is redirected to stdout/stderr of reloader
* `staging` - default updates dir is reloader-s `$cwd/staging/`

Windows service
---------------

You may register a new service with `sc.exe` (for `LOCAL SYSTEM` service cwd is `C:\Windows\system32`):

```shell script

$> sc.exe create service_name binPath= "C:\reloader.exe --daemonize --staging C:\ C:\sleep.exe arg"
```

Linux daemon
------------

Go-reloader uses `go-daemon` to create detached process for reloader behaving like a unix daemon, but for now it doesn't
create pid file nor it doesn't double-detach.
