package transport

import (
	"fmt"

	"github.com/mrshanahan/deploy-assets/internal/sshclient"
	"github.com/mrshanahan/deploy-assets/internal/util"
	"github.com/mrshanahan/deploy-assets/pkg/config"
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

func (t *scpTransport) Yaml(indent int) string {
	propIndent := util.YamlIndentString(indent + util.TabsToIndent(1))
	return fmt.Sprintf(
		`%sscp:
%sname: %s
%saddr: %v
%suser: %s`,
		util.YamlIndentString(indent),
		propIndent, t.name,
		propIndent, t.client.RemoteAddr(),
		propIndent, t.client.User())
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
