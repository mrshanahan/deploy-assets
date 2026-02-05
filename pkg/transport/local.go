package transport

import (
	"os"
	"path/filepath"

	"github.com/mrshanahan/deploy-assets/internal/util"
	"github.com/mrshanahan/deploy-assets/pkg/config"
)

func NewLocalTransport() config.Transport {
	return &localTransport{}
}

type localTransport struct{}

func (t *localTransport) Yaml(indent int) string {
	return util.YamlIndentString(indent) + "local:"
}

func (t *localTransport) Validate(exec config.Executor) error {
	return nil
}

func (t *localTransport) TransferFile(src config.Executor, srcPath string, dst config.Executor, dstPath string) error {
	intDirPath := util.GetTempFilePath("deploy-assets-local-transfer")
	if err := os.MkdirAll(intDirPath, 0700); err != nil {
		return err
	}
	defer os.RemoveAll(intDirPath)

	intFilePath := filepath.Join(intDirPath, filepath.Base(srcPath))
	if _, _, err := src.ExecuteCommand("cp", srcPath, intFilePath); err != nil {
		return err
	}

	if _, _, err := dst.ExecuteCommand("cp", intFilePath, dstPath); err != nil {
		return err
	}
	return nil
}
