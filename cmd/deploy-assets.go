package main

import (
	"flag"
	"io"
	"log/slog"
	"os"

	"github.com/mrshanahan/deploy-assets/internal/config"
	"github.com/mrshanahan/deploy-assets/internal/manifest"
	"github.com/mrshanahan/deploy-assets/internal/util"
)

func main() {
	var manifestParam *string = flag.String("manifest", "", "local manifest to use for deployment")
	var debugParam *bool = flag.Bool("debug", false, "Enables debug logging")
	var dryRunParam *bool = flag.Bool("dry-run", false, "Performs a dry run (no actual copies)")
	var continueOnErrorParam *bool = flag.Bool("continue-on-error", false, "If a particular asset fails, continue with remaining")
	flag.Parse()

	if *manifestParam == "" {
		slog.Error("-manifest param required")
		os.Exit(1)
	}
	if *debugParam {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	manifestFile, err := os.Open(*manifestParam)
	if err != nil {
		slog.Error("failed to open manifest file", "path", *manifestParam, "err", err)
		os.Exit(1)
	}
	manifestBytes, err := io.ReadAll(manifestFile)
	if err != nil {
		slog.Error("failed to read manifest file", "path", *manifestParam, "err", err)
		os.Exit(1)
	}

	parsedManifest, err := manifest.ParseManifest(manifestBytes)
	if err != nil {
		slog.Error("failed to parse manifest file", "path", *manifestParam, "err", err)
		os.Exit(1)
	}

	manifest, err := manifest.BuildManifest(parsedManifest)
	if err != nil {
		slog.Error("failed to configure application from manifest", "path", *manifestParam, "err", err)
		os.Exit(1)
	}

	for _, e := range util.Values(manifest.Executors) {
		defer e.Close()
	}

	for _, providerConfig := range manifest.Providers {
		src, dst := providerConfig.Src, providerConfig.Dst
		srcExecutor := manifest.Executors[src]
		var dstExecutors []config.Executor
		if dst == "*" {
			dstExecutors = util.Filter(util.Values(manifest.Executors), func(e config.Executor) bool { return e.Name() != src })
		} else {
			dstExecutors = []config.Executor{manifest.Executors[dst]}
		}

		if err := manifest.Transport.Validate(srcExecutor); err != nil {
			slog.Error("failed to validate transport accessibility from source",
				"src", src,
				"err", err)
			os.Exit(1)
		}
		for _, dstExecutor := range dstExecutors {
			if err := manifest.Transport.Validate(dstExecutor); err != nil {
				slog.Error("failed to validate transport accessibility from destination",
					"dst", dstExecutor.Name(),
					"err", err)
				if !*continueOnErrorParam {
					os.Exit(1)
				} else {
					slog.Warn("continuing with remaining destinations despite error")
				}
			}

			synced, err := providerConfig.Provider.Sync(config.SyncConfig{
				SrcExecutor: srcExecutor,
				DstExecutor: dstExecutor,
				Transport:   manifest.Transport,
				DryRun:      *dryRunParam,
			})
			if err != nil {
				slog.Error("failed to sync asset",
					"asset", providerConfig.Provider.Name(),
					"src", srcExecutor.Name(),
					"dst", dstExecutor.Name(),
					"err", err)
				if !*continueOnErrorParam {
					os.Exit(1)
				} else {
					slog.Warn("continuing with remaining destinations despite error")
				}
			}

			for _, postCommand := range providerConfig.PostCommands {
				if !*dryRunParam || postCommand.Trigger == "always" || (postCommand.Trigger == "on_changed" && synced) {
					slog.Info("executing post-command",
						"command", postCommand.Command,
						"trigger", postCommand.Trigger,
						"synced", synced,
						"asset", providerConfig.Provider.Name(),
						"src", srcExecutor.Name(),
						"dst", dstExecutor.Name())
					stdout, stderr, err := dstExecutor.ExecuteShell(postCommand.Command)
					if err != nil {
						slog.Error("failed to execute post-command",
							"asset", providerConfig.Provider.Name(),
							"src", srcExecutor.Name(),
							"dst", dstExecutor.Name(),
							"err", err,
							"stdout", stdout,
							"stderr", stderr)
						if !*continueOnErrorParam {
							os.Exit(1)
						} else {
							slog.Warn("continuing with remaining destinations despite error")
						}
					}
				} else {
					slog.Debug("skipping post-command execution",
						"command", postCommand.Command,
						"trigger", postCommand.Trigger,
						"synced", synced,
						"asset", providerConfig.Provider.Name(),
						"src", srcExecutor.Name(),
						"dst", dstExecutor.Name())
				}
			}
		}
	}
}

// func tryGetManifestPath() (string, error) {
// 	cwd, err := os.Getwd()
// 	if err != nil {
// 		return "", err
// 	}

// 	dirEntries, err := os.ReadDir(cwd)
// 	if err != nil {
// 		return "", err
// 	}

// 	explicitManifestEntries :=
// }
