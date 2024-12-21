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
	"strconv"
	"strings"
	"time"
)

// A file that was changed in a Commit.
type FileDiff struct {
	Path         string
	LinesAdded   int
	LinesRemoved int
}

type Commit struct {
	Hash        string
	ShortHash   string
	AuthorName  string
	AuthorEmail string
	Date        time.Time
	Subject     string
	FileDiffs   []FileDiff
}

func (c Commit) String() string {
	return fmt.Sprintf(
		"{ hash:%s email:%s date:%s files:%d }",
		c.ShortHash,
		c.AuthorEmail,
		c.Date,
		len(c.FileDiffs),
	)
}

func parseFileDiff(line string) (FileDiff, error) {
	var diff FileDiff

	parts := strings.Split(line, "\t")
	if len(parts) != 3 {
		return diff, fmt.Errorf("could not parse file diff: %s", line)
	}

	added, err := strconv.Atoi(parts[0])
	if err != nil {
		return diff,
			fmt.Errorf("could not parse %s as int: %w", parts[0], err)
	}
	diff.LinesAdded = added

	removed, err := strconv.Atoi(parts[1])
	if err != nil {
		return diff,
			fmt.Errorf("could not parse %s as int: %w", parts[1], err)
	}
	diff.LinesRemoved = removed

	diff.Path = parts[2]
	return diff, nil
}

// Turns an iterator over lines from git log into an iterator of commits
func parseCommits(lines iter.Seq2[string, error]) iter.Seq2[Commit, error] {
	return func(yield func(Commit, error) bool) {
		commit := Commit{FileDiffs: make([]FileDiff, 0)}
		linesThisCommit := 0

		for line, err := range lines {
			if err != nil {
				yield(commit, fmt.Errorf("error parsing commits: %w", err))
				return
			}

			if len(line) == 0 {
				logger().Debug(
					"yielding parsed commit",
					"hash",
					commit.ShortHash,
				)
				if !yield(commit, nil) {
					return
				}

				commit = Commit{FileDiffs: make([]FileDiff, 0)}
				linesThisCommit = 0
				continue
			}

			switch {
			case linesThisCommit == 0:
				commit.Hash = line
			case linesThisCommit == 1:
				commit.ShortHash = line
			case linesThisCommit == 2:
				commit.AuthorName = line
			case linesThisCommit == 3:
				commit.AuthorEmail = line
			case linesThisCommit == 4:
				i, err := strconv.Atoi(line)
				if err != nil {
					yield(commit, fmt.Errorf("error parsing commits: %w", err))
					return
				}

				commit.Date = time.Unix(int64(i), 0)
			case linesThisCommit == 5:
				commit.Subject = line
			case linesThisCommit >= 6:
				diff, err := parseFileDiff(line)
				if err != nil {
					yield(commit, err)
					return
				}
				commit.FileDiffs = append(commit.FileDiffs, diff)
			}

			linesThisCommit += 1
		}

		if linesThisCommit > 0 {
			logger().Debug("yielding parsed commit", "hash", commit.ShortHash)
			yield(commit, nil)
		}
	}
}

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
