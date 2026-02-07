package config

import (
	"fmt"
	"strings"

	"github.com/mrshanahan/deploy-assets/internal/util"
)

type Executor interface {
	Name() string
	Yaml(depth int) string
	ExecuteCommand(name string, args ...string) (string, string, error)
	ExecuteCommandInDir(workingDir string, name string, args ...string) (string, string, error)
	ExecuteShell(cmd string) (string, string, error)
	ExecuteShellInDir(workingDir string, cmd string) (string, string, error)
	Close()
}

type SyncConfig struct {
	SrcExecutor Executor
	DstExecutor Executor
	Transport   Transport
	DryRun      bool
}

type SyncResult int

const (
	SYNC_RESULT_NOCHANGE SyncResult = iota
	SYNC_RESULT_CREATED
	SYNC_RESULT_UPDATED
)

type Provider interface {
	Name() string
	Yaml(depth int) string
	Sync(config SyncConfig) (SyncResult, error)
}

type ProviderConfig struct {
	Provider     Provider
	Src          string
	Dst          string
	PostCommands []*PostCommand
}

func (c *ProviderConfig) Yaml(indent int) string {
	mainIndent := util.YamlIndentString(indent)
	subpropIndent := util.YamlIndentString(indent + 2)
	postCommandYamlLines := []string{}
	for _, c := range c.PostCommands {
		postCommandYamlLines = append(postCommandYamlLines, c.Yaml(indent+2+util.TabsToIndent(1)))
	}
	postCommandYaml := ""
	if len(postCommandYamlLines) > 0 {
		postCommandYaml = "\n" + strings.Join(postCommandYamlLines, "\n")
	}
	return fmt.Sprintf(
		`%s- src: %s
%sdst: %s
%sprovider:
%s
%spost_commands:%s`,
		mainIndent, c.Src,
		subpropIndent, c.Dst,
		subpropIndent, c.Provider.Yaml(indent+2+util.TabsToIndent(1)),
		subpropIndent, postCommandYaml)
}

type PostCommand struct {
	Command string
	Trigger string
}

func (c *PostCommand) Yaml(indent int) string {
	mainIndent := util.YamlIndentString(indent)
	subIndent := util.YamlIndentString(indent + 2)
	return fmt.Sprintf(
		`%s- command: "%s"
%strigger: %s`,
		mainIndent, c.Command,
		subIndent, c.Trigger)
}

type Transport interface {
	Validate(exec Executor) error
	Yaml(depth int) string
	TransferFile(src Executor, srcPath string, dst Executor, dstPath string) error
}
