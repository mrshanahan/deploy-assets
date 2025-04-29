package executor

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/mrshanahan/deploy-assets/internal/config"
	"github.com/mrshanahan/deploy-assets/internal/util"
)

type localExecutor struct {
	name string
}

func NewLocalExecutor(name string) config.Executor {
	return &localExecutor{name}
}

func (e *localExecutor) Name() string { return e.name }

func (e *localExecutor) executeCommand(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)

	var stdoutBuffer, stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer
	err := command.Run()
	stdout := stdoutBuffer.String()
	stderr := stderrBuffer.String()
	slog.Debug("executed local command", "name", name, "args", args, "stdout", stdout, "stderr", stderr, "err", err)
	return stdout, stderr, err
}

func (e *localExecutor) executeCommandWithLogging(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)

	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutReader.Close()
	stdoutBuilder := &strings.Builder{}
	stdoutMultiWriter := io.MultiWriter(stdoutWriter, stdoutBuilder)
	command.Stdout = stdoutMultiWriter

	stderrReader, stderrWriter := io.Pipe()
	defer stderrReader.Close()
	stderrBuilder := &strings.Builder{}
	stderrMultiWriter := io.MultiWriter(stderrWriter, stderrBuilder)
	command.Stderr = stderrMultiWriter

	if err := command.Start(); err != nil {
		return "", "", fmt.Errorf("failed to start command: %v", err)
	}

	bufStdoutReader, bufStderrReader := bufio.NewScanner(stdoutReader), bufio.NewScanner(stderrReader)
	bufStdoutReader.Split(util.ScanUntil('\n', '\r'))
	bufStderrReader.Split(util.ScanUntil('\n', '\r'))

	stdoutDone, stderrDone := make(chan bool), make(chan bool)

	// TODO: Scanner.Err()

	go func() {
		for bufStdoutReader.Scan() {
			line := bufStdoutReader.Text()
			slog.Debug("local stdout", "location", e.name, "command-name", name, "line", line)
			time.Sleep(10 * time.Millisecond)
		}
		// slog.Debug("local stdout eof", "location", e.name, "command-name", name)
		stdoutDone <- true
	}()

	go func() {
		for bufStderrReader.Scan() {
			line := bufStderrReader.Text()
			slog.Debug("local stderr", "location", e.name, "command-name", name, "line", line)
			time.Sleep(10 * time.Millisecond)
		}
		// slog.Debug("local stderr eof", "location", e.name, "command-name", name)
		stderrDone <- true
	}()

	err := command.Wait()
	stdoutWriter.Close()
	stderrWriter.Close()
	<-stdoutDone
	<-stderrDone
	stdout := stdoutBuilder.String()
	stderr := stderrBuilder.String()
	slog.Debug("executed local command", "name", name, "args", args, "stdout", stdout, "stderr", stderr, "err", err)
	return stdout, stderr, err
}

func (e *localExecutor) ExecuteCommand(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)

	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutReader.Close()
	stdoutBuilder := &strings.Builder{}
	stdoutMultiWriter := io.MultiWriter(stdoutWriter, stdoutBuilder)
	command.Stdout = stdoutMultiWriter

	stderrReader, stderrWriter := io.Pipe()
	defer stderrReader.Close()
	stderrBuilder := &strings.Builder{}
	stderrMultiWriter := io.MultiWriter(stderrWriter, stderrBuilder)
	command.Stderr = stderrMultiWriter

	if err := command.Start(); err != nil {
		return "", "", fmt.Errorf("failed to start command: %v", err)
	}

	bufStdoutReader, bufStderrReader := bufio.NewScanner(stdoutReader), bufio.NewScanner(stderrReader)
	bufStdoutReader.Split(util.ScanUntil('\n', '\r'))
	bufStderrReader.Split(util.ScanUntil('\n', '\r'))

	stdoutDone, stderrDone := make(chan bool), make(chan bool)

	// TODO: Scanner.Err()

	go func() {
		for bufStdoutReader.Scan() {
			line := bufStdoutReader.Text()
			slog.Debug("local stdout", "location", e.name, "command-name", name, "line", line)
			time.Sleep(10 * time.Millisecond)
		}
		// slog.Debug("local stdout eof", "location", e.name, "command-name", name)
		stdoutDone <- true
	}()

	go func() {
		for bufStderrReader.Scan() {
			line := bufStderrReader.Text()
			slog.Debug("local stderr", "location", e.name, "command-name", name, "line", line)
			time.Sleep(10 * time.Millisecond)
		}
		// slog.Debug("local stderr eof", "location", e.name, "command-name", name)
		stderrDone <- true
	}()

	err := command.Wait()
	stdoutWriter.Close()
	stderrWriter.Close()
	<-stdoutDone
	<-stderrDone
	stdout := stdoutBuilder.String()
	stderr := stderrBuilder.String()
	slog.Debug("executed local command", "name", name, "args", args, "stdout", stdout, "stderr", stderr, "err", err)
	return stdout, stderr, err
}

// TODO: Make shell configurable
func (e *localExecutor) ExecuteShell(cmd string) (string, string, error) {
	return e.ExecuteCommand("bash", "-c", cmd)
}

func (e *localExecutor) Close() {}
