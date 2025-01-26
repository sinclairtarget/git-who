package tally_test

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
	"github.com/sinclairtarget/git-who/internal/utils/iterutils"
)

func TestTallyCommitsTree(t *testing.T) {
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

	worktreeset := map[string]bool{"foo/bim.txt": true, "foo/bar.txt": true}
	seq := iterutils.WithoutErrors(slices.Values(commits))
	opts := tally.TallyOpts{
		Mode: tally.CommitMode,
		Key:  func(c git.Commit) string { return c.AuthorEmail },
	}

	root, err := tally.TallyCommitsTree(seq, opts, worktreeset, "")
	if err != nil {
		t.Fatalf("TallyCommits() returned error: %v", err)
	}

	root = root.Rank(opts.Mode)

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

	expected := tally.FinalTally{
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

	expected = tally.FinalTally{
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

func TestTallyCommitsTreeNoCommits(t *testing.T) {
	seq := iterutils.WithoutErrors(slices.Values([]git.Commit{}))
	opts := tally.TallyOpts{
		Mode: tally.CommitMode,
		Key:  func(c git.Commit) string { return c.AuthorEmail },
	}
	worktreeset := map[string]bool{}

	_, err := tally.TallyCommitsTree(seq, opts, worktreeset, "")
	if err != tally.EmptyTreeErr {
		t.Fatalf(
			"TallyCommits() should have returned EmptyTreeErr but returned %v",
			err,
		)
	}
}
