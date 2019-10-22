package executable

import (
	"io"
	"os"
)

// CloseFile closes file and panics if close fails.
func CloseFile(c io.Closer) {
	if err := c.Close(); err != nil {
		panic(err)
	}
}

// PerformSwitch updates executable binary with new version from staging directory.
func PerformSwitch(src, dst string) error {
	if r, err := os.Open(src); err != nil {
		return err
	} else {
		defer CloseFile(r)
		if w, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0751); err != nil {
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
