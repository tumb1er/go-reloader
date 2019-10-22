package executable

import "io"

// CloseFile closes file and panics if close fails.
func CloseFile(c io.Closer) {
	if err := c.Close(); err != nil {
		panic(err)
	}
}
