package config

type Executor interface {
	Name() string
	Yaml() string
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
	Yaml() string
	Sync(config SyncConfig) (SyncResult, error)
}

type ProviderConfig struct {
	Provider     Provider
	Src          string
	Dst          string
	PostCommands []*PostCommand
}

type PostCommand struct {
	Command string
	Trigger string
}

type Transport interface {
	Validate(exec Executor) error
	Yaml() string
	TransferFile(src Executor, srcPath string, dst Executor, dstPath string) error
}
