package transport

import (
	"fmt"
	"path/filepath"
	"time"

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
	timestamp := time.Now().UnixMicro()
	dstTmpDirName := fmt.Sprintf("scp_%d", timestamp)
	dstTmpDirPath := filepath.Join("/tmp/deploy-assets/", dstTmpDirName)
	if _, _, err := dst.ExecuteCommand("mkdir", "-p", dstTmpDirPath); err != nil {
		return fmt.Errorf("failed to create tmp directory on dst: %w", err)
	}

	defer dst.ExecuteCommand("rm", "-rf", dstTmpDirPath)

	if _, _, err := src.ExecuteCommand("scp", "-i", t.keyPath, srcPath, fmt.Sprintf("%s@%s:%s", t.user, t.addr, dstTmpDirPath)); err != nil {
		return fmt.Errorf("failed to transfer file to remote: %w", err)
	}

	filename := filepath.Base(srcPath)
	dstTmpFilePath := filepath.Join(dstTmpDirPath, filename)
	if _, _, err := dst.ExecuteCommand("cp", dstTmpFilePath, dstPath); err != nil {
		return fmt.Errorf("failed to copy file from temp path to final path on remote: %w", err)
	}

	return nil
}
