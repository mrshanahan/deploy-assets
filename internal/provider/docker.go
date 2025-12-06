package provider

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mrshanahan/deploy-assets/internal/config"
	"github.com/mrshanahan/deploy-assets/internal/util"
)

func NewDockerProvider(name string, repositories []string, compareLabel string) config.Provider {
	return &dockerProvider{
		name:         name,
		repositories: repositories,
		compareLabel: compareLabel,
	}
}

type dockerProvider struct {
	name         string
	repositories []string
	compareLabel string
}

func (p *dockerProvider) Name() string { return p.name }

// TODO: Clean up old temp folders (?)
func (p *dockerProvider) Sync(cfg config.SyncConfig) (config.SyncResult, error) {
	srcEntries, err := loadDockerImageEntries(cfg.SrcExecutor, p.repositories, p.compareLabel)
	if err != nil {
		return config.SYNC_RESULT_NOCHANGE, err
	}

	dstEntries, err := loadDockerImageEntries(cfg.DstExecutor, p.repositories, p.compareLabel)
	if err != nil {
		return config.SYNC_RESULT_NOCHANGE, err
	}

	entriesToTransfer, changeType := getEntriesToTransfer(srcEntries, dstEntries)
	if len(entriesToTransfer) == 0 {
		slog.Info("no images to transfer", "name", p.Name(), "src", cfg.SrcExecutor.Name(), "dst", cfg.DstExecutor.Name())
		return config.SYNC_RESULT_NOCHANGE, nil
	}

	if cfg.DryRun {
		slog.Info("DRY RUN: copying images", "src", cfg.SrcExecutor.Name(), "dst", cfg.DstExecutor.Name())
		for _, e := range entriesToTransfer {
			slog.Info("DRY RUN: copy", "image", e)
		}
		return changeType, nil
	}

	tempPath := util.GetTempFilePath("deploy-assets-docker")
	if _, _, err := cfg.SrcExecutor.ExecuteCommand("mkdir", "-p", tempPath); err != nil {
		slog.Error("could not create src temp directory", "dir", tempPath, "err", err)
		return config.SYNC_RESULT_NOCHANGE, err
	}
	defer cfg.SrcExecutor.ExecuteCommand("rm", "-rf", tempPath)

	srcName := cfg.SrcExecutor.Name()
	dstName := cfg.DstExecutor.Name()

	slog.Info("syncing docker images", "src", srcName, "dst", dstName)

	// TODO: Cache comparisons, and then don't need to re-export each time for multiple dsts
	// TODO: Sub-logger with src/dst/image
	for _, e := range entriesToTransfer {
		// docker save "$I" -o "./$FILENAME"
		fileName := strings.Replace(e.Repository, "/", "_", -1) + ".tar.gz"
		filePath := filepath.Join(tempPath, fileName)

		if _, stderr, err := cfg.SrcExecutor.ExecuteCommand("docker", "save", e.Repository, "-o", filePath); err != nil {
			slog.Error("failed to export image", "src", srcName, "dst", dstName, "image", e.Repository, "stderr", stderr, "err", err)
		}

		fileSize := ""
		stdout, _, err := cfg.SrcExecutor.ExecuteCommand("stat", "-c", "%s", filePath)
		if err != nil {
			slog.Warn("failed to get file size; continuing without it", "src", srcName, "dst", dstName, "image", e.Repository, "err", err)
		} else {
			fileSizeBytes, err := strconv.ParseInt(strings.TrimSpace(stdout), 10, 64)
			if err != nil {
				slog.Warn("failed to parse file size; continuing without it", "src", srcName, "dst", dstName, "image", e.Repository, "err", err)
			} else {
				fileSize = util.HumanReadableSize(fileSizeBytes)
			}
		}

		if _, _, err := cfg.DstExecutor.ExecuteCommand("mkdir", "-p", tempPath); err != nil {
			slog.Error("could not create dst temp directory", "dst", dstName, "dir", tempPath, "err", err)
			return config.SYNC_RESULT_NOCHANGE, err
		}
		defer cfg.DstExecutor.ExecuteCommand("rm", "-rf", tempPath)

		slog.Info("transferring image",
			"src", srcName,
			"dst", dstName,
			"image", e.Repository,
			"file-size", fileSize)

		if err := cfg.Transport.TransferFile(cfg.SrcExecutor, filePath, cfg.DstExecutor, filePath); err != nil {
			slog.Error("failed to transfer file", "dst", dstName, "file", filePath, "err", err)
			return config.SYNC_RESULT_NOCHANGE, err
		}

		if _, stderr, err := cfg.DstExecutor.ExecuteShell(fmt.Sprintf("cat %s | sudo docker load", filePath)); err != nil {
			slog.Error("failed to load image on remote", "dst", dstName, "file", filePath, "image", e.Repository, "stderr", stderr, "err", err)
			return config.SYNC_RESULT_NOCHANGE, err
		}
	}

	return changeType, nil
}

