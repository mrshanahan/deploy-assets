package provider

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mrshanahan/deploy-assets/internal/executor"
	"github.com/mrshanahan/deploy-assets/internal/transport"
	"github.com/mrshanahan/deploy-assets/internal/util"
	"github.com/mrshanahan/deploy-assets/pkg/config"
)

func getCurrentFilePath() string {
	_, fname, _, ok := runtime.Caller(1)
	if !ok {
		return ""
	}
	return fname
}

type fileDef struct {
	name    string
	modTime string
	content string
}

type testDef struct {
	name             string
	recursive        bool
	srcRelativePath  string
	srcEntries       []fileDef
	dstRelativePath  string
	dstEntriesBefore []fileDef
	dstEntriesAfter  []fileDef
}

const (
	EARLY_MOD_TIME = "2020-01-01T00:00:00Z"
	LATER_MOD_TIME = "2025-01-01T00:00:00Z"
)

func TestSync(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	var tests = []testDef{
		{
			"copy file to existing file with different name",
			false,
			"test.txt",
			[]fileDef{
				{"test.txt", LATER_MOD_TIME, "foobar"},
			},
			"test2.txt",
			[]fileDef{
				{"test2.txt", EARLY_MOD_TIME, "barbaz"},
			},
			[]fileDef{
				{"test2.txt", LATER_MOD_TIME, "foobar"},
			},
		},
		{
			"skip copy file to different name if timestamps match",
			false,
			"test.txt",
			[]fileDef{
				{"test.txt", EARLY_MOD_TIME, "foobar"},
			},
			"test2.txt",
			[]fileDef{
				{"test2.txt", EARLY_MOD_TIME, "barbaz"},
			},
			[]fileDef{
				{"test2.txt", EARLY_MOD_TIME, "barbaz"},
			},
		},
		{
			"copy file to empty dir with different name",
			false,
			"test.txt",
			[]fileDef{
				{"test.txt", LATER_MOD_TIME, "foobar"},
			},
			"test2.txt",
			[]fileDef{},
			[]fileDef{
				{"test2.txt", LATER_MOD_TIME, "foobar"},
			},
		},
		{
			"copy file to non-empty dir as different name but with matching file name in dst",
			false,
			"test.txt",
			[]fileDef{
				{"test.txt", EARLY_MOD_TIME, "foobar"},
			},
			"test2.txt",
			[]fileDef{
				{"test.txt", LATER_MOD_TIME, "barbaz"},
			},
			[]fileDef{
				{"test.txt", LATER_MOD_TIME, "barbaz"},
				{"test2.txt", EARLY_MOD_TIME, "foobar"},
			},
		},
		{
			"skip copy file due to newer timestamp on dst",
			false,
			"test.txt",
			[]fileDef{
				{"test.txt", EARLY_MOD_TIME, "foobar"},
			},
			"test.txt",
			[]fileDef{
				{"test.txt", EARLY_MOD_TIME, "barbaz"},
			},
			[]fileDef{
				{"test.txt", EARLY_MOD_TIME, "barbaz"},
			},
		},
		{
			"copy even if newer timestamp on dst",
			false,
			"test.txt",
			[]fileDef{
				{"test.txt", EARLY_MOD_TIME, "foobar"},
			},
			"test.txt",
			[]fileDef{
				{"test.txt", LATER_MOD_TIME, "barbaz"},
			},
			[]fileDef{
				{"test.txt", EARLY_MOD_TIME, "foobar"},
			},
		},
		{
			"copy dir non-recursively",
			false,
			"test",
			[]fileDef{
				{"test/test1.txt", EARLY_MOD_TIME, "biz"},
				{"test/test2.txt", EARLY_MOD_TIME, "bang"},
				{"test/foo/test3.txt", EARLY_MOD_TIME, "boop"},
			},
			"test",
			[]fileDef{},
			[]fileDef{
				{"test/test1.txt", EARLY_MOD_TIME, "biz"},
				{"test/test2.txt", EARLY_MOD_TIME, "bang"},
			},
		},
		{
			"copy dir non-recursively to new name",
			false,
			"test",
			[]fileDef{
				{"test/test1.txt", EARLY_MOD_TIME, "biz"},
				{"test/test2.txt", EARLY_MOD_TIME, "bang"},
				{"test/foo/test3.txt", EARLY_MOD_TIME, "boop"},
			},
			"plebh",
			[]fileDef{},
			[]fileDef{
				{"plebh/test1.txt", EARLY_MOD_TIME, "biz"},
				{"plebh/test2.txt", EARLY_MOD_TIME, "bang"},
			},
		},
		{
			"copy dir recursively",
			true,
			"test",
			[]fileDef{
				{"test/test1.txt", EARLY_MOD_TIME, "biz"},
				{"test/test2.txt", EARLY_MOD_TIME, "bang"},
				{"test/foo/test3.txt", EARLY_MOD_TIME, "boop"},
			},
			"test",
			[]fileDef{},
			[]fileDef{
				{"test/test1.txt", EARLY_MOD_TIME, "biz"},
				{"test/test2.txt", EARLY_MOD_TIME, "bang"},
				{"test/foo/test3.txt", EARLY_MOD_TIME, "boop"},
			},
		},
		{
			"copy dir recursively to new name",
			true,
			"test",
			[]fileDef{
				{"test/test1.txt", EARLY_MOD_TIME, "biz"},
				{"test/test2.txt", EARLY_MOD_TIME, "bang"},
				{"test/foo/test3.txt", EARLY_MOD_TIME, "boop"},
			},
			"plebh",
			[]fileDef{},
			[]fileDef{
				{"plebh/test1.txt", EARLY_MOD_TIME, "biz"},
				{"plebh/test2.txt", EARLY_MOD_TIME, "bang"},
				{"plebh/foo/test3.txt", EARLY_MOD_TIME, "boop"},
			},
		},
		{
			"copy dir non-recursively with same timestamps has no updates",
			false,
			"test",
			[]fileDef{
				{"test/test1.txt", EARLY_MOD_TIME, "biz"},
				{"test/test2.txt", EARLY_MOD_TIME, "bang"},
				{"test/foo/test3.txt", LATER_MOD_TIME, "boop"},
			},
			"test",
			[]fileDef{
				{"test/test1.txt", EARLY_MOD_TIME, "faz"},
				{"test/test2.txt", EARLY_MOD_TIME, "bloop"},
				{"test/foo/test3.txt", EARLY_MOD_TIME, "fop"},
			},
			[]fileDef{
				{"test/test1.txt", EARLY_MOD_TIME, "faz"},
				{"test/test2.txt", EARLY_MOD_TIME, "bloop"},
				{"test/foo/test3.txt", EARLY_MOD_TIME, "fop"},
			},
		},
		{
			"copy dir recursively with timestamp-only updates",
			true,
			"test",
			[]fileDef{
				{"test/test1.txt", EARLY_MOD_TIME, "biz"},
				{"test/test2.txt", LATER_MOD_TIME, "bang"},
				{"test/foo/test3.txt", EARLY_MOD_TIME, "boop"},
			},
			"test",
			[]fileDef{
				{"test/test1.txt", EARLY_MOD_TIME, "faz"},
				{"test/test2.txt", EARLY_MOD_TIME, "bloop"},
				{"test/foo/test3.txt", EARLY_MOD_TIME, "fop"},
			},
			[]fileDef{
				{"test/test1.txt", EARLY_MOD_TIME, "faz"},
				{"test/test2.txt", LATER_MOD_TIME, "bang"},
				{"test/foo/test3.txt", EARLY_MOD_TIME, "fop"},
			},
		},

		// TODO:
		// - copies over later mod times (recursively & non-recursively)
		// - skips copying over later mod times when non-recursive (even if src exists)
	}

	for _, test := range tests {
		t.Run(test.name, func(s *testing.T) {
			executeTest(test, s)
		})
	}
}

