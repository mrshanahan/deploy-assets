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

type targetFileEntry struct {
	path      string
	fileEntry *fileEntry
}

type fileEntry struct {
	path         string
	relativePath string
	modifiedAt   time.Time
}

type mappedFileEntry struct {
	Src *fileEntry
	Dst *targetFileEntry
}

type fileInfo struct {
	FullPath    string
	IsDirectory bool
	Exists      bool
	DirExists   bool
}

// TODO: This is all fucked up. There shouldn't be all this random branching for dir/non-dir & we should just
// treat it as a collection of absolute paths mapped from one to the other. Fix this!

func loadFileEntries(finfo *fileInfo, executor config.Executor, recursive bool) (map[string]*fileEntry, error) {
	server := executor.Name()
	entries := make(map[string]*fileEntry)

	var dirPath string
	if finfo.IsDirectory {
		dirPath = finfo.FullPath
	} else {
		// TODO: This is a little sus. Technically we're expecting all executors to be able to run POSIX-ish commands,
		// but this is explicitly using the local filepath lib to parse out a remote filepath, which seems bad. Just
		// seems a bit silly to
		dirPath = filepath.Dir(finfo.FullPath)
	}
	dirPath = strings.TrimRight(dirPath, "/") + "/"

	maxDepthArg := ""
	if !recursive {
		maxDepthArg = "-maxdepth 1 "
	}
	cmd := fmt.Sprintf("find \"%s\" -type f %s-exec ls -l --time-style=+%%s '{}' \\; | sed -E 's/ +/ /g' | cut -d ' ' -f6-", finfo.FullPath, maxDepthArg)
	slog.Debug("executing file discovery", "server", server, "cmd", cmd)
	stdout, stderr, err := executor.ExecuteShell(cmd)
	if err != nil {
		slog.Error("failed to perform file discovery", "server", server, "stdout", stdout, "stderr", stderr, "err", err)
		return nil, err
	}
	files := strings.Split(strings.TrimSpace(stdout), "\n")
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
		relativePath := strings.Replace(filePath, dirPath, "", -1)
		entries[relativePath] = &fileEntry{filePath, relativePath, time.Unix(timestamp, 0)}
		slog.Debug("file entry",
			"server", server,
			"relative-path", entries[relativePath].relativePath,
			"full-path", entries[relativePath].path,
			"modified-at", entries[relativePath].modifiedAt.UTC().Format(time.RFC3339))
	}
	return entries, nil
}

func getFileInfo(path string, executor config.Executor) (*fileInfo, error) {
	server := executor.Name()

	// TODO: Paths ending in a return/newline will be incorrect after trim. I _hope_ we don't have to worry about this.
	canonPath, stderr, err := executor.ExecuteShell(fmt.Sprintf("realpath -m \"%s\"", path))
	if err != nil {
		slog.Error("failed to canonicalize path", "stderr", stderr, "err", err)
		return nil, err
	}
	canonPath = strings.TrimRight(canonPath, "\r\n")

	dirName := filepath.Dir(canonPath)

	stdout, stderr, err := executor.ExecuteShell(fmt.Sprintf("test -e \"%s\"", dirName))
	if err != nil && stderr == "" {
		return &fileInfo{
			FullPath:    canonPath,
			IsDirectory: false,
			Exists:      false,
			DirExists:   false,
		}, nil
	} else if err != nil {
		slog.Error("failed to check for file path parent existence", "server", server, "path", dirName, "stdout", stdout, "stderr", stderr, "err", err)
		return nil, err
	}

	stdout, stderr, err = executor.ExecuteShell(fmt.Sprintf("test -e \"%s\"", canonPath))
	if err != nil && stderr == "" {
		return &fileInfo{
			FullPath:    canonPath,
			IsDirectory: false,
			Exists:      false,
			DirExists:   true,
		}, nil
	} else if err != nil {
		slog.Error("failed to check for file path existence", "server", server, "path", canonPath, "stdout", stdout, "stderr", stderr, "err", err)
		return nil, err
	}

	fileType, stderr, err := executor.ExecuteShell(fmt.Sprintf("stat \"%s\" -c %%F", canonPath))
	if err != nil {
		slog.Error("failed to get file type", "server", server, "path", canonPath, "stdout", fileType, "stderr", stderr, "err", err)
		return nil, err
	}
	fileType = strings.TrimSpace(fileType)

	return &fileInfo{
		FullPath:    canonPath,
		IsDirectory: fileType == "directory",
		Exists:      true,
		DirExists:   true,
	}, nil
}

