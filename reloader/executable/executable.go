package executable

import (
	"bytes"
	"crypto/sha256"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"
)

// Executable is a structure representing an executable that may be updated.
type Executable struct {
	// full path to executable binary
	path string
	// executable checksum
	checksum []byte
	// executable last modification time
	modified time.Time
	// program args
	args []string
	// child process handler
	cmd *exec.Cmd
}

// String returns an executable name
func (e Executable) String() string {
	return path.Base(e.path)
}

// GetModified retrieves file modification time from file system.
func (e Executable) GetModified() (time.Time, error) {
	if fi, err := os.Stat(e.path); err != nil {
		return e.modified, err
	} else {
		return fi.ModTime(), nil
	}
}

// GetChecksum computes checksum for executable binary.
func (e Executable) GetChecksum() ([]byte, error) {
	h := sha256.New()
	if f, err := os.Open(e.path); err != nil {
		return nil, err
	} else {
		defer CloseFile(f)
		if _, err := io.Copy(h, f); err != nil {
			return nil, err
		}
		return h.Sum(nil), nil
	}
}

// Latest checks whether Executable instance is running latest version of binary kept in staging directory.
func (e Executable) Latest(dir string) (bool, error) {
	p := filepath.Join(dir, filepath.Base(e.path))
	if stage, err := NewExecutable(p); err != nil {
		return false, err
	} else {
		if stage.modified.Before(e.modified) {
			return true, nil
		}
		if bytes.Equal(stage.checksum, e.checksum) {
			return true, nil
		}
		return false, nil
	}
}

// Switch updates executable binary with new version from staging directory.
func (e Executable) Switch(dir string) error {
	src := filepath.Join(dir, filepath.Base(e.path))
	var size int64
	if fileInfo, err := os.Stat(src); err != nil {
		return err
	} else {
		size = fileInfo.Size()
	}
	if r, err := os.Open(src); err != nil {
		return err
	} else {
		defer CloseFile(r)
		if w, err := os.OpenFile(e.path, os.O_WRONLY|os.O_TRUNC, 0751); err != nil {
			return err
		} else {
			defer CloseFile(w)
			if _, err := io.CopyN(w, r, size); err != nil {
				return err
			}
		}
	}
	return nil
}

// Start initializes and starts new subprocess
func (e *Executable) Start(stdout io.Writer, stderr io.Writer) error {
	e.cmd = exec.Command(e.path, e.args...)
	log.Printf("exec %s %v", e.path, e.args)
	e.cmd.Stdout = stdout
	e.cmd.Stderr = stderr
	e.setCmdFlags()
	return e.cmd.Start()
}

// Terminate stops child process and waits for process exit
func (e Executable) Terminate(tree bool) error {
	var killer func() error
	if tree {
		killer = e.terminateProcessTree
	} else {
		killer = e.terminateProcess
	}
	return killer()
}

// Wait waits for child process exit and return exit code
func (e Executable) Wait() (int, error) {
	if state, err := e.cmd.Process.Wait(); err != nil {
		return 0, err
	} else {
		return state.ExitCode(), nil
	}
}

func (e Executable) Path() string {
	return e.path
}

// NewExecutable initializes an instance representing executable file.
func NewExecutable(path string, args ...string) (*Executable, error) {
	var err error
	if path, err = filepath.Abs(path); err != nil {
		return nil, err
	}
	e := &Executable{
		path: path,
		args: args,
	}
	if e.modified, err = e.GetModified(); err != nil {
		return nil, err
	}
	if e.checksum, err = e.GetChecksum(); err != nil {
		return nil, err
	}
	return e, nil
}
