package tally_test

import (
	"slices"
	"testing"

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

	if root.Tally.AuthorEmail != "bob@mail.com" {
		t.Errorf(
			"expected root tally author email to be \"%s\" but found \"%s\"",
			"bob@mail.com",
			root.Tally.AuthorEmail,
		)
	}

	if root.Tally.FileCount != 2 {
		t.Errorf(
			"expected root tally file count to be 2 but found %d",
			root.Tally.FileCount,
		)
	}

	if root.Tally.Commits != 2 {
		t.Errorf(
			"expected root commits to be 2 but found %d",
			root.Tally.Commits,
		)
	}

	if root.Tally.LinesAdded != 4+8+23 {
		t.Errorf(
			"expected root lines added to be %d but found %d",
			4+8+23,
			root.Tally.LinesAdded,
		)
	}

	if root.Tally.LinesRemoved != 2 {
		t.Errorf(
			"expected root lines removed to be 2 but found %d",
			root.Tally.LinesRemoved,
		)
	}

	if bimNode.Tally.Commits != 2 {
		t.Errorf(
			"expected bim node commits to be 2 but found %d",
			bimNode.Tally.Commits,
		)
	}

	if bimNode.Tally.FileCount != 1 {
		t.Errorf(
			"expected bim node file count to be 1 but found %d",
			bimNode.Tally.FileCount,
		)
	}

	if bimNode.Tally.LinesAdded != 4+23 {
		t.Errorf(
			"expected bim node lines added to be %d but found %d",
			4+23,
			bimNode.Tally.LinesAdded,
		)
	}

	if bimNode.Tally.LinesRemoved != 0 {
		t.Errorf(
			"expected bim node lines removed to be 0 but found %d",
			bimNode.Tally.LinesRemoved,
		)
	}
}