func (p *fileProvider) Name() string { return p.name }

// TODO: Combine tmp file usage, both in code & on system
func (p *fileProvider) Sync(config config.SyncConfig) (bool, error) {
	srcFileInfo, err := getFileInfo(p.srcPath, config.SrcExecutor)
	if err != nil {
		return false, err
	}

	if !srcFileInfo.Exists {
		return false, fmt.Errorf("src file is missing")
	}

	dstFileInfo, err := getFileInfo(p.dstPath, config.DstExecutor)
	if err != nil {
		return false, err
	}

	var dstEntries map[string]*fileEntry
	if dstFileInfo.Exists {
		if dstFileInfo.IsDirectory != srcFileInfo.IsDirectory {
			return false, fmt.Errorf("mismatch in file type")
		}

		// NB: This should work the same way whether or not the source
		// is a file or a directory.
		dstEntries, err = loadFileEntries(dstFileInfo, config.DstExecutor, p.recursive)
		if err != nil {
			return false, err
		}

	} else {
		if !dstFileInfo.DirExists {
			return false, fmt.Errorf("target base directory does not exist")
		}

		dstEntries = make(map[string]*fileEntry)
	}

	srcEntries, err := loadFileEntries(srcFileInfo, config.SrcExecutor, p.recursive)
	if err != nil {
		return false, err
	}

	entriesToTransfer := compareFilesForTransfer(srcEntries, dstEntries, srcFileInfo, dstFileInfo)

	if len(entriesToTransfer) == 0 {
		slog.Info("no files to transfer", "name", p.Name(), "src", config.SrcExecutor.Name(), "dst", config.DstExecutor.Name())
		return false, nil
	}

	if config.DryRun {
		slog.Info("DRY RUN: copying files", "name", p.Name(), "src", config.SrcExecutor.Name(), "dst", config.DstExecutor.Name(), "num-files", len(entriesToTransfer))
		for _, e := range entriesToTransfer {
			var dstModifiedAt, dstPath any
			srcPath, srcModifiedAt := e.Src.path, e.Src.modifiedAt.Format(time.RFC3339)
			if e.Dst.fileEntry != nil {
				dstPath, dstModifiedAt = e.Dst.path, e.Dst.fileEntry.modifiedAt.Format(time.RFC3339)
			} else {
				dstPath, dstModifiedAt = nil, nil
			}
			slog.Info("DRY RUN: copy",
				"src-path", srcPath, "src-modified-at", srcModifiedAt,
				"dst-path", dstPath, "dst-modified-at", dstModifiedAt)
		}

		return false, nil
	}

	tempFolderPath := util.GetTempFilePath("deploy-assets-file")
	tempPackageFolderName := "package"
	tempPackageFolderPath := filepath.Join(tempFolderPath, tempPackageFolderName)
	if _, _, err := config.SrcExecutor.ExecuteCommand("mkdir", "-p", tempPackageFolderPath); err != nil {
		slog.Error("could not create src temp directory", "dir", tempPackageFolderPath, "err", err)
		return false, err
	}
	defer config.SrcExecutor.ExecuteCommand("rm", "-rf", tempFolderPath)

	srcServerName := config.SrcExecutor.Name()
	dstServerName := config.DstExecutor.Name()

	slog.Info("syncing files", "name", p.Name(), "src", srcServerName, "dst", dstServerName, "num-files", len(entriesToTransfer))
	for _, mapped := range entriesToTransfer {
		src := mapped.Src
		dir := filepath.Dir(src.relativePath)
		targetDir := filepath.Join(tempPackageFolderPath, dir)
		_, _, err := config.SrcExecutor.ExecuteCommand("mkdir", "-p", targetDir)
		if err != nil {
			return false, err
		}

		_, _, err = config.SrcExecutor.ExecuteCommand("cp", "-a", src.path, targetDir)
		if err != nil {
			return false, err
		}
	}

	tempPackageName := tempPackageFolderName + ".tar"
	tempPackagePath := filepath.Join(tempFolderPath, tempPackageName)
	if _, _, err := config.SrcExecutor.ExecuteCommand("tar", "cvf", tempPackagePath, "-C", tempFolderPath, tempPackageFolderName); err != nil {
		return false, err
	}

	if _, _, err := config.SrcExecutor.ExecuteCommand("gzip", tempPackagePath); err != nil {
		return false, err
	}
	compressedPackagePath := tempPackagePath + ".gz"

	if _, _, err := config.DstExecutor.ExecuteCommand("mkdir", "-p", tempFolderPath); err != nil {
		slog.Error("could not create dst temp directory", "dst", dstServerName, "dir", tempFolderPath, "err", err)
		return false, err
	}
	defer config.DstExecutor.ExecuteCommand("rm", "-rf", tempFolderPath)

	if err := config.Transport.TransferFile(config.SrcExecutor, compressedPackagePath, config.DstExecutor, compressedPackagePath); err != nil {
		return false, err
	}

	if _, _, err := config.DstExecutor.ExecuteCommand("gunzip", compressedPackagePath); err != nil {
		return false, err
	}

	if _, _, err := config.DstExecutor.ExecuteCommand("tar", "xvf", tempPackagePath, "-C", tempFolderPath); err != nil {
		return false, err
	}

	// If dir & dst doesn't exist: 	package 	-> dst
	// If dir & dst exists: 		package/* 	-> dst	(otherwise if dst exists then we would get dst/package)
	// If file:						package/src	-> dst

	var srcCopyPath string
	if srcFileInfo.IsDirectory && dstFileInfo.Exists {
		srcCopyPath = filepath.Join(tempPackageFolderPath, "*")
	} else if srcFileInfo.IsDirectory {
		srcCopyPath = tempPackageFolderPath
	} else {
		srcName := filepath.Base(p.srcPath)
		srcCopyPath = filepath.Join(tempPackageFolderPath, srcName)
	}

	if _, _, err := config.DstExecutor.ExecuteShell(fmt.Sprintf("cp -ar %s %s", srcCopyPath, p.dstPath)); err != nil {
		return false, err
	}

	return true, nil
}

