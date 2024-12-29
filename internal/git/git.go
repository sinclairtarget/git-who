/*
* Wraps access to data needed from Git.
*
* We invoke Git directly as a subprocess and parse the output rather than using
* git2go/libgit2.
 */
package git

import (
	"fmt"
	"iter"
	"time"
)

type Commit struct {
	Hash        string
	ShortHash   string
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
		"{ hash:%s author:%s <%s> date:%s subject:%s }",
		c.Name(),
		c.AuthorName,
		c.AuthorEmail,
		c.Date.Format("Jan 2, 2006"),
		c.Subject,
	)
}

// A file that was changed in a Commit.
type FileDiff struct {
	Path         string
	LinesAdded   int
	LinesRemoved int
	MoveDest     string // Empty unless the file was renamed
}

func (d FileDiff) String() string {
	return fmt.Sprintf(
		"{ path:\"%s\" move:\"%s\" added:%d removed:%d }",
		d.Path,
		d.MoveDest,
		d.LinesAdded,
		d.LinesRemoved,
	)
}

// Returns an iterator over commits identified by the given revisions and paths.
//
// Also returns a closer() function for cleanup and an error when encountered.
func Commits(revs []string, paths []string) (
	iter.Seq2[Commit, error],
	func() error,
	error,
) {
	subprocess, err := RunLog(revs, paths)
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
