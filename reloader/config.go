package reloader

import (
	"io"
	"log"
	"path/filepath"
	"time"
)

type Config struct {
	// reloader version
	version string
	// path to staging directory
	staging string
	// update check interval
	interval time.Duration
	// terminate process tree flag
	tree bool
	// child auto restart flag
	restart bool
	// child executable
	child string
	// child process args
	args []string
	// check self binary flag
	selfCheck bool

	stderr io.Writer
	stdout io.Writer
	logger *log.Logger
}

// SetStaging configures updates directory path.
func (c *Config) SetStaging(staging string) error {
	var err error
	if c.staging, err = filepath.Abs(staging); err != nil {
		return err
	}
	return nil
}

// SetInterval configures update check interval.
func (c *Config) SetInterval(interval time.Duration) {
	c.interval = interval
}

// SetTerminateTree configures terminate process tree flag.
func (c *Config) SetTerminateTree(tree bool) {
	c.tree = tree
}

// SetLogger configures reloader logger.
func (c *Config) SetLogger(logger *log.Logger) {
	c.logger = logger
}

// SetStdout configures child process stdout redirection.
func (c *Config) SetStdout(s io.Writer) {
	c.stdout = s
}

// SetStderr configures child process stderr redirection.
func (c *Config) SetStderr(s io.Writer) {
	c.stderr = s
}

// SetRestart configures child automatic restarts.
func (c *Config) SetRestart(restart bool) {
	c.restart = restart
}

// SetChild configures child cmd and arguments.
func (c *Config) SetChild(child string, args ...string) {
	c.child = child
	c.args = args
}
