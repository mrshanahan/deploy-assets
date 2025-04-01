package executor

import (
	"log/slog"
	"os/exec"
	"strings"

	"github.com/mrshanahan/deploy-assets/internal/config"
)

type localExecutor struct {
	name string
}

func NewLocalExecutor(name string) config.Executor {
	return &localExecutor{name}
}

func (e *localExecutor) Name() string { return e.name }

func (e *localExecutor) ExecuteCommand(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	var stdoutReceiver strings.Builder
	var stderrReceiver strings.Builder
	command.Stdout = &stdoutReceiver
	command.Stderr = &stderrReceiver

	err := command.Run()
	stdout := stdoutReceiver.String()
	stderr := stderrReceiver.String()
	slog.Debug("executed local command", "name", name, "args", args, "stdout", stdout, "stderr", stderr, "err", err)
	return stdout, stderr, err
}

// TODO: Make shell configurable
func (e *localExecutor) ExecuteShell(cmd string) (string, string, error) {
	return e.ExecuteCommand("bash", "-c", cmd)
}

func (e *localExecutor) Close() {}
