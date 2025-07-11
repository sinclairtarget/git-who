package git_test

import (
	"context"
	"slices"
	"testing"

	"github.com/sinclairtarget/git-who/internal/git"
)

func TestCommitsFileRename(t *testing.T) {
	path := "file-rename"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seq, finish := git.CommitsWithOpts(
		ctx,
		[]string{"HEAD"},
		[]string{path},
		git.LogFilters{},
		true,
		git.RepoConfigFiles{},
	)
	commits := slices.Collect(seq)
	err := finish()
	if err != nil {
		t.Errorf("error iterating commits: %v", err)
	}

	if len(commits) != 3 {
		t.Fatalf("expected 3 commits but found %d", len(commits))
	}

	commit := commits[1]
	if commit.Hash != "879e94bbbcbbec348ba1df332dd46e7314c62df1" {
		t.Errorf(
			"expected commit to have hash %s but got %s",
			"879e94bbbcbbec348ba1df332dd46e7314c62df1",
			commit.Hash,
		)
	}

	if len(commit.FileDiffs) != 1 {
		t.Errorf(
			"len of commit file diffs should be 1, but got %d",
			len(commit.FileDiffs),
		)
	}

	diff := commit.FileDiffs[0]
	if diff.Path != "file-rename/bim.go" {
		t.Errorf(
			"expected diff path to be %s but got \"%s\"",
			"file-rename/bim.go",
			diff.Path,
		)
	}
}

// Test moving a file into a new directory (to make sure we handle { => foo})
func TestCommitsFileRenameNewDir(t *testing.T) {
	path := "rename-new-dir"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seq, finish := git.CommitsWithOpts(
		ctx,
		[]string{"HEAD"},
		[]string{path},
		git.LogFilters{},
		true,
		git.RepoConfigFiles{},
	)
	commits := slices.Collect(seq)
	err := finish()
	if err != nil {
		t.Errorf("error iterating commits: %v", err)
	}

	if len(commits) != 2 {
		t.Fatalf("expected 2 commits but found %d", len(commits))
	}

	commit := commits[1]
	if commit.Hash != "13b6f4f70c682ab06da9ef433cdb4fcbf65d78c3" {
		t.Errorf(
			"expected commit to have hash %s but got %s",
			"13b6f4f70c682ab06da9ef433cdb4fcbf65d78c3",
			commit.Hash,
		)
	}

	if len(commit.FileDiffs) != 1 {
		t.Errorf(
			"len of commit file diffs should be 1, but got %d",
			len(commit.FileDiffs),
		)
	}

	diff := commit.FileDiffs[0]
	if diff.Path != "rename-new-dir/foo/hello.txt" {
		t.Errorf(
			"expected diff path to be %s but got \"%s\"",
			"rename-new-dir/foo/hello.txt",
			diff.Path,
		)
	}
}

// Test moving where change will look like /foo/{bim/bar => baz/biz}/hello.txt
func TestCommitsRenameDeepDir(t *testing.T) {
	path := "rename-across-deep-dirs"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seq, finish := git.CommitsWithOpts(
		ctx,
		[]string{"HEAD"},
		[]string{path},
		git.LogFilters{},
		true,
		git.RepoConfigFiles{},
	)
	commits := slices.Collect(seq)
	err := finish()
	if err != nil {
		t.Errorf("error iterating commits: %v", err)
	}

	if len(commits) != 2 {
		t.Fatalf("expected 2 commits but found %d", len(commits))
	}

	commit := commits[1]
	if commit.Hash != "b9acb309a2c20ab6b93549bc7468b3e3ae5fc05e" {
		t.Errorf(
			"expected commit to have hash %s but got %s",
			"b9acb309a2c20ab6b93549bc7468b3e3ae5fc05e",
			commit.Hash,
		)
	}

	if len(commit.FileDiffs) != 1 {
		t.Errorf(
			"len of commit file diffs should be 1, but got %d",
			len(commit.FileDiffs),
		)
	}

	diff := commit.FileDiffs[0]
	if diff.Path != "rename-across-deep-dirs/zim/zam/hello.txt" {
		t.Errorf(
			"expected diff path to be %s but got \"%s\"",
			"rename-across-deep-dirs/zim/zam/hello.txt",
			diff.Path,
		)
	}
}

func TestParseWholeLog(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, finish := git.CommitsWithOpts(
		ctx,
		[]string{"HEAD"},
		[]string{"."},
		git.LogFilters{},
		true,
		git.RepoConfigFiles{},
	)
	err := finish()
	if err != nil {
		t.Errorf("error iterating commits: %v", err)
	}
}
