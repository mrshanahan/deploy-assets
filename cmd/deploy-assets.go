package main

import (
	"flag"
	"io"
	"log/slog"
	"os"

	"github.com/mrshanahan/deploy-assets/internal/manifest"
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

	if err := runner.Execute(manifest, *dryRunParam, *continueOnErrorParam); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
