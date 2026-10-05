package data_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/braginantonev/mhserver/internal/services"
	"github.com/braginantonev/mhserver/internal/services/data"
	"github.com/braginantonev/mhserver/pkg/contextkeys"
)

const Username string = "V"

func compileExpectedDir(dir string) string {
	return fmt.Sprintf("%s/%s", Username, strings.TrimPrefix(dir, "/"))
}

func compileExpectedFilepath(dir, file string) string {
	return fmt.Sprintf("%s/%s/%s", Username, strings.TrimPrefix(strings.TrimSuffix(dir, "/"), "/"), file)
}

func TestCompileUserDirectory(t *testing.T) {
	ctx := context.WithValue(t.Context(), contextkeys.USERNAME, Username)

	cases := [...]struct {
		name         string
		dir          string
		service      services.ServiceName
		expected     string
		expected_err error
	}{
		{
			name:         "empty dir",
			expected_err: data.ErrBadDirSyntax,
		},
		{
			name:         "with twice point",
			dir:          "/../a/",
			expected_err: data.ErrBadDirSyntax,
		},
		{
			name:     "single dir",
			dir:      "single",
			service:  data.SERVICE_NAME,
			expected: compileExpectedDir("single"),
		},
		{
			name:     "with included dirs",
			dir:      "first/second/third",
			service:  data.SERVICE_NAME,
			expected: compileExpectedDir("first/second/third"),
		},
		{
			name:     "with twice separator",
			dir:      "abc//cba",
			service:  data.SERVICE_NAME,
			expected: compileExpectedDir("abc/cba"),
		},
		{
			name:     "with suffix and prefix",
			dir:      "/ab/av/",
			service:  data.SERVICE_NAME,
			expected: compileExpectedDir("/ab/av"),
		},
		{
			name:     "another service",
			dir:      "ab/av",
			service:  "llm",
			expected: compileExpectedDir("llm/ab/av"),
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := data.CompileUserDirectory(ctx, test.dir, test.service)
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
	ctx := context.WithValue(t.Context(), contextkeys.USERNAME, Username)
	dir := "test"

	cases := [...]struct {
		name         string
		file         string
		expected     string
		expected_err error
	}{
		{
			name:         "empty file",
			expected_err: data.ErrBadFilenameSyntax,
		},
		{
			name:         "dir in file",
			file:         "wrong/a.docx",
			expected_err: data.ErrBadFilenameSyntax,
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
			got, err := data.CompileUserFilepath(ctx, dir, test.file, data.SERVICE_NAME)
			if err != test.expected_err {
				t.Errorf("expected error: `%v`, but got: `%v`", test.expected_err, err)
			}

			if got != test.expected {
				t.Errorf("expected dir: `%s`, but got: `%s`", test.expected, got)
			}
		})
	}
}
