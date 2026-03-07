package transport

import (
	"fmt"

	"github.com/mrshanahan/deploy-assets/internal/sshclient"
	"github.com/mrshanahan/deploy-assets/internal/util"
	"github.com/mrshanahan/deploy-assets/pkg/config"
)

func NewScpTransport(name string, addr string, user string, keyPath string, keyPassphrase string) (config.Transport, error) {
	client, err := sshclient.CreateSshClient(addr, user, keyPath, keyPassphrase)
	if err != nil {
		return nil, err
	}
	client.Close()

	return &scpTransport{name, addr, user, keyPath, keyPassphrase}, nil
}

type scpTransport struct {
	name          string
	addr          string
	user          string
	keyPath       string
	keyPassphrase string
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
		propIndent, t.addr,
		propIndent, t.user)
}

func (t *scpTransport) Validate(exec config.Executor) error {
	_, _, err := exec.ExecuteShell("which scp")
	if err != nil {
		return fmt.Errorf("could not find scp on path: %w", err)
	}
	return nil
}

func (t *scpTransport) TransferFile(src config.Executor, srcPath string, dst config.Executor, dstPath string) error {
	if _, _, err := src.ExecuteCommand("scp", "-i", t.keyPath, srcPath, fmt.Sprintf("%s@%s:%s", t.user, t.addr, dstPath)); err != nil {
		return err
	}
	return nil
}
