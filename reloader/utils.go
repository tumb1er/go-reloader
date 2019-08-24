package reloader

import "io"

func CloseFile(c io.Closer) {
	if err := c.Close(); err != nil {
		panic(err)
	}
}
