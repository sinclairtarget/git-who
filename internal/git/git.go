/*
* Wraps access to data needed from Git.
*
* We invoke Git directly as a subprocess and parse the output rather than using
* git2go/libgit2.
 */
package git

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"slices"
	"time"

	"github.com/sinclairtarget/git-who/internal/git/cmd"
	"github.com/sinclairtarget/git-who/internal/git/config"
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

// Returns a single-use iterator over commits identified by the given revisions
// and paths.
func CommitsWithOpts(
	ctx context.Context,
	revs []string,
	pathspecs []string,
	filters cmd.LogFilters,
	populateDiffs bool,
	configFiles config.SupplementalFiles,
) (
	iter.Seq[Commit],
	func() error,
) {
	empty := slices.Values([]Commit{})

	ignoreRevs, err := configFiles.IgnoreRevs()
	if err != nil {
		return empty, func() error { return err }
	}

	subprocess, err := cmd.RunLog(
		ctx,
		revs,
		pathspecs,
		filters,
		populateDiffs,
		configFiles.HasMailmap(),
	)
	if err != nil {
		return empty, func() error { return err }
	}

	lines, finishLines := subprocess.StdoutNullDelimitedLines()
	commits, finishCommits := ParseCommits(lines)
	commits = SkipIgnored(commits, ignoreRevs)

	finish := func() error {
		iterErr := finishCommits()
		iterErr = errors.Join(iterErr, finishLines())
		if iterErr != nil {
			return fmt.Errorf("error iterating commits: %v", iterErr)
		}

		return subprocess.Wait()
	}

	return commits, finish
}

func RevList(
	ctx context.Context,
	revranges []string,
	pathspecs []string,
	filters cmd.LogFilters,
) (_ []string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error getting full rev list: %w", err)
		}
	}()

	revs := []string{}

	subprocess, err := cmd.RunRevList(ctx, revranges, pathspecs, filters)
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
			err = fmt.Errorf("failed to get Git root directory: %w", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subprocess, err := cmd.RunRevParseTopLevel(ctx)
	if err != nil {
		return "", err
	}

	root, err := subprocess.StdoutText()
	if err != nil {
		return "", err
	}

	err = subprocess.Wait()
	if err != nil {
		return "", err
	}

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

	subprocess, err := cmd.RunLsFiles(ctx, pathspecs)
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
				// If we don't have any explicit includes, then everything is
				// included.
				shouldInclude := len(includes) == 0
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
