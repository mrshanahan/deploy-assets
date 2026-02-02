package runner

import (
	"fmt"
	"log/slog"

	"github.com/mrshanahan/deploy-assets/internal/config"
	"github.com/mrshanahan/deploy-assets/internal/manifest"
	"github.com/mrshanahan/deploy-assets/internal/util"
)

func Execute(m *manifest.Manifest, dryRun bool, continueOnError bool) error {
	for _, e := range util.Values(m.Executors) {
		defer e.Close()
	}

	for _, providerConfig := range m.Providers {
		src, dst := providerConfig.Src, providerConfig.Dst
		srcExecutor := m.Executors[src]
		var dstExecutors []config.Executor
		if dst == "*" {
			dstExecutors = util.Filter(util.Values(m.Executors), func(e config.Executor) bool { return e.Name() != src })
		} else {
			dstExecutors = []config.Executor{m.Executors[dst]}
		}

		if err := m.Transport.Validate(srcExecutor); err != nil {
			return fmt.Errorf("failed to validate transport accessibility from source %s: %w", src, err)
		}
		for _, dstExecutor := range dstExecutors {
			if err := m.Transport.Validate(dstExecutor); err != nil {
				if !continueOnError {
					return fmt.Errorf("failed to validate transport accessibility from destination %s: %w",
						dstExecutor.Name(), err)
				} else {
					slog.Warn("failed to validate transport accessibility from destination; continuing with remaining destinations despite error",
						"dst", dstExecutor.Name(),
						"err", err)
				}
			}

			syncResult, err := providerConfig.Provider.Sync(config.SyncConfig{
				SrcExecutor: srcExecutor,
				DstExecutor: dstExecutor,
				Transport:   m.Transport,
				DryRun:      dryRun,
			})
			if err != nil {
				if !continueOnError {
					return fmt.Errorf("failed to sync asset %s (%s -> %s): %w",
						providerConfig.Provider.Name(),
						srcExecutor.Name(),
						dstExecutor.Name(),
						err)
				} else {
					slog.Warn("failed to sync asset; continuing with remaining destinations despite error",
						"asset", providerConfig.Provider.Name(),
						"src", srcExecutor.Name(),
						"dst", dstExecutor.Name(),
						"err", err)
				}
			}

			for _, postCommand := range providerConfig.PostCommands {
				if postCommand.Trigger == "always" ||
					(syncResult != config.SYNC_RESULT_NOCHANGE && postCommand.Trigger == "on_changed") ||
					(syncResult == config.SYNC_RESULT_CREATED && postCommand.Trigger == "on_created") ||
					(syncResult == config.SYNC_RESULT_UPDATED && postCommand.Trigger == "on_updated") {

					if !dryRun {
						slog.Info("executing post-command",
							"command", postCommand.Command,
							"trigger", postCommand.Trigger,
							"synced", syncResult,
							"asset", providerConfig.Provider.Name(),
							"src", srcExecutor.Name(),
							"dst", dstExecutor.Name())
						stdout, stderr, err := dstExecutor.ExecuteShell(postCommand.Command)
						if err != nil {
							if !continueOnError {
								return fmt.Errorf("failed to execute post-command on %s (%s -> %s) (stdout: %s) (stderr: %s): %w",
									providerConfig.Provider.Name(),
									srcExecutor.Name(),
									dstExecutor.Name(),
									stdout,
									stderr,
									err)
							} else {
								slog.Warn("failed to execute post-command; continuing with remaining destinations despite error",
									"asset", providerConfig.Provider.Name(),
									"src", srcExecutor.Name(),
									"dst", dstExecutor.Name(),
									"err", err,
									"stdout", stdout,
									"stderr", stderr)
							}
						}
					} else {
						slog.Info("DRY RUN: executing post-command",
							"command", postCommand.Command,
							"trigger", postCommand.Trigger,
							"synced", syncResult,
							"asset", providerConfig.Provider.Name(),
							"src", srcExecutor.Name(),
							"dst", dstExecutor.Name())
					}
				} else {
					slog.Debug("skipping post-command execution",
						"command", postCommand.Command,
						"trigger", postCommand.Trigger,
						"synced", syncResult,
						"asset", providerConfig.Provider.Name(),
						"src", srcExecutor.Name(),
						"dst", dstExecutor.Name())
				}
			}
		}
	}

	return nil
}