func getEntriesToTransfer(src, dst map[string]*dockerImageEntry) ([]*dockerImageEntry, config.SyncResult) {
	entries := []*dockerImageEntry{}
	changeType := config.SYNC_RESULT_NOCHANGE
	for k, srce := range src {
		dste, existse := dst[k]
		if !existse || srce.CompareValue != dste.CompareValue { // && srce.CreatedAt.After(dste.CreatedAt)) {
			if !existse {
				changeType = config.SYNC_RESULT_CREATED
			} else if changeType == config.SYNC_RESULT_NOCHANGE {
				changeType = config.SYNC_RESULT_UPDATED
			}
			slog.Debug("found image entry to transfer",
				"src", srce,
				"dst", dste)
			entries = append(entries, srce)
		}
	}
	return entries, changeType
}

// TODO: figure out digests - currently none of the images have digests
func loadDockerImageEntries(executor config.Executor, repositories []string, compareLabel string) (map[string]*dockerImageEntry, error) {
	var compareLabelFormat string
	if compareLabel != "" {
		// TODO: Make sure funky stuff can't happen here with a carefully-crafted label
		compareLabelFormat = fmt.Sprintf("{{ index .Config.Labels \"%s\" }}", compareLabel)
	} else {
		compareLabelFormat = "{{ \"\" }}"
	}

	dockerInspectFormat := fmt.Sprintf("{{ index .RepoTags 0 }},{{ .ID }},{{ .Created }},%s", compareLabelFormat)
	dockerArgs := []string{"image", "inspect", "--format", dockerInspectFormat}
	dockerArgs = append(dockerArgs, repositories...)

	stdout, _, err := executor.ExecuteCommand("docker", dockerArgs...)
	if err != nil {
		return nil, err
	}

	entries, err := parseDockerImageEntries(stdout)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.CompareValue == "" {
			if compareLabel != "" {
				slog.Warn("could not find compare_label label on image; defaulting to id",
					"location", executor.Name(),
					"image-repository", e.Repository,
					"image-id", e.ID,
					"label", compareLabel)
			}
			e.CompareValue = fmt.Sprintf("@id:%s", e.ID)
		} else {
			// If label is e.g. "foo.bar" and label value is "baz", then final CompareValue is "foo.bar:baz"
			e.CompareValue = fmt.Sprintf("%s:%s", compareLabel, e.CompareValue)
		}
	}

	entriesMap := make(map[string]*dockerImageEntry)
	for _, e := range entries {
		entriesMap[e.Repository] = e
	}

	return entriesMap, nil
}

type dockerImageEntry struct {
	Repository   string
	ID           string
	CreatedAt    time.Time
	CompareValue string
}

func tryParseTimes(layouts []string, value string) (time.Time, error) {
	errs := []error{}
	var parsed *time.Time
	for _, l := range layouts {
		if t, err := time.Parse(l, value); err != nil {
			errs = append(errs, err)
		} else {
			parsed = &t
			break
		}
	}

	if parsed != nil {
		return *parsed, nil
	}
	return time.Time{}, errors.Join(errs...)
}

func parseDockerImageEntries(output string) ([]*dockerImageEntry, error) {
	var entries []*dockerImageEntry
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i, l := range lines {
		comps := strings.Split(l, ",")
		if len(comps) != 4 {
			return nil, fmt.Errorf("error in `docker image ls` output line %d (wrong number of CSV components): %s", i+1, l)
		}
		createdAt, err := tryParseTimes(
			[]string{
				// Docker image timestamps appear to have an imprecise number of zeroes. :/
				"2006-01-02T15:04:05.000000000-07:00",
				"2006-01-02T15:04:05.00000000-07:00",
				"2006-01-02T15:04:05.0000000-07:00",
				"2006-01-02T15:04:05.000000-07:00",
				"2006-01-02T15:04:05.000000000Z"},
			comps[2])
		if err != nil {
			return nil, fmt.Errorf("error in `docker image ls` output line %d (invalid date: %v): %s", i+1, err, l)
		}
		entries = append(entries, &dockerImageEntry{
			Repository:   comps[0],
			ID:           comps[1],
			CreatedAt:    createdAt,
			CompareValue: comps[3],
		})
	}
	return entries, nil
}
