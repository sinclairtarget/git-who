package tally_test

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/iterutils"
	"github.com/sinclairtarget/git-who/internal/tally"
)

func TestTallyCommitsByPath(t *testing.T) {
	commits := []git.Commit{
		git.Commit{
			Hash:        "baa",
			ShortHash:   "baa",
			AuthorName:  "bob",
			AuthorEmail: "bob@mail.com",
			FileDiffs: []git.FileDiff{
				git.FileDiff{
					Path:         "foo/bim.txt",
					LinesAdded:   4,
					LinesRemoved: 0,
				},
				git.FileDiff{
					Path:         "foo/bar.txt",
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
					Path:         "foo/bim.txt",
					LinesAdded:   3,
					LinesRemoved: 1,
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
					Path:         "foo/bim.txt",
					LinesAdded:   23,
					LinesRemoved: 0,
				},
			},
		},
	}

	seq := iterutils.WithoutErrors(slices.Values(commits))
	root, err := tally.TallyCommitsByPath(seq, tally.CommitMode)
	if err != nil {
		t.Fatalf("TallyCommits() returned error: %v", err)
	}

	if len(root.Children) == 0 {
		t.Fatalf("root node has no children")
	}

	fooNode, ok := root.Children["foo"]
	if !ok {
		t.Fatalf("root node has no \"foo\" child")
	}

	bimNode, ok := fooNode.Children["bim.txt"]
	if !ok {
		t.Errorf("\"foo\" node has no \"bim.txt\" child")
	}

	_, ok = fooNode.Children["bar.txt"]
	if !ok {
		t.Errorf("\"foo\" node has no \"bar.txt\" child")
	}

	expected := tally.Tally{
		AuthorName:   "bob",
		AuthorEmail:  "bob@mail.com",
		Commits:      2,
		LinesAdded:   4 + 8 + 23,
		LinesRemoved: 2,
		FileCount:    2,
	}
	if diff := cmp.Diff(expected, root.Tally); diff != "" {
		t.Errorf("bob's tally is wrong:\n%s", diff)
	}

	expected = tally.Tally{
		AuthorName:   "bob",
		AuthorEmail:  "bob@mail.com",
		Commits:      2,
		LinesAdded:   4 + 23,
		LinesRemoved: 0,
		FileCount:    1,
	}
	if diff := cmp.Diff(expected, bimNode.Tally); diff != "" {
		t.Errorf("bob's second tally is wrong:\n%s", diff)
	}
}
