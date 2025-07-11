package backends_test

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/sinclairtarget/git-who/internal/cache/backends"
	"github.com/sinclairtarget/git-who/internal/git"
)

func TestAddGetClear(t *testing.T) {
	dir := t.TempDir()
	c := backends.JSONBackend{
		Path: filepath.Join(dir, "commits.json"),
	}

	err := c.Open()
	if err != nil {
		t.Fatalf("could not open cache: %v", err)
	}
	defer func() {
		err = c.Close()
		if err != nil {
			t.Fatalf("could not close cache: %v", err)
		}
	}()

	commit := git.Commit{
		ShortHash:   "9e9ea7662b1",
		Hash:        "9e9ea7662b1001d860471a4cece5e2f1de8062fb",
		AuthorName:  "Bob",
		AuthorEmail: "bob@work.com",
		Date: time.Date(
			2025, 1, 31, 16, 35, 26, 0, time.UTC,
		),
		FileDiffs: []git.FileDiff{
			{
				Path:         "foo/bar.txt",
				LinesAdded:   3,
				LinesRemoved: 5,
			},
		},
	}

	// -- Add --
	err = c.Add([]git.Commit{commit})
	if err != nil {
		t.Fatalf("add commits to cache failed with error: %v", err)
	}

	// -- Get --
	revs := []string{commit.Hash}
	seq, finish := c.Get(revs)
	commits := slices.Collect(seq)
	err = finish()
	if err != nil {
		t.Fatalf("error iterating cached commits: %v", err)
	}

	if len(commits) == 0 {
		t.Fatal("not enough commits in result")
	}

	cachedCommit := commits[0]
	if diff := cmp.Diff(commit, cachedCommit); diff != "" {
		t.Errorf("commit is wrong:\n%s", diff)
	}

	// -- Clear --
	err = c.Clear()
	if err != nil {
		t.Fatalf("clearing cache failed with error: %v", err)
	}

	seq, finish = c.Get(revs)
	commits = slices.Collect(seq)
	err = finish()
	if err != nil {
		t.Fatalf(
			"get commits from cache after clear failed with error: %v",
			err,
		)
	}

	if len(commits) > 0 {
		t.Errorf("cache result after clear should have been empty")
	}
}

func TestAddGetAddGet(t *testing.T) {
	dir := t.TempDir()
	c := backends.JSONBackend{
		Path: filepath.Join(dir, "commits.json"),
	}

	err := c.Open()
	if err != nil {
		t.Fatalf("could not open cache: %v", err)
	}
	defer func() {
		err = c.Close()
		if err != nil {
			t.Fatalf("could not close cache: %v", err)
		}
	}()

	commitOne := git.Commit{
		ShortHash:   "1e9ea7662b1",
		Hash:        "1e9ea7662b1001d860471a4cece5e2f1de8062fb",
		AuthorName:  "Bob",
		AuthorEmail: "bob@work.com",
		Date: time.Date(
			2025, 1, 30, 16, 35, 26, 0, time.UTC,
		),
		FileDiffs: []git.FileDiff{
			{
				Path:         "foo/bar.txt",
				LinesAdded:   3,
				LinesRemoved: 5,
			},
		},
	}
	commitTwo := git.Commit{
		ShortHash:   "2e9ea7662b1",
		Hash:        "2e9ea7662b1001d860471a4cece5e2f1de8062fb",
		AuthorName:  "Bob",
		AuthorEmail: "bob@work.com",
		Date: time.Date(
			2025, 1, 31, 16, 35, 26, 0, time.UTC,
		),
		FileDiffs: []git.FileDiff{
			{
				Path:         "foo/bim.txt",
				LinesAdded:   4,
				LinesRemoved: 0,
			},
		},
	}
	revs := []string{commitOne.Hash, commitTwo.Hash}

	err = c.Add([]git.Commit{commitOne})
	if err != nil {
		t.Fatalf("add commits to cache failed with error: %v", err)
	}

	seq, finish := c.Get(revs)
	commits := slices.Collect(seq)
	err = finish()
	if err != nil {
		t.Fatalf("error iterating commits from cache: %v", err)
	}

	if len(commits) != 1 {
		t.Errorf(
			"expected to get one commit from cache, but got %d",
			len(commits),
		)
	}

	err = c.Add([]git.Commit{commitTwo})
	if err != nil {
		t.Fatalf("add commits to cache failed with error: %v", err)
	}

	seq, finish = c.Get(revs)
	commits = slices.Collect(seq)
	err = finish()
	if err != nil {
		t.Fatalf("error iterating commits from cache: %v", err)
	}

	if len(commits) != 2 {
		t.Errorf(
			"expected to get two commits from cache, but got %d",
			len(commits),
		)
	}
}
