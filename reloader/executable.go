package reloader

import (
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"time"
)


type Executable struct {
	Path string
	Checksum []byte
	Modified time.Time
}

func (e Executable) GetModified() (time.Time, error) {
	if fi, err := os.Stat(e.Path); err != nil {
		return e.Modified, err
	} else {
		return fi.ModTime(), nil
	}
}

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

func (e Executable) Latest(dir string) (bool, error) {
	path := filepath.Join(dir, filepath.Base(e.Path))
	if stage, err := NewExecutable(path); err != nil {
		if _, ok := err.(*os.PathError); !ok {
			return false, err
		} else {
			return true, nil
		}
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

func NewExecutable(path string) (*Executable, error) {
	var err error
	if path, err = filepath.Abs(path); err != nil {
		return nil, err
	}
	e := &Executable{
		Path:     path,
	}
	if e.Modified, err = e.GetModified(); err != nil {
		return nil, err
	}
	if e.Checksum, err = e.GetChecksum(); err != nil {
		return nil, err
	}
	return e, nil
}
