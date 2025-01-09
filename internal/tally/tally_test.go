package tally_test

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/iterutils"
	"github.com/sinclairtarget/git-who/internal/tally"
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
	wtreeset := map[string]bool{"bim.txt": true, "vim.txt": true}
	opts := tally.TallyOpts{
		Mode: tally.LinesMode,
		Key: func(c git.Commit) string {
			return c.AuthorEmail
		},
	}
	tallies, err := tally.TallyCommits(seq, wtreeset, false, opts)
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
		LinesAdded:   12,
		LinesRemoved: 2,
		FileCount:    2, // Only two files in working tree
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

func TestTallyCommitsRename(t *testing.T) {
	commits := []git.Commit{
		git.Commit{
			Hash:        "baa",
			ShortHash:   "baa",
			AuthorName:  "bob",
			AuthorEmail: "bob@mail.com",
			FileDiffs: []git.FileDiff{
				git.FileDiff{ // This diff should be lost, too many renames
					Path:         "nim.txt",
					LinesAdded:   1,
					LinesRemoved: 1,
					MoveDest:     "bim.txt",
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
					MoveDest:     "bar.txt",
				},
			},
		},
		git.Commit{
			Hash:        "bac",
			ShortHash:   "bac",
			AuthorName:  "bob",
			AuthorEmail: "bob@mail.com",
			FileDiffs: []git.FileDiff{
				git.FileDiff{
					Path:         "bar.txt",
					LinesAdded:   4,
					LinesRemoved: 1,
				},
			},
		},
	}

	seq := iterutils.WithoutErrors(slices.Values(commits))
	wtreeset := map[string]bool{"bar.txt": true}
	opts := tally.TallyOpts{
		Mode: tally.LinesMode,
		Key: func(c git.Commit) string {
			return c.AuthorEmail
		},
	}
	tallies, err := tally.TallyCommits(seq, wtreeset, false, opts)
	if err != nil {
		t.Fatalf("TallyCommits() returned error: %v", err)
	}

	rankedTallies := tally.Rank(tallies, opts.Mode)
	if len(rankedTallies) != 2 {
		t.Fatal("TallyCommits() returned wrong number of tallies")
	}

	bob := rankedTallies[0]
	expected := tally.FinalTally{
		AuthorName:   "bob",
		AuthorEmail:  "bob@mail.com",
		Commits:      1,
		LinesAdded:   4,
		LinesRemoved: 1,
		FileCount:    1, // Should just be 1, since it's only file in tree
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
