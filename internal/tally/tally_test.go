package tally_test

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
	"github.com/sinclairtarget/git-who/internal/utils/iterutils"
)

func TestTallyCommits(t *testing.T) {
	commits := []git.Commit{
		git.Commit{
			Hash:        "baa",
			ShortHash:   "baa",
			AuthorName:  "bob",
			AuthorEmail: "bob@mail.com",
			FileDiffs: []git.FileDiff{
				git.FileDiff{
					Path:         "bim.txt",
					LinesAdded:   4,
					LinesRemoved: 0,
				},
				git.FileDiff{
					Path:         "vim.txt",
					LinesAdded:   8,
					LinesRemoved: 2,
				},
				git.FileDiff{
					Path:         "nim.txt",
					LinesAdded:   2,
					LinesRemoved: 1,
				},
			},
		},
		git.Commit{
			Hash:        "bab",
			ShortHash:   "bab",
			AuthorName:  "jim",
			AuthorEmail: "jim@mail.com",
			FileDiffs: []git.FileDiff{
				git.FileDiff{
					Path:         "bim.txt",
					LinesAdded:   3,
					LinesRemoved: 1,
				},
			},
		},
	}

	seq := iterutils.WithoutErrors(slices.Values(commits))
	opts := tally.TallyOpts{
		Mode: tally.LinesMode,
		Key: func(c git.Commit) string {
			return c.AuthorEmail
		},
	}
	tallies, err := tally.TallyCommits(seq, opts)
	rankedTallies := tally.Rank(tallies, opts.Mode)
	if err != nil {
		t.Fatalf("TallyCommits() returned error: %v", err)
	}

	if len(rankedTallies) == 0 {
		t.Fatalf("TallyCommits() returned empty slice")
	}

	bob := rankedTallies[0]
	expected := tally.FinalTally{
		AuthorName:   "bob",
		AuthorEmail:  "bob@mail.com",
		Commits:      1,
		LinesAdded:   14,
		LinesRemoved: 3,
		FileCount:    3,
	}
	if diff := cmp.Diff(expected, bob); diff != "" {
		t.Errorf("bob's tally is wrong:\n%s", diff)
	}

	jim := rankedTallies[1]
	expected = tally.FinalTally{
		AuthorName:   "jim",
		AuthorEmail:  "jim@mail.com",
		Commits:      1,
		LinesAdded:   3,
		LinesRemoved: 1,
		FileCount:    1,
	}
	if diff := cmp.Diff(expected, jim); diff != "" {
		t.Errorf("jim's tally is wrong:\n%s", diff)
	}
}
