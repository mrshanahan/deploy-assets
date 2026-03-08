package provider

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mrshanahan/deploy-assets/internal/util"
	"github.com/mrshanahan/deploy-assets/pkg/config"
	"github.com/mrshanahan/deploy-assets/pkg/executor"
	"github.com/mrshanahan/deploy-assets/pkg/transport"
)

func TestLiteralYamlSingleLine(t *testing.T) {
	p := NewLiteralProvider("foobar", "boop sdlkjf lskdfjlksd lkfjsdlkfdsjlksfdj kl", "blap.txt")
	expected :=
		`literal:
    name: foobar
    value: |
        boop sdlkjf lskdfjlksd lkfjsdlkfdsjlksfdj kl
    dst_path: blap.txt`
	actual := p.Yaml(0)
	if expected != actual {
		t.Errorf("yaml contents not equal:\nexpected:\n=======\n%s\n=======\ngot:\n=======\n%s\n=======", expected, actual)
	}
}

func TestLiteralYamlMultiLine(t *testing.T) {
	p := NewLiteralProvider("foobar", "boop sdlkjf\nlskdfjlksd lkfjsdlkfdsjlksfdj\nkl", "blap.txt")
	expected :=
		`literal:
    name: foobar
    value: |
        boop sdlkjf
        lskdfjlksd lkfjsdlkfdsjlksfdj
        kl
    dst_path: blap.txt`
	actual := p.Yaml(0)
	if expected != actual {
		t.Errorf("yaml contents not equal:\nexpected:\n=======\n%s\n=======\ngot:\n=======\n%s\n=======", expected, actual)
	}
}

func TestLiteralYamlTrailingLeadingMultiLine(t *testing.T) {
	p := NewLiteralProvider("foobar", "\nboop sdlkjf\nlskdfjlksd lkfjsdlkfdsjlksfdj\nkl\n\n", "blap.txt")
	expected :=
		`literal:
    name: foobar
    value: |
        
        boop sdlkjf
        lskdfjlksd lkfjsdlkfdsjlksfdj
        kl
        
        
    dst_path: blap.txt`
	actual := p.Yaml(0)
	if expected != actual {
		t.Errorf("yaml contents not equal:\nexpected:\n=======\n%s\n=======\ngot:\n=======\n%s\n=======", expected, actual)
	}
}

func TestLiteralSync(t *testing.T) {
	tests := []struct {
		name             string
		value            string
		dstPath          string
		dstEntriesBefore []fileDef
		dstEntriesAfter  []fileDef
	}{
		{
			"create",
			"abc",
			"foo.bar",
			[]fileDef{},
			[]fileDef{
				{
					name:    "foo.bar",
					content: "abc",
				},
			},
		},
		{
			"update-if-different",
			"abc",
			"foo.bar",
			[]fileDef{
				{
					name:    "foo.bar",
					content: "def",
					modTime: EARLY_MOD_TIME,
				},
			},
			[]fileDef{
				{
					name:    "foo.bar",
					content: "abc",
				},
			},
		},
		{
			"update-if-same",
			"abc",
			"foo.bar",
			[]fileDef{
				{
					name:    "foo.bar",
					content: "abc",
					modTime: EARLY_MOD_TIME,
				},
			},
			[]fileDef{
				{
					name:    "foo.bar",
					content: "abc",
					modTime: "NOW",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(s *testing.T) {
			executeLiteralSyncTest(test.value, test.dstPath, test.dstEntriesBefore, test.dstEntriesAfter, s)
		})
	}
}

func executeLiteralSyncTest(value string, dstPath string, dstEntriesBefore []fileDef, dstEntriesAfter []fileDef, t *testing.T) {
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

	for _, e := range dstEntriesBefore {
		if err := createTestFile(dstRootPath, e); err != nil {
			t.Fatalf("failed to create dst test file: %v", err)
		}
	}

	dstFile := filepath.Join(dstRootPath, dstPath)

	sut := NewLiteralProvider("test", value, dstFile)
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
	for _, e := range dstEntriesAfter {
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

		// Check mod time
		actualFile, err := os.Stat(path)
		if err != nil {
			t.Errorf("failed to stat dst file that should exist (%s): %v", path, err)
			continue
		}

		actualModTime := actualFile.ModTime()
		if e.modTime != "" { // Not all test cases care about mod time.
			if e.modTime == "NOW" {
				if time.Now().Sub(actualModTime) > time.Hour*24*7 {
					t.Errorf("expected recent mod time, instead was %v", actualModTime)
				}
			} else {
				expectedModTime, err := time.Parse(time.RFC3339, e.modTime)
				if err != nil {
					t.Errorf("failed to parse expected mod time %s: %v", e.modTime, err)
					continue
				}
				if expectedModTime != actualModTime {
					t.Errorf("expected mod time %v, got %v (%s)", expectedModTime, actualModTime, path)
				}
			}
		}
	}

	// ...and that no unexpected entries exist
	dstFilePaths := flattenDirEntry(dstFiles)
	for _, p := range dstFilePaths {
		if util.All(dstEntriesAfter, func(x fileDef) bool { return x.name != p }) {
			t.Errorf("unexpected file in dst: %s", p)
		}
	}
}