func isDir(s string) bool { return strings.HasSuffix(s, "/") }

func createTestFile(root string, f fileDef) error {
	path := filepath.Join(root, f.name)
	dirName := filepath.Dir(path)
	if err := os.MkdirAll(dirName, 0700); err != nil {
		return fmt.Errorf("failed to create parent dirs (%s) in path %s: %v", dirName, path, err)
	}
	if err := os.WriteFile(path, []byte(f.content), 0600); err != nil {
		return fmt.Errorf("failed to create file %s: %v", path, err)
	}
	modTime, err := time.Parse(time.RFC3339, f.modTime)
	if err != nil {
		return fmt.Errorf("failed to parse mod time (%s); ensure tests are correct: %v", f.modTime, err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		return fmt.Errorf("failed to change (%s) mod time (%v): %v", path, modTime, err)
	}
	return nil
}

func executeTest(test testDef, t *testing.T) {
	testRunDir := filepath.Join("/tmp", fmt.Sprintf("go-test-%d", time.Now().Unix()))
	if err := os.Mkdir(testRunDir, 0700); err != nil {
		t.Fatalf("failed to create initial temp directory: %v", err)
	}
	defer os.RemoveAll(testRunDir)

	srcExecutor, srcRootPath := executor.NewLocalExecutor("src"), filepath.Join(testRunDir, "src")
	dstExecutor, dstRootPath := executor.NewLocalExecutor("dst"), filepath.Join(testRunDir, "dst")

	for _, d := range []string{srcRootPath, dstRootPath} {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatalf("failed to create directory %s: %v", d, err)
		}
	}

	for _, e := range test.srcEntries {
		if err := createTestFile(srcRootPath, e); err != nil {
			t.Fatalf("failed to create src test file: %v", err)
		}
	}

	for _, e := range test.dstEntriesBefore {
		if err := createTestFile(dstRootPath, e); err != nil {
			t.Fatalf("failed to create dst test file: %v", err)
		}
	}

	srcFile := filepath.Join(srcRootPath, test.srcRelativePath)
	dstFile := filepath.Join(dstRootPath, test.dstRelativePath)

	// TODO: Look at how we parameterize these guys. This is a little awkward.
	sut := NewFileProvider("test", srcFile, dstFile, test.recursive, false)
	config := config.SyncConfig{
		SrcExecutor: srcExecutor,
		DstExecutor: dstExecutor,
		Transport:   transport.NewLocalTransport(),
		DryRun:      false,
	}

	// TODO: Actually test return value
	if _, err := sut.Sync(config); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// TODO: Also check that no other files exist

	dstFiles, _ := readDirR(dstRootPath)
	slog.Info("dst files", "dst", dstRootPath, "files", dstFiles)

	// Check that all expected entries exist with correct content...
	for _, e := range test.dstEntriesAfter {
		path := filepath.Join(dstRootPath, e.name)
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to open dst file that should exist (%s): %v", path, err)
			continue
		}

		actual, expected := string(contentBytes), e.content
		if actual != expected {
			t.Errorf("invalid content for dst file %s: should be '%s', was '%s'", path, expected, actual)
		}
	}

	// ...and that no unexpected entries exist
	dstFilePaths := flattenDirEntry(dstFiles)
	for _, p := range dstFilePaths {
		if util.All(test.dstEntriesAfter, func(x fileDef) bool { return x.name != p }) {
			t.Errorf("unexpected file in dst: %s", p)
		}
	}
}

