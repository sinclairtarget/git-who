// This file contains tests for git-who's argument parsing.
//
// Since properly parsing the arguments involves invoking git rev-parse as a
// subprocess, these tests run against the test repo submodule.

package git_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/sinclairtarget/git-who/internal/git"
)

const safeTag string = "root"
const safeCommit string = "6afef287af5ca43f7d741e7ceff61aad38055b6a"
const filename string = "README.md"

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expRevs  []string
		expPaths []string
	}{
		{
			name:     "empty_args",
			args:     []string{},
			expRevs:  []string{"HEAD"},
			expPaths: []string{"."},
		},
		{
			name:     "commit",
			args:     []string{safeTag},
			expRevs:  []string{safeCommit},
			expPaths: []string{"."},
		},
		{
			name:     "commit_path",
			args:     []string{safeTag, filename},
			expRevs:  []string{safeCommit},
			expPaths: []string{filename},
		},
		{
			name:     "path",
			args:     []string{filename},
			expRevs:  []string{"HEAD"},
			expPaths: []string{filename},
		},
		{
			name:     "separator",
			args:     []string{safeTag, "--", filename},
			expRevs:  []string{safeCommit},
			expPaths: []string{filename},
		},
		{
			name:     "nonexistant_path_after_separator",
			args:     []string{safeTag, "--", "foobar"},
			expRevs:  []string{safeCommit},
			expPaths: []string{"foobar"},
		},
		{
			name:     "trailing_separator",
			args:     []string{safeTag, "--"},
			expRevs:  []string{safeCommit},
			expPaths: []string{"."},
		},
		{
			name:     "leading_separator",
			args:     []string{"--", filename},
			expRevs:  []string{"HEAD"},
			expPaths: []string{filename},
		},
		{
			name: "multiple_args",
			args: []string{
				safeTag,
				safeTag,
				filename,
				filename,
			},
			expRevs:  []string{safeCommit, safeCommit},
			expPaths: []string{filename, filename},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			revs, paths, err := git.ParseArgs(test.args)
			if err != nil {
				var subErr git.SubprocessErr
				if errors.As(err, &subErr) {
					t.Logf("subprocess error output:\n%s", subErr.Stderr)
				}
				t.Errorf("got error: %v", err)
			}

			if !slices.Equal(revs, test.expRevs) {
				t.Errorf(
					"expected %v as revs but got %v",
					test.expRevs,
					revs,
				)
			}

			if !slices.Equal(paths, test.expPaths) {
				t.Errorf(
					"expected %v as paths but got %v",
					test.expPaths,
					paths,
				)
			}
		})
	}
}

func TestParseArgsError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "not_path_or_rev",
			args: []string{"foobar"},
		},
		{
			name: "not_path",
			args: []string{safeTag, "foobar"},
		},
		{
			name: "not_rev",
			args: []string{"foobar", "--", filename},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := git.ParseArgs(test.args)
			if err == nil {
				t.Error("expected error, but none returned")
			}
		})
	}
}

func TestParseArgsRange(t *testing.T) {
	revs, paths, err := git.ParseArgs([]string{"HEAD~3.."})
	if err != nil {
		t.Errorf("got unexpected error: %v", err)
	}

	if len(revs) != 2 {
		t.Errorf("expected revs to have length 2, but got: %v", revs)
	}

	expPaths := []string{"."}
	if !slices.Equal(paths, expPaths) {
		t.Errorf("expected %v as paths but got %v", expPaths, paths)
	}
}
