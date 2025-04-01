package executor

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/mrshanahan/deploy-assets/internal/config"
	"github.com/mrshanahan/deploy-assets/internal/util"
	"golang.org/x/crypto/ssh"
)

type sshClient struct {
	name        string
	client      *ssh.Client
	runElevated bool
}

func NewSSHExecutor(name string, addr string, user string, keyPath string, runElevated bool) (config.Executor, error) {
	client, err := openSSHConnection(addr, user, keyPath)
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
	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := c.client.NewSession()
	if err != nil {
		slog.Error("failed to create ssh session", "name", c.name, "run-elevated", c.runElevated)
		return "", "", err
	}
	defer session.Close()

	// TODO: Create single folder for all these files & then delete them

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

	stdout, stderr, err = c.executeCommand(runCmd)
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
	slog.Debug("executed ssh command", "cmd", cmd, "stdout", stdout, "stderr", stderr, "err", err)
	return stdout, stderr, err
}

func openSSHConnection(addr string, user string, keyPath string) (*ssh.Client, error) {
	// Significant components of this taken from example in docs:
	// https://pkg.go.dev/golang.org/x/crypto@v0.36.0/ssh#example-PublicKeys
	// https://pkg.go.dev/golang.org/x/crypto@v0.36.0/ssh#Dial

	// var hostKey ssh.PublicKey

	key, err := os.ReadFile(keyPath)
	if err != nil {
		slog.Error("unable to read private key", "key-path", keyPath, "err", err)
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		slog.Error("unable to parse private key", "key-path", keyPath, "err", err)
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		// HostKeyCallback: ssh.FixedHostKey(hostKey),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	slog.Debug("dialing ssh server", "addr", addr, "config", config)

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		slog.Error("unable to connect to remove server", "addr", addr, "config", config, "err", err)
		return nil, err
	}

	slog.Debug("successfully dialed ssh server", "addr", addr, "config", config)
	return client, nil
}