type dirEntry struct {
	Path         string
	RelativePath string
	IsDir        bool
	Children     []*dirEntry
}

func newFile(root, path string) *dirEntry {
	return &dirEntry{
		Path:         filepath.Join(root, path),
		RelativePath: path,
		IsDir:        false,
		Children:     []*dirEntry{},
	}
}

func newDir(root, path string) *dirEntry {
	return &dirEntry{
		Path:         filepath.Join(root, path),
		RelativePath: path,
		IsDir:        true,
		Children:     []*dirEntry{},
	}
}

func (d *dirEntry) appendChild(e *dirEntry) {
	d.Children = append(d.Children, e)
}

func (d *dirEntry) String() string {
	if d.IsDir {
		return fmt.Sprintf("d %s %v", d.RelativePath, d.Children)
	}
	return fmt.Sprintf("f %s", d.RelativePath)
}

func flattenDirEntry(d *dirEntry) []string {
	paths := []string{}
	next := []*dirEntry{d}
	for len(next) > 0 {
		var cur *dirEntry
		cur, next = next[0], next[1:]
		for _, c := range cur.Children {
			if c.IsDir {
				next = append(next, c)
			} else {
				paths = append(paths, c.RelativePath)
			}
		}
	}
	return paths
}

func readDirR(path string) (*dirEntry, error) {
	root := newDir(path, "")
	next := []*dirEntry{root}
	for len(next) > 0 {
		var cur *dirEntry
		cur, next = next[0], next[1:]
		curEntries, err := os.ReadDir(cur.Path)
		if err != nil {
			return nil, err
		}
		for _, e := range curEntries {
			erelpath := filepath.Join(cur.RelativePath, e.Name())
			if e.IsDir() {
				de := newDir(path, erelpath)
				cur.appendChild(de)
				next = append(next, de)
			} else {
				fe := newFile(path, erelpath)
				cur.appendChild(fe)
			}
		}
	}
	return root, nil
}