func compareFilesForTransfer(src, dst map[string]*fileEntry, srcFileInfo, dstFileInfo *fileInfo) []*mappedFileEntry {
	entries := []*mappedFileEntry{}
	if srcFileInfo.IsDirectory {
		for k, srce := range src {
			dste, existse := dst[k]
			if !existse {
				dstTargetPath := filepath.Join(dstFileInfo.FullPath, srce.relativePath)
				entries = append(entries, &mappedFileEntry{
					Src: srce,
					Dst: &targetFileEntry{
						path:      dstTargetPath,
						fileEntry: nil,
					},
				})
				// } else if srce.modifiedAt.After(dste.modifiedAt) {
			} else if srce.modifiedAt != dste.modifiedAt {
				entries = append(entries, &mappedFileEntry{
					Src: srce,
					Dst: &targetFileEntry{
						path:      dste.path,
						fileEntry: dste,
					},
				})
			}
		}
	} else {
		if len(src) > 1 {
			panic("more than 1 source item for non-dir resource - not supported!")
		}
		dstk := filepath.Base(dstFileInfo.FullPath)
		dste, existse := dst[dstk]
		srce := util.Values(src)[0]
		if !existse {
			entries = append(entries, &mappedFileEntry{
				Src: srce,
				Dst: &targetFileEntry{
					path:      dstFileInfo.FullPath,
					fileEntry: nil,
				},
			})
		} else if srce.modifiedAt != dste.modifiedAt {
			entries = append(entries, &mappedFileEntry{
				Src: srce,
				Dst: &targetFileEntry{
					path:      dste.path,
					fileEntry: dste,
				},
			})
		}

	}

	return entries
}
