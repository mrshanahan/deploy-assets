package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/mrshanahan/deploy-assets/internal/config"
	"github.com/mrshanahan/deploy-assets/internal/executor"
	"github.com/mrshanahan/deploy-assets/internal/provider"
	"github.com/mrshanahan/deploy-assets/internal/transport"
)

func main() {
	var serverParam *string = flag.String("server", "", "remote server to connect to")
	var userParam *string = flag.String("user", "", "user to authenticate with")
	var keyParam *string = flag.String("key-path", "", "path to the private key")
	var debugParam *bool = flag.Bool("debug", false, "Enables debug logging")
	var dryRunParam *bool = flag.Bool("dry-run", false, "Performs a dry run (no actual copies)")
	flag.Parse()

	if *serverParam == "" {
		slog.Error("-server param required")
		os.Exit(1)
	}
	if *userParam == "" {
		slog.Error("-user param required")
		os.Exit(1)
	}
	if *keyParam == "" {
		slog.Error("-key-path param required")
		os.Exit(1)
	}
	if *debugParam {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	localExec := executor.NewLocalExecutor("local")
	defer localExec.Close()

	sshExec, err := executor.NewSSHExecutor("quemot.dev", *serverParam, *userParam, *keyParam, true)
	if err != nil {
		slog.Error("failed to create SSH executor", "err", err)
		os.Exit(1)
	}
	defer sshExec.Close()

	transport := transport.NewS3Transport("s3://quemot-dev-bucket/deploy-assets")
	// dirProvider := provider.NewDirProvider("/home/matt/repos/notes-service/package-files", "/home/ubuntu/package")
	// if err = dirProvider.Sync(config.SyncConfig{
	dockerProvider := provider.NewDockerProvider(
		"notes-api/auth",
		"notes-api/auth-db",
		"notes-api/auth-cli",
		"notes-api/api",
		"notes-api/web",
	)
	if err = dockerProvider.Sync(config.SyncConfig{
		SrcExecutor: localExec,
		DstExecutor: sshExec,
		Transport:   transport,
		DryRun:      *dryRunParam,
	}); err != nil {
		slog.Error("failed to run sync", "err", err)
	}
}
