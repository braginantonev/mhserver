package files_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/braginantonev/mhserver/internal/services/files"
)

const Username string = "V"

func compileExpectedDir(dir string) string {
	return fmt.Sprintf("%s/%s", Username, strings.TrimPrefix(dir, "/"))
}

func compileExpectedFilepath(dir, file string) string {
	return fmt.Sprintf("%s/%s/%s", Username, strings.TrimPrefix(strings.TrimSuffix(dir, "/"), "/"), file)
}

func TestCompileUserDirectory(t *testing.T) {
	cases := [...]struct {
		name         string
		dir          string
		expected     string
		expected_err error
	}{
		{
			name:         "empty dir",
			expected_err: files.ErrBadDirSyntax,
		},
		{
			name:         "with twice point",
			dir:          "/../a/",
			expected_err: files.ErrBadDirSyntax,
		},
		{
			name:     "single dir",
			dir:      "single",
			expected: compileExpectedDir("single"),
		},
		{
			name:     "with included dirs",
			dir:      "first/second/third",
			expected: compileExpectedDir("first/second/third"),
		},
		{
			name:     "with twice separator",
			dir:      "abc//cba",
			expected: compileExpectedDir("abc/cba"),
		},
		{
			name:     "with suffix and prefix",
			dir:      "/ab/av/",
			expected: compileExpectedDir("/ab/av"),
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := files.CompileUserDirectory(Username, test.dir)
			if err != test.expected_err {
				t.Errorf("expected error: `%v`, but got: `%v`", test.expected_err, err)
			}

			if got != test.expected {
				t.Errorf("expected dir: `%s`, but got: `%s`", test.expected, got)
			}
		})
	}
}

func TestCompileUserFilepath(t *testing.T) {
	dir := "test"

	cases := [...]struct {
		name         string
		file         string
		expected     string
		expected_err error
	}{
		{
			name:         "empty file",
			expected_err: files.ErrBadFilenameSyntax,
		},
		{
			name:         "dir in file",
			file:         "wrong/a.docx",
			expected_err: files.ErrBadFilenameSyntax,
		},
		{
			name:     "simple file",
			file:     "simple.txt",
			expected: compileExpectedFilepath(dir, "simple.txt"),
		},
		{
			name:     "multi extensions",
			file:     "multi.txt.bak.default",
			expected: compileExpectedFilepath(dir, "multi.txt.bak.default"),
		},
		{
			name:     "hidden file",
			file:     ".env",
			expected: compileExpectedFilepath(dir, ".env"),
		},
		{
			name:     "linux executable",
			file:     "zxc",
			expected: compileExpectedFilepath(dir, "zxc"),
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := files.CompileUserFilepath(Username, dir, test.file)
			if err != test.expected_err {
				t.Errorf("expected error: `%v`, but got: `%v`", test.expected_err, err)
			}

			if got != test.expected {
				t.Errorf("expected dir: `%s`, but got: `%s`", test.expected, got)
			}
		})
	}
}
