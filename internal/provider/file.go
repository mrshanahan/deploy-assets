package provider

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mrshanahan/deploy-assets/internal/config"
	"github.com/mrshanahan/deploy-assets/internal/util"
)

func NewFileProvider(name, srcPath, dstPath string, recursive bool) config.Provider {
	return &fileProvider{name, srcPath, dstPath, recursive, make(map[string]*fileEntry), make(map[string]map[string]*fileEntry)}
}

type fileProvider struct {
	name       string
	srcPath    string
	dstPath    string
	recursive  bool
	srcEntries map[string]*fileEntry
	dstEntries map[string]map[string]*fileEntry
}

type fileEntry struct {
	path       string
	truncPath  string
	modifiedAt time.Time
}

func loadFileEntries(path string, executor config.Executor, recursive bool) (map[string]*fileEntry, error) {
	server := executor.Name()
	entries := make(map[string]*fileEntry)

	realPath, stderr, err := executor.ExecuteShell(fmt.Sprintf("realpath %s", path))
	if err != nil {
		slog.Error("failed to get real file path", "server", server, "path", path, "stdout", realPath, "stderr", stderr, "err", err)
		return nil, err
	}
	realPath = strings.TrimSpace(realPath)

	maxDepthArg := ""
	if !recursive {
		maxDepthArg = "-maxdepth 1 "
	}
	cmd := fmt.Sprintf("find %s -type f %s-exec ls -l --time-style=+%%s '{}' \\; | sed -E 's/ +/ /g' | cut -d ' ' -f6-", realPath, maxDepthArg)
	slog.Debug("executing file discovery", "server", server, "cmd", cmd)
	stdout, stderr, err := executor.ExecuteShell(cmd)
	if err != nil {
		slog.Error("failed to perform file discovery", "server", server, "stdout", stdout, "stderr", stderr, "err", err)
		return nil, err
	}
	files := strings.Split(stdout, "\n")
	slog.Debug("found files", "server", server, "num-files", len(files))
	for _, f := range files {
		if f == "" {
			continue
		}
		comps := strings.Split(f, " ")
		timestampStr := comps[0]
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			return nil, err
		}
		filePath := strings.Join(comps[1:], " ")
		truncPath := strings.Replace(filePath, realPath, "", -1)
		entries[truncPath] = &fileEntry{filePath, truncPath, time.Unix(timestamp, 0)}
		slog.Debug("file entry",
			"server", server,
			"trunc-path", entries[truncPath].truncPath,
			"full-path", entries[truncPath].path,
			"modified-at", entries[truncPath].modifiedAt.UTC().Format(time.RFC3339))
	}
	return entries, nil
}

func (p *fileProvider) Name() string { return p.name }

// TODO: Combine tmp file usage, both in code & on system
func (p *fileProvider) Sync(config config.SyncConfig) error {
	srcEntries, err := loadFileEntries(p.srcPath, config.SrcExecutor, p.recursive)
	if err != nil {
		return err
	}

	dstEntries, err := loadFileEntries(p.dstPath, config.DstExecutor, p.recursive)
	if err != nil {
		return err
	}

	entriesToTransfer := getFilesToTransfer(srcEntries, dstEntries)
	if len(entriesToTransfer) == 0 {
		slog.Info("no files to transfer", "src", config.SrcExecutor.Name(), "dst", config.DstExecutor.Name())
		return nil
	}

	if config.DryRun {
		slog.Info("DRY RUN: copying files", "src", config.SrcExecutor.Name(), "dst", config.DstExecutor.Name(), "num-files", len(entriesToTransfer))
		for _, e := range entriesToTransfer {
			var dstModifiedAtStr string
			dstEntry, prs := dstEntries[e.truncPath]
			if !prs {
				dstModifiedAtStr = ""
			} else {
				dstModifiedAtStr = dstEntry.modifiedAt.Format(time.RFC3339)
			}
			slog.Info("DRY RUN: copy",
				"trunc-path", e.truncPath,
				"src-modified-at", e.modifiedAt.Format(time.RFC3339),
				"dst-modified-at", dstModifiedAtStr)
		}

		return nil
	}

	tempFolderPath := util.GetTempFilePath("deploy-assets-file")
	tempPackageFolderName := "package"
	tempPackageFolderPath := filepath.Join(tempFolderPath, tempPackageFolderName)
	if _, _, err := config.SrcExecutor.ExecuteCommand("mkdir", "-p", tempPackageFolderPath); err != nil {
		slog.Error("could not create src temp directory", "dir", tempPackageFolderPath, "err", err)
		return err
	}
	defer config.SrcExecutor.ExecuteCommand("rm", "-rf", tempFolderPath)

	srcName := config.SrcExecutor.Name()
	dstName := config.DstExecutor.Name()

	slog.Info("syncing files", "src", srcName, "dst", dstName, "num-files", len(entriesToTransfer))
	for _, e := range entriesToTransfer {
		dir := filepath.Dir(e.truncPath)
		targetDir := filepath.Join(tempPackageFolderPath, dir)
		_, _, err := config.SrcExecutor.ExecuteCommand("mkdir", "-p", targetDir)
		if err != nil {
			return err
		}
		_, _, err = config.SrcExecutor.ExecuteCommand("cp", "-a", e.path, targetDir)
		if err != nil {
			return err
		}
	}

	tempPackageName := tempPackageFolderName + ".tar"
	tempPackagePath := filepath.Join(tempFolderPath, tempPackageName)
	if _, _, err := config.SrcExecutor.ExecuteCommand("tar", "cvf", tempPackagePath, "-C", tempFolderPath, tempPackageFolderName); err != nil {
		return err
	}

	if _, _, err := config.SrcExecutor.ExecuteCommand("gzip", tempPackagePath); err != nil {
		return err
	}
	compressedPackagePath := tempPackagePath + ".gz"

	if _, _, err := config.DstExecutor.ExecuteCommand("mkdir", "-p", tempFolderPath); err != nil {
		slog.Error("could not create dst temp directory", "dst", dstName, "dir", tempFolderPath, "err", err)
		return err
	}
	defer config.DstExecutor.ExecuteCommand("rm", "-rf", tempFolderPath)

	if err := config.Transport.TransferFile(config.SrcExecutor, compressedPackagePath, config.DstExecutor, compressedPackagePath); err != nil {
		return err
	}

	if _, _, err := config.DstExecutor.ExecuteCommand("gunzip", compressedPackagePath); err != nil {
		return err
	}

	if _, _, err := config.DstExecutor.ExecuteCommand("tar", "xvf", tempPackagePath, "-C", tempFolderPath); err != nil {
		return err
	}

	copyGlob := filepath.Join(tempPackageFolderPath, "*")
	if _, _, err := config.DstExecutor.ExecuteShell(fmt.Sprintf("cp -ar %s %s", copyGlob, p.dstPath)); err != nil {
		return err
	}
	return nil
}

func getFilesToTransfer(src, dst map[string]*fileEntry) []*fileEntry {
	entries := []*fileEntry{}
	for k, srce := range src {
		dste, existse := dst[k]
		if !existse || srce.modifiedAt != dste.modifiedAt {
			entries = append(entries, srce)
		}
	}
	return entries
}
