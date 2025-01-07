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
	"strconv"
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
	Subject     string
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
		"{ hash:%s author:%s <%s> date:%s subject:%s merge:%v }",
		c.Name(),
		c.AuthorName,
		c.AuthorEmail,
		c.Date.Format("Jan 2, 2006"),
		c.Subject,
		c.IsMerge,
	)
}

type FileAction int

const (
	NoAction FileAction = iota
	Create
	Delete
	Rename
)

// A file that was changed in a Commit.
type FileDiff struct {
	Path         string
	Action       FileAction
	LinesAdded   int
	LinesRemoved int
	MoveDest     string // Empty unless the file was renamed
}

func (d FileDiff) String() string {
	return fmt.Sprintf(
		"{ path:\"%s\" action:%d move:\"%s\" added:%d removed:%d }",
		d.Path,
		d.Action,
		d.MoveDest,
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
	paths []string,
	filters LogFilters,
	populateDiffs bool,
) (
	iter.Seq2[Commit, error],
	func() error,
	error,
) {
	subprocess, err := RunLog(ctx, revs, paths, filters, populateDiffs)
	if err != nil {
		return nil, nil, err
	}

	lines := subprocess.StdoutLines()
	commits := parseCommits(lines)

	closer := func() error {
		return subprocess.Wait()
	}
	return commits, closer, nil
}

func Commits(ctx context.Context, revs []string, paths []string) (
	iter.Seq2[Commit, error],
	func() error,
	error,
) {
	return CommitsWithOpts(ctx, revs, paths, LogFilters{}, true)
}

func NumCommits(
	revs []string,
	paths []string,
	filters LogFilters,
) (_ int, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error getting commit count: %w", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subprocess, err := RunRevList(ctx, revs, paths, filters, true)
	if err != nil {
		return 0, err
	}

	lines := subprocess.StdoutLines()
	next, stop := iter.Pull2(lines)
	defer stop()

	line, err, ok := next()
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, errors.New("no output from git rev-list")
	}

	count, err := strconv.Atoi(line)
	if err != nil {
		return 0, err
	}

	subprocess.Wait()
	return count, nil
}

// Returns all paths in the working tree under the given paths.
func WorkingTreeFiles(paths []string) (_ map[string]bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error gettign tree files: %w", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wtreeset := map[string]bool{}

	subprocess, err := RunLsFiles(ctx, paths)
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

	subprocess.Wait()
	return wtreeset, nil
}
