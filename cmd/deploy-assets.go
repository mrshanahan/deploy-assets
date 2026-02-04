package main

import (
	"flag"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mrshanahan/deploy-assets/pkg/manifest"
	"github.com/mrshanahan/deploy-assets/pkg/runner"
)

func main() {
	var manifestParam *string = flag.String("manifest", "", "local manifest to use for deployment")
	var debugParam *bool = flag.Bool("debug", false, "Enables debug logging")
	var dryRunParam *bool = flag.Bool("dry-run", false, "Performs a dry run (no actual copies)")
	var continueOnErrorParam *bool = flag.Bool("continue-on-error", false, "If a particular asset fails, continue with remaining")
	flag.Parse()

	if *debugParam {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if *manifestParam == "" {
		slog.Error("-manifest param required")
		os.Exit(1)
	}

	manifestFilePath := *manifestParam
	manifestFile, err := os.Open(manifestFilePath)
	if err != nil {
		slog.Error("failed to open manifest file", "path", manifestFilePath, "err", err)
		os.Exit(1)
	}
	manifestDirRel := filepath.Dir(manifestFilePath)
	manifestDir, err := filepath.Abs(manifestDirRel)
	if err != nil {
		slog.Error("failed to get absolute path for manifest file dir", "dir", manifestDirRel, "err", err)
		os.Exit(1)
	}

	manifestBytes, err := io.ReadAll(manifestFile)
	if err != nil {
		slog.Error("failed to read manifest file", "path", manifestFilePath, "err", err)
		os.Exit(1)
	}

	parsedManifest, err := manifest.ParseManifest(manifestBytes)
	if err != nil {
		slog.Error("failed to parse manifest file", "path", manifestFilePath, "err", err)
		os.Exit(1)
	}

	manifest, err := manifest.BuildManifest(manifestDir, parsedManifest)
	if err != nil {
		slog.Error("failed to configure application from manifest", "path", manifestFilePath, "err", err)
		os.Exit(1)
	}

	if err := runner.Execute(manifest, *dryRunParam, *continueOnErrorParam); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
