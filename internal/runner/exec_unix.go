//go:build !windows

package runner

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// Exec replaces the cuck process with the agent, so the agent owns the
// terminal, receives signals directly and its exit status is ours. On success
// it does not return.
func Exec(c *Command) (int, error) {
	path, argv := "/bin/sh", []string{"sh", "-c", c.Shell + ` "$@"`, "sh"}
	if c.Shell != "" {
		argv = append(argv, c.Extra...)
	} else {
		p, err := exec.LookPath(c.Argv[0])
		if err != nil {
			return 127, err
		}
		path, argv = p, c.Argv
	}
	signal.Reset()
	if err := syscall.Exec(path, argv, os.Environ()); err != nil {
		return 126, err
	}
	return 0, nil
}
