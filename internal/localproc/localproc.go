package localproc

import (
	"os/exec"
	"strings"
)

func ExecuteCommand(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	var stdoutReceiver strings.Builder
	var stderrReceiver strings.Builder
	command.Stdout = &stdoutReceiver
	command.Stderr = &stderrReceiver

	err := command.Run()
	stdout := stdoutReceiver.String()
	stderr := stderrReceiver.String()
	return stdout, stderr, err
}
