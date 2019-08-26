package reloader

import (
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Executable is a structure representing an executable that may be updated.
type Executable struct {
	// full path to executable binary
	Path string
	// executable checksum
	Checksum []byte
	// executable last modification time
	Modified time.Time
}

// GetModified retrieves file modification time from file system.
func (e Executable) GetModified() (time.Time, error) {
	if fi, err := os.Stat(e.Path); err != nil {
		return e.Modified, err
	} else {
		return fi.ModTime(), nil
	}
}

// GetChecksum computes checksum for executable binary.
func (e Executable) GetChecksum() ([]byte, error) {
	h := sha256.New()
	if f, err := os.Open(e.Path); err != nil {
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
	path := filepath.Join(dir, filepath.Base(e.Path))
	if stage, err := NewExecutable(path); err != nil {
		return false, err
	} else {
		if stage.Modified.Before(e.Modified) {
			return true, nil
		}
		if bytes.Equal(stage.Checksum, e.Checksum) {
			return true, nil
		}
		return false, nil
	}
}

// Switch updates executable binary with new version from staging directory.
func (e Executable) Switch(dir string) error {
	if r, err := os.Open(filepath.Join(dir, filepath.Base(e.Path))); err != nil {
		return err
	} else {
		defer CloseFile(r)
		if w, err := os.OpenFile(e.Path, os.O_WRONLY, 0751); err != nil {
			return err
		} else {
			defer CloseFile(w)
			if _, err := io.Copy(w, r); err != nil {
				return err
			}
		}
	}
	return nil
}

// NewExecutable initializes an instance representing executable file.
func NewExecutable(path string) (*Executable, error) {
	var err error
	if path, err = filepath.Abs(path); err != nil {
		return nil, err
	}
	e := &Executable{
		Path: path,
	}
	if e.Modified, err = e.GetModified(); err != nil {
		return nil, err
	}
	if e.Checksum, err = e.GetChecksum(); err != nil {
		return nil, err
	}
	return e, nil
}
