/*
* Wraps access to data needed from Git.
*
* We invoke Git directly as a subprocess and parse the output rather than using
* git2go/libgit2.
 */
package git

import (
	"context"
	"fmt"
	"io"
	"iter"
	"strings"
	"time"
)

type Commit struct {
	Hash        string
	ShortHash   string
	IsMerge     bool
	AuthorName  string
	AuthorEmail string
	Date        time.Time
	FileDiffs   []FileDiff
}

func (c Commit) Name() string {
	if c.ShortHash != "" {
		return c.ShortHash
	} else if c.Hash != "" {
		return c.Hash
	} else {
		return "unknown"
	}
}

func (c Commit) String() string {
	return fmt.Sprintf(
		"{ hash:%s author:%s <%s> date:%s merge:%v }",
		c.Name(),
		c.AuthorName,
		c.AuthorEmail,
		c.Date.Format("Jan 2, 2006"),
		c.IsMerge,
	)
}

// A file that was changed in a Commit.
type FileDiff struct {
	Path         string
	LinesAdded   int
	LinesRemoved int
}

func (d FileDiff) String() string {
	return fmt.Sprintf(
		"{ path:\"%s\" added:%d removed:%d }",
		d.Path,
		d.LinesAdded,
		d.LinesRemoved,
	)
}

// Returns an iterator over commits identified by the given revisions and paths.
//
// Also returns a closer() function for cleanup and an error when encountered.
func CommitsWithOpts(
	ctx context.Context,
	revs []string,
	pathspecs []string,
	filters LogFilters,
	populateDiffs bool,
	repoFiles RepoConfigFiles,
) (
	iter.Seq[Commit],
	func() error,
	error,
) {
	ignoreRevs, err := repoFiles.IgnoreRevs()
	if err != nil {
		return nil, nil, err
	}

	subprocess, err := RunLog(
		ctx,
		revs,
		pathspecs,
		filters,
		populateDiffs,
		repoFiles.HasMailmap(),
	)
	if err != nil {
		return nil, nil, err
	}

	lines, finishLines := subprocess.StdoutNullDelimitedLines()
	commits, finishCommits := ParseCommits(lines)
	commits = SkipIgnored(commits, ignoreRevs)

	finish := func() error {
		err = finishLines()
		if err != nil {
			return err
		}

		err = finishCommits()
		if err != nil {
			return err
		}

		return subprocess.Wait()
	}

	return commits, finish, nil
}

func RevList(
	ctx context.Context,
	revranges []string,
	pathspecs []string,
	filters LogFilters,
) (_ []string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error getting full rev list: %w", err)
		}
	}()

	revs := []string{}

	subprocess, err := RunRevList(ctx, revranges, pathspecs, filters)
	if err != nil {
		return revs, err
	}

	lines, finish := subprocess.StdoutLines()
	for line := range lines {
		revs = append(revs, line)
	}

	err = finish()
	if err != nil {
		return revs, err
	}

	err = subprocess.Wait()
	if err != nil {
		return revs, err
	}

	return revs, nil
}

func GetRoot() (_ string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf(
				"failed to run git rev-parse --show-toplevel: %w",
				err,
			)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	args := []string{"rev-parse", "--show-toplevel"}
	subprocess, err := run(ctx, args, false)
	if err != nil {
		return "", err
	}

	b, err := io.ReadAll(subprocess.stdout)
	if err != nil {
		return "", err
	}

	err = subprocess.Wait()
	if err != nil {
		return "", err
	}

	root := strings.TrimSpace(string(b))
	return root, nil
}

// Returns all paths in the working tree under the given pathspecs.
func WorkingTreeFiles(pathspecs []string) (_ map[string]bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error getting tree files: %w", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wtreeset := map[string]bool{}

	subprocess, err := RunLsFiles(ctx, pathspecs)
	if err != nil {
		return wtreeset, err
	}

	lines, finish := subprocess.StdoutNullDelimitedLines()
	for line := range lines {
		wtreeset[line] = true
	}

	err = finish()
	if err != nil {
		return wtreeset, err
	}

	err = subprocess.Wait()
	if err != nil {
		return wtreeset, err
	}

	return wtreeset, nil
}

// Returns all commits in the input iterator, but for each commit, strips out
// any file diff not modifying one of the given pathspecs
func LimitDiffsByPathspec(
	commits iter.Seq[Commit],
	pathspecs []string,
) (iter.Seq[Commit], error) {
	if len(pathspecs) == 0 {
		return commits, nil
	}

	// Check all pathspecs are supported
	for _, p := range pathspecs {
		if !IsSupportedPathspec(p) {
			err := fmt.Errorf("unsupported magic in pathspec: \"%s\"", p)
			return commits, err
		}
	}

	return func(yield func(Commit) bool) {
		includes, excludes := SplitPathspecs(pathspecs)

		for commit := range commits {
			filtered := []FileDiff{}
			for _, diff := range commit.FileDiffs {
				shouldInclude := false
				for _, p := range includes {
					if PathspecMatch(p, diff.Path) {
						shouldInclude = true
						break
					}
				}

				shouldExclude := false
				for _, p := range excludes {
					if PathspecMatch(p, diff.Path) {
						shouldExclude = true
						break
					}
				}

				if shouldInclude && !shouldExclude {
					filtered = append(filtered, diff)
				}
			}

			commit.FileDiffs = filtered
			if !yield(commit) {
				return
			}
		}
	}, nil
}

// Returns an iterator over commits that skips any revs in the given list.
func SkipIgnored(
	commits iter.Seq[Commit],
	ignoreRevs []string,
) iter.Seq[Commit] {
	ignoreSet := map[string]bool{}
	for _, rev := range ignoreRevs {
		ignoreSet[rev] = true
	}

	return func(yield func(Commit) bool) {
		for commit := range commits {
			if shouldIgnore := ignoreSet[commit.Hash]; shouldIgnore {
				continue // skip this commit
			}

			if !yield(commit) {
				break
			}
		}
	}
}
