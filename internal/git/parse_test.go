package git_test

import (
	"iter"
	"slices"
	"strings"
	"testing"

	"github.com/sinclairtarget/git-who/internal/git"
)

const fileRenameDump = `bf4136de996e9fb1f38620350cb7185613d71193
bf4136d
6afef28
Sinclair Target
sinclairtarget@gmail.com
1735304504
9	0	file-rename/foo.go

879e94bbbcbbec348ba1df332dd46e7314c62df1
879e94b
bf4136d
Sinclair Target
sinclairtarget@gmail.com
1735304522
0	0
file-rename/foo.go
file-rename/bim.go

ad6d3789cf56b4a8ae3f8632d43fa65f2ec823a0
ad6d378
879e94b
Sinclair Target
sinclairtarget@gmail.com
1735304546
1	1	file-rename/bim.go

`

const renameNewDirDump = `7f62cecd2b889b91828db026ba7c4314de1e8f3a
7f62cec
e4b688d
Sinclair Target
sinclairtarget@gmail.com
1735487061
1	0	rename-new-dir/hello.txt

13b6f4f70c682ab06da9ef433cdb4fcbf65d78c3
13b6f4f
7f62cec
Sinclair Target
sinclairtarget@gmail.com
1735487089
0	0
rename-new-dir/hello.txt
rename-new-dir/foo/hello.txt

`

const renameDeepDirDump = `5def9bfbddde001f6f324f2b781b6f2144bc3662
5def9bf
13b6f4f
Sinclair Target
sinclairtarget@gmail.com
1735507602
1	0	rename-across-deep-dirs/foo/bar/hello.txt

b9acb309a2c20ab6b93549bc7468b3e3ae5fc05e
b9acb30
5def9bf
Sinclair Target
sinclairtarget@gmail.com
1735507662
0	0
rename-across-deep-dirs/foo/bar/hello.txt
rename-across-deep-dirs/zim/zam/hello.txt

`

func readDump(dump string) iter.Seq[string] {
	return slices.Values(strings.Split(dump, "\n"))
}

func TestParseFileRename(t *testing.T) {
	lines := readDump(fileRenameDump)

	seq, finish := git.ParseCommits(lines)
	commits := slices.Collect(seq)
	err := finish()
	if err != nil {
		t.Fatalf("error iterating commits: %v", err)
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

// Test moving a file into a new directory
func TestCommitsFileRenameNewDir(t *testing.T) {
	lines := readDump(renameNewDirDump)

	seq, finish := git.ParseCommits(lines)
	commits := slices.Collect(seq)
	err := finish()
	if err != nil {
		t.Fatalf("error iterating commits: %v", err)
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
	lines := readDump(renameDeepDirDump)

	seq, finish := git.ParseCommits(lines)
	commits := slices.Collect(seq)
	err := finish()
	if err != nil {
		t.Fatalf("error iterating commits: %v", err)
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
