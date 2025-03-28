package git_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sinclairtarget/git-who/internal/git"
)

func TestSupportedPathspec(t *testing.T) {
	tests := []struct {
		name     string
		pathspec string
		expected bool
	}{
		{
			name:     "empty_pathspec",
			pathspec: "",
			expected: true,
		},
		{
			name:     "literal_path",
			pathspec: "foo/bar.txt",
			expected: true,
		},
		{
			name:     "directory_prefix",
			pathspec: "foo/",
			expected: true,
		},
		{
			name:     "glob",
			pathspec: "foo/*.txt",
			expected: true,
		},
		{
			name:     "double_glob",
			pathspec: "foo/**/*.txt",
			expected: true,
		},
		{
			name:     "single_wildcard",
			pathspec: "foo/?ar.txt",
			expected: true,
		},
		{
			name:     "range",
			pathspec: "foo/[a-z]ar.txt",
			expected: true,
		},
		{
			name:     "ignore",
			pathspec: ":(exclude)vendor/",
			expected: true,
		},
		{
			name:     "ignore_short",
			pathspec: ":!vendor/",
			expected: true,
		},
		{
			name:     "ignore_short_caret",
			pathspec: ":^vendor/",
			expected: true,
		},
		{
			name:     "ignore_short_optional_colon",
			pathspec: ":!:vendor/",
			expected: true,
		},
		{
			name:     "ignore_leading_whitespace",
			pathspec: ":! foo.txt",
			expected: true,
		},
		{
			name:     "ignore_leading_tab",
			pathspec: ":!\tfoo.txt",
			expected: true,
		},
		{
			name:     "ignore_glob",
			pathspec: ":!*.txt",
			expected: true,
		},
		{
			name:     "ignore_pycache",
			pathspec: ":!__pycache__/",
			expected: true,
		},
		{
			name:     "attr",
			pathspec: ":(attr: foo)vendor/",
			expected: false,
		},
		{
			name:     "literal",
			pathspec: ":(literal)vendor/",
			expected: false,
		},
		{
			name:     "glob",
			pathspec: ":(glob)vendor/",
			expected: false,
		},
		{
			name:     "icase",
			pathspec: ":(icase)vendor/",
			expected: false,
		},
		{
			name:     "top",
			pathspec: ":(top)vendor/",
			expected: false,
		},
		{
			name:     "top_short",
			pathspec: ":/foo/bar.txt",
			expected: false,
		},
		{
			name:     "multiple",
			pathspec: ":(icase,exclude)foo/*.txt",
			expected: false,
		},
		{
			name:     "multiple_short",
			pathspec: ":!/foo/*.txt",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := git.IsSupportedPathspec(test.pathspec)
			if result != test.expected {
				t.Errorf(
					"expected pathspec \"%s\" supported is %v but got %v",
					test.pathspec,
					test.expected,
					result,
				)
			}
		})
	}
}

func TestSplitPathspecs(t *testing.T) {
	tests := []struct {
		name      string
		pathspecs []string
		includes  []string
		excludes  []string
	}{
		{
			name:      "long",
			pathspecs: []string{"*.txt", ":(exclude)vendor/"},
			includes:  []string{"*.txt"},
			excludes:  []string{"vendor/"},
		},
		{
			name:      "short",
			pathspecs: []string{"*.txt", ":!vendor/"},
			includes:  []string{"*.txt"},
			excludes:  []string{"vendor/"},
		},
		{
			name:      "caret",
			pathspecs: []string{"*.txt", ":^vendor/"},
			includes:  []string{"*.txt"},
			excludes:  []string{"vendor/"},
		},
		{
			name:      "optional_colon",
			pathspecs: []string{"*.txt", ":!:vendor/"},
			includes:  []string{"*.txt"},
			excludes:  []string{"vendor/"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			includes, excludes := git.SplitPathspecs(test.pathspecs)
			if diff := cmp.Diff(test.includes, includes); diff != "" {
				t.Errorf("includes is wrong:\n%s", diff)
			}
			if diff := cmp.Diff(test.excludes, excludes); diff != "" {
				t.Errorf("excludes is wrong:\n%s", diff)
			}
		})
	}
}

func TestPathspecMatch(t *testing.T) {
	tests := []struct {
		name     string
		pathspec string
		path     string
		expected bool
	}{
		{
			name:     "empty_path",
			pathspec: "*",
			path:     "",
			expected: true,
		},
		{
			name:     "directory_prefix",
			pathspec: "foo/",
			path:     "foo/bar.txt",
			expected: true,
		},
		{
			name:     "glob",
			pathspec: "*",
			path:     "foo",
			expected: true,
		},
		{
			name:     "dir",
			pathspec: "foo",
			path:     "foo/bar.txt",
			expected: true,
		},
		{
			name:     "glob_dir",
			pathspec: "foo/*",
			path:     "foo/bar.txt",
			expected: true,
		},
		{
			name:     "glob_dir_ext",
			pathspec: "foo/*.txt",
			path:     "foo/bar.txt",
			expected: true,
		},
		{
			name:     "double_glob",
			pathspec: "foo/**/bar.txt",
			path:     "foo/bim/bam/bar.txt",
			expected: true,
		},
		{
			name:     "double_glob_dir",
			pathspec: "foo/",
			path:     "foo/bim/bam/bar.txt",
			expected: true,
		},
		{
			name:     "toplevel_glob",
			pathspec: "*_test.go",
			path:     "foo/bim/bam/foo_test.go",
			expected: true,
		},
		{
			name:     "glob_not_match",
			pathspec: "*.txt",
			path:     "README.md",
			expected: false,
		},
		{
			name:     "glob_dir_not_match",
			pathspec: "foo/*.txt",
			path:     "foo/bim/bam/bar.txt",
			expected: false,
		},
		{
			name:     "subdir_not_match",
			pathspec: "foo/bim",
			path:     "foo/bar.txt",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := git.PathspecMatch(test.pathspec, test.path)
			if result != test.expected {
				t.Errorf(
					"expected match of path \"%s\" to pathspec \"%s\" to be %v but got %v",
					test.path,
					test.pathspec,
					test.expected,
					result,
				)
			}
		})
	}
}
