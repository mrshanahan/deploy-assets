package provider

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/mrshanahan/deploy-assets/internal/config"
	"github.com/mrshanahan/deploy-assets/internal/util"
	"github.com/pkg/errors"
)

func NewDockerProvider(name string, repositories ...string) config.Provider {
	return &dockerProvider{
		name:         name,
		repositories: repositories,
	}
}

type dockerProvider struct {
	name         string
	repositories []string
}

func (p *dockerProvider) Name() string { return p.name }

// TODO: Clean up old temp folders (?)
func (p *dockerProvider) Sync(config config.SyncConfig) error {
	srcEntries, err := loadDockerImageEntries(config.SrcExecutor, p.repositories)
	if err != nil {
		return err
	}

	dstEntries, err := loadDockerImageEntries(config.DstExecutor, p.repositories)
	if err != nil {
		return err
	}

	entriesToTransfer := getEntriesToTransfer(srcEntries, dstEntries)
	if len(entriesToTransfer) == 0 {
		slog.Info("no images to transfer", "src", config.SrcExecutor.Name(), "dst", config.DstExecutor.Name())
		return nil
	}

	if config.DryRun {
		slog.Info("DRY RUN: copying images", "src", config.SrcExecutor.Name(), "dst", config.DstExecutor.Name())
		for _, e := range entriesToTransfer {
			slog.Info("DRY RUN: copy", "image", e)
		}
		return nil
	}

	tempPath := util.GetTempFilePath("deploy-assets-docker")
	if _, _, err := config.SrcExecutor.ExecuteCommand("mkdir", "-p", tempPath); err != nil {
		slog.Error("could not create src temp directory", "dir", tempPath, "err", err)
		return err
	}
	defer config.SrcExecutor.ExecuteCommand("rm", "-rf", tempPath)

	srcName := config.SrcExecutor.Name()
	dstName := config.DstExecutor.Name()

	slog.Info("syncing docker images", "src", srcName, "dst", dstName)

	// TODO: Cache comparisons, and then don't need to re-export each time for multiple dsts
	for _, e := range entriesToTransfer {
		// docker save "$I" -o "./$FILENAME"
		fileName := strings.Replace(e.Repository, "/", "_", -1) + ".tar.gz"
		filePath := filepath.Join(tempPath, fileName)
		config.SrcExecutor.ExecuteCommand("docker", "save", e.Repository, "-o", filePath)

		if _, _, err := config.DstExecutor.ExecuteCommand("mkdir", "-p", tempPath); err != nil {
			slog.Error("could not create dst temp directory", "dst", dstName, "dir", tempPath, "err", err)
			return err
		}
		defer config.DstExecutor.ExecuteCommand("rm", "-rf", tempPath)

		if err := config.Transport.TransferFile(config.SrcExecutor, filePath, config.DstExecutor, filePath); err != nil {
			slog.Error("failed to transfer file", "dst", dstName, "file", filePath, "err", err)
			return err
		}

		if _, _, err := config.DstExecutor.ExecuteShell(fmt.Sprintf("cat %s | sudo docker load", filePath)); err != nil {
			slog.Error("failed to load image on remote", "dst", dstName, "file", filePath, "image", e.Repository, "err", err)
			return err
		}
	}

	return nil
}

func getEntriesToTransfer(src, dst map[string]*dockerImageEntry) []*dockerImageEntry {
	entries := []*dockerImageEntry{}
	for k, srce := range src {
		dste, existse := dst[k]
		if !existse || srce.ID != dste.ID { // && srce.CreatedAt.After(dste.CreatedAt)) {
			entries = append(entries, srce)
		}
	}
	return entries
}

// TODO: figure out digests - currently none of the images have digests
func loadDockerImageEntries(executor config.Executor, repositories []string) (map[string]*dockerImageEntry, error) {
	dockerArgs := []string{"image", "ls", "--format", "{{ .Repository }},{{ .ID }},{{ .CreatedAt }}"}
	for _, r := range repositories {
		dockerArgs = append(dockerArgs, "--filter")
		dockerArgs = append(dockerArgs, fmt.Sprintf("reference=%s", r))
	}
	stdout, _, err := executor.ExecuteCommand("docker", dockerArgs...)
	if err != nil {
		return nil, err
	}

	entries, err := parseDockerImageEntries(stdout)
	if err != nil {
		return nil, err
	}

	entriesMap := make(map[string]*dockerImageEntry)
	for _, e := range entries {
		entriesMap[e.Repository] = e
	}

	return entriesMap, nil
}

type dockerImageEntry struct {
	Repository string
	ID         string
	CreatedAt  time.Time
}

func parseDockerImageEntries(output string) ([]*dockerImageEntry, error) {
	var entries []*dockerImageEntry
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i, l := range lines {
		comps := strings.Split(l, ",")
		if len(comps) != 3 {
			return nil, errors.Errorf("Error in `docker image ls` output line %d (wrong number of CSV components): %s", i+1, l)
		}
		createdAt, err := time.Parse("2006-01-02 15:04:05 -0700 MST", comps[2])
		if err != nil {
			return nil, errors.Errorf("Error in `docker image ls` output line %d (invalid date: %v): %s", i+1, err, l)
		}
		entries = append(entries, &dockerImageEntry{
			Repository: comps[0],
			ID:         comps[1],
			CreatedAt:  createdAt,
		})
	}
	return entries, nil
}
