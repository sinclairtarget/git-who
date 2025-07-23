package git_test

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sinclairtarget/git-who/internal/git"
)

func TestLimitDiffsByPathspec(t *testing.T) {
	commit := git.Commit{
		Hash:        "abc123",
		ShortHash:   "abc123",
		IsMerge:     false,
		AuthorName:  "Bob",
		AuthorEmail: "bob@foo.com",
		FileDiffs: []git.FileDiff{
			git.FileDiff{Path: "foo.txt"},
			git.FileDiff{Path: "main.go"},
		},
	}

	tests := []struct {
		name      string
		commits   []git.Commit
		pathspecs []string
		expected  []git.FileDiff
	}{
		{
			name:      "empty_case",
			commits:   []git.Commit{commit},
			pathspecs: []string{},
			expected: []git.FileDiff{
				git.FileDiff{Path: "foo.txt"},
				git.FileDiff{Path: "main.go"},
			},
		},
		{
			name:      "include_only",
			commits:   []git.Commit{commit},
			pathspecs: []string{"*.js"},
			expected:  []git.FileDiff{},
		},
		{
			name:      "include_only_multiple",
			commits:   []git.Commit{commit},
			pathspecs: []string{"*.js", "*.txt"},
			expected:  []git.FileDiff{git.FileDiff{Path: "foo.txt"}},
		},
		{
			name:      "exclude_only",
			commits:   []git.Commit{commit},
			pathspecs: []string{":!*.txt"},
			expected:  []git.FileDiff{git.FileDiff{Path: "main.go"}},
		},
		{
			name:      "exclude_only_multiple",
			commits:   []git.Commit{commit},
			pathspecs: []string{":!*.txt", "!*.go"},
			expected:  []git.FileDiff{},
		},
		{
			name:      "include_exclude",
			commits:   []git.Commit{commit},
			pathspecs: []string{"*.txt", ":!*.txt"},
			expected:  []git.FileDiff{},
		},
		{
			name:      "include_exclude_longform_magic",
			commits:   []git.Commit{commit},
			pathspecs: []string{"*.txt", ":(exclude)*.txt"},
			expected:  []git.FileDiff{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commits := slices.Values(test.commits)
			seq, err := git.LimitDiffsByPathspec(commits, test.pathspecs)
			if err != nil {
				t.Fatalf("got error value: %v", err)
			}

			limitedCommits := slices.Collect(seq)

			diffs := []git.FileDiff{}

			// slices.Collect(slices.Values([]T{})) returns nil for some reason
			if limitedCommits != nil {
				if len(limitedCommits) < 1 {
					t.Fatalf("not enough commits returned")
				}

				diffs = limitedCommits[0].FileDiffs
			}

			if diff := cmp.Diff(test.expected, diffs); diff != "" {
				t.Errorf("found file diffs do not match:\n%s", diff)
			}
		})
	}
}
