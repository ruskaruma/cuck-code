//go:build windows

package runner

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

// Exec runs the agent as a child with the console attached and returns its
// exit code. Windows has no exec(2), so cuck waits for the agent and ignores
// Ctrl+C itself, leaving the agent to handle it.
func Exec(c *Command) (int, error) {
	var cmd *exec.Cmd
	if c.Shell != "" {
		comspec := os.Getenv("ComSpec")
		if comspec == "" {
			comspec = "cmd.exe"
		}
		line := c.Shell
		for _, a := range c.Extra {
			line += " " + syscall.EscapeArg(a)
		}
		cmd = exec.Command(comspec)
		cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `/d /s /c "` + strings.TrimSpace(line) + `"`}
	} else {
		path, err := exec.LookPath(c.Argv[0])
		if err != nil {
			return 127, err
		}
		cmd = exec.Command(path, c.Argv[1:]...)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	signal.Ignore(os.Interrupt)
	defer signal.Reset(os.Interrupt)

	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), nil
	}
	if err != nil {
		return 126, err
	}
	return 0, nil
}
