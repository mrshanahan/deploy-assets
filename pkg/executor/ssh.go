package executor

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/mrshanahan/deploy-assets/internal/sshclient"
	"github.com/mrshanahan/deploy-assets/internal/util"
	"github.com/mrshanahan/deploy-assets/pkg/config"
	"golang.org/x/crypto/ssh"
)

type sshClient struct {
	name        string
	client      *ssh.Client
	runElevated bool
}

func NewSSHExecutor(name string, addr string, user string, keyPath string, keyPassphrase string, runElevated bool) (config.Executor, error) {
	client, err := sshclient.CreateSshClient(addr, user, keyPath, keyPassphrase)
	if err != nil {
		return nil, err
	}
	return &sshClient{name, client, runElevated}, nil
}

func (c *sshClient) Name() string { return c.name }

func (c *sshClient) ExecuteCommand(name string, args ...string) (string, string, error) {
	var s strings.Builder
	s.WriteString(name)
	for _, a := range args {
		s.WriteRune(' ')
		s.WriteRune('\'')
		s.WriteString(a)
		s.WriteRune('\'')
	}
	return c.runCommandInSession(s.String())
}

func (c *sshClient) ExecuteShell(cmd string) (string, string, error) {
	return c.runCommandInSession(cmd)
}

func (c *sshClient) Close() {
	c.client.Close()
}

// TODO: Can we make this more efficient? I.e. re-using sessions
func (c *sshClient) runCommandInSession(cmd string) (string, string, error) {
	// TODO: Create single folder for all these files & then delete them

	slog.Debug("executing ssh command", "cmd", cmd)
	scriptPathBase64 := util.GetTempFilePath("deploy-assets-ssh-b64")
	scriptContentsBase64 := base64.StdEncoding.EncodeToString([]byte(cmd))
	stdout, stderr, err := c.executeCommand(fmt.Sprintf("echo '%s' > %s", scriptContentsBase64, scriptPathBase64))
	if err != nil {
		slog.Error("failed to create temp execution file", "executor", "ssh", "name", c.name, "run-elevated", c.runElevated, "stdout", stdout, "stderr", stderr, "err", err)
		return "", "", nil
	}
	defer c.executeCommand(fmt.Sprintf("rm %s", scriptPathBase64))

	// TODO: Check for base64 utility/use another workaround
	scriptPath := util.GetTempFilePath("deploy-assets-ssh")
	stdout, stderr, err = c.executeCommand(fmt.Sprintf("cat %s | base64 -d > %s", scriptPathBase64, scriptPath))
	if err != nil {
		slog.Error("failed to create temp execution file", "executor", "ssh", "name", c.name, "run-elevated", c.runElevated, "stdout", stdout, "stderr", stderr, "err", err)
		return "", "", nil
	}
	defer c.executeCommand(fmt.Sprintf("rm %s", scriptPath))

	// TODO: Option for shell
	var runCmd string
	if c.runElevated {
		runCmd = fmt.Sprintf("sudo bash %s", scriptPath)
	} else {
		runCmd = fmt.Sprintf("bash %s", scriptPath)
	}

	stdout, stderr, err = c.executeCommandWithLogging(runCmd)
	slog.Debug("executed ssh command", "cmd", cmd, "stdout", stdout, "stderr", stderr, "err", err)
	return stdout, stderr, err
}

func (c *sshClient) executeCommand(cmd string) (string, string, error) {
	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	session, err := c.client.NewSession()
	if err != nil {
		slog.Error("failed to create ssh session", "name", c.name, "run-elevated", c.runElevated)
		return "", "", err
	}
	defer session.Close()

	var stdoutBuffer, stderrBuffer bytes.Buffer
	session.Stdout = &stdoutBuffer
	session.Stderr = &stderrBuffer
	err = session.Run(cmd)
	stdout := stdoutBuffer.String()
	stderr := stderrBuffer.String()
	// slog.Debug("executed ssh command", "cmd", cmd, "stdout", stdout, "stderr", stderr, "err", err)
	return stdout, stderr, err
}

func (c *sshClient) executeCommandWithLogging(cmd string) (string, string, error) {
	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	session, err := c.client.NewSession()
	if err != nil {
		slog.Error("failed to create ssh session", "name", c.name, "run-elevated", c.runElevated)
		return "", "", err
	}
	defer session.Close()

	stdoutReader, stdoutWriter := io.Pipe()
	defer stdoutReader.Close()
	stdoutBuilder := &strings.Builder{}
	stdoutMultiWriter := io.MultiWriter(stdoutWriter, stdoutBuilder)
	session.Stdout = stdoutMultiWriter

	stderrReader, stderrWriter := io.Pipe()
	defer stderrReader.Close()
	stderrBuilder := &strings.Builder{}
	stderrMultiWriter := io.MultiWriter(stderrWriter, stderrBuilder)
	session.Stderr = stderrMultiWriter

	//slog.Debug("executing ssh command", "cmd", cmd)
	if err := session.Start(cmd); err != nil {
		return "", "", fmt.Errorf("failed to start ssh command: %v", err)
	}

	bufStdoutReader, bufStderrReader := bufio.NewScanner(stdoutReader), bufio.NewScanner(stderrReader)
	bufStdoutReader.Split(util.ScanUntil('\n', '\r'))
	bufStderrReader.Split(util.ScanUntil('\n', '\r'))

	stdoutDone, stderrDone := make(chan bool), make(chan bool)

	// TODO: Scanner.Err()
	go func() {
		for bufStdoutReader.Scan() {
			line := bufStdoutReader.Text()
			slog.Debug("ssh stdout", "location", c.name, "line", line)
			time.Sleep(10 * time.Millisecond)
		}
		stdoutDone <- true
	}()

	go func() {
		for bufStderrReader.Scan() {
			line := bufStderrReader.Text()
			slog.Debug("ssh stderr", "location", c.name, "line", line)
			time.Sleep(10 * time.Millisecond)
		}
		stderrDone <- true
	}()

	err = session.Wait()
	//slog.Debug("ssh command completed", "cmd", cmd, "err", err)
	stdoutWriter.Close()
	stderrWriter.Close()
	<-stderrDone
	<-stdoutDone
	stdout := stdoutBuilder.String()
	stderr := stderrBuilder.String()
	//slog.Debug("executed ssh command", "cmd", cmd, "stdout", stdout, "stderr", stderr, "err", err)
	return stdout, stderr, err
}
