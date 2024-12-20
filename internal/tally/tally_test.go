package tally_test

import (
	"testing"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

func TestTallyCommits(t *testing.T) {
	commits := []git.Commit {
		git.Commit {
			Hash: "baa",
			ShortHash: "baa",
			AuthorName: "bob",
			AuthorEmail: "bob@mail.com",
			FileDiffs: []git.FileDiff {
				git.FileDiff {
					Path: "bim.txt",
					LinesAdded: 4,
					LinesRemoved: 0,
				},
				git.FileDiff {
					Path: "vim.txt",
					LinesAdded: 8,
					LinesRemoved: 2,
				},
			},
		},
		git.Commit {
			Hash: "bab",
			ShortHash: "bab",
			AuthorName: "jim",
			AuthorEmail: "jim@mail.com",
			FileDiffs: []git.FileDiff {
				git.FileDiff {
					Path: "bim.txt",
					LinesAdded: 3,
					LinesRemoved: 1,
				},
			},
		},
	}

	iter := func(yield func(git.Commit, error) bool) {
		for _, commit := range commits {
			if !yield(commit, nil) {
				break
			}
		}
	}

	tallies, err := tally.TallyCommits(iter)
	if err != nil {
		t.Fatalf("TallyCommits() returned error: %v", err)
	}

	if len(tallies) == 0 {
		t.Fatalf("TallyCommits() returned empty map")
	}
}
