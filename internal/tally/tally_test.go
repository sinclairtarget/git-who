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
	tallies, err := tally.TallyCommits(seq, tally.CommitMode)
	if err != nil {
		t.Fatalf("TallyCommits() returned error: %v", err)
	}

	if len(tallies) == 0 {
		t.Fatalf("TallyCommits() returned empty slice")
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
				git.FileDiff{
					Path:         "bim.txt",
					LinesAdded:   4,
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
	tallies, err := tally.TallyCommits(seq, tally.CommitMode)
	if err != nil {
		t.Fatalf("TallyCommits() returned error: %v", err)
	}

	if len(tallies) == 0 {
		t.Fatal("TallyCommits() returned empty slice")
	}

	bob := tallies[0]
	expected := tally.Tally{
		AuthorName:   "bob",
		AuthorEmail:  "bob@mail.com",
		Commits:      2,
		LinesAdded:   8,
		LinesRemoved: 2,
		FileCount:    1, // Should just be 1 since file was moved
	}
	if diff := cmp.Diff(expected, bob); diff != "" {
		t.Errorf("bob's tally is wrong:\n%s", diff)
	}

	jim := tallies[1]
	expected = tally.Tally{
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
