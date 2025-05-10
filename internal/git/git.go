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
	iter.Seq2[Commit, error],
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
	)
	if err != nil {
		return nil, nil, err
	}

	lines := subprocess.StdoutLogLines()
	commits := ParseCommits(lines)
	commits = SkipIgnored(commits, ignoreRevs)

	closer := func() error {
		return subprocess.Wait()
	}
	return commits, closer, nil
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

	lines := subprocess.StdoutLines()
	for line, err := range lines {
		if err != nil {
			return revs, err
		}

		revs = append(revs, line)
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

	lines := subprocess.StdoutLines()
	for line, err := range lines {
		if err != nil {
			return wtreeset, err
		}
		wtreeset[strings.TrimSpace(line)] = true
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
	commits iter.Seq2[Commit, error],
	pathspecs []string,
) iter.Seq2[Commit, error] {
	if len(pathspecs) == 0 {
		return commits
	}

	return func(yield func(Commit, error) bool) {
		// Check all pathspecs are supported
		for _, p := range pathspecs {
			if !IsSupportedPathspec(p) {
				yield(
					Commit{},
					fmt.Errorf("unsupported magic in pathspec: \"%s\"", p),
				)
				return
			}
		}

		includes, excludes := SplitPathspecs(pathspecs)

		for commit, err := range commits {
			if err != nil {
				yield(commit, err)
				return
			}

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
			yield(commit, nil)
		}
	}
}

// Returns an iterator over commits that skips any revs in the given list.
func SkipIgnored(
	commits iter.Seq2[Commit, error],
	ignoreRevs []string,
) iter.Seq2[Commit, error] {
	ignoreSet := map[string]bool{}
	for _, rev := range ignoreRevs {
		ignoreSet[rev] = true
	}

	return func(yield func(Commit, error) bool) {
		for commit, err := range commits {
			if err != nil {
				yield(commit, err)
				return
			}

			if shouldIgnore := ignoreSet[commit.Hash]; shouldIgnore {
				continue // skip this commit
			}

			if !yield(commit, nil) {
				break
			}
		}
	}
}
