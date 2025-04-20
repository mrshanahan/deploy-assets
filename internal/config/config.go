package config

type Executor interface {
	Name() string
	ExecuteCommand(name string, args ...string) (string, string, error)
	ExecuteShell(cmd string) (string, string, error)
	Close()
}

type SyncConfig struct {
	SrcExecutor Executor
	DstExecutor Executor
	Transport   Transport
	DryRun      bool
}

type Provider interface {
	Name() string
	Sync(config SyncConfig) error
}

type ProviderConfig struct {
	Provider    Provider
	Src         string
	Dst         string
	PostCommand string
}

type Transport interface {
	Validate(exec Executor) error
	TransferFile(src Executor, srcPath string, dst Executor, dstPath string) error
}
