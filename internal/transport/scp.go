package transport

import (
	"fmt"

	"github.com/mrshanahan/deploy-assets/internal/config"
	"github.com/mrshanahan/deploy-assets/internal/sshclient"
	"golang.org/x/crypto/ssh"
)

func NewScpTransport(name string, addr string, user string, keyPath string, keyPassphrase string) (config.Transport, error) {
	client, err := sshclient.CreateSshClient(addr, user, keyPath, keyPassphrase)
	if err != nil {
		return nil, err
	}
	return &scpTransport{name, client}, nil
}

type scpTransport struct {
	name   string
	client *ssh.Client
}

func (t *scpTransport) Validate(exec config.Executor) error {
	_, _, err := exec.ExecuteShell("which scp")
	if err != nil {
		return fmt.Errorf("could not find scp on path: %w", err)
	}
	return nil
}

func (t *scpTransport) TransferFile(src config.Executor, srcPath string, dst config.Executor, dstPath string) error {
	src.ExecuteCommand("scp", srcPath)
	return nil
}
