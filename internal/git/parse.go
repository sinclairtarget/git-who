package git

import (
	"fmt"
	"iter"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var fileRenameRegexp *regexp.Regexp
var commitHashRegexp *regexp.Regexp

func init() {
	fileRenameRegexp = regexp.MustCompile(`{(.*) => (.*)}`)
	commitHashRegexp = regexp.MustCompile(`^\^?[a-f0-9]+$`)
}

func parseLinesChanged(s string, line string) (int, error) {
	changed, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("could not parse %s as int on line \"%s\": %w",
			s,
			line,
			err,
		)
	}

	return changed, nil
}

func allowCommit(commit Commit, now time.Time) bool {
	if commit.AuthorName == "" && commit.AuthorEmail == "" {
		logger().Debug(
			"skipping commit with no author",
			"commit",
			commit.Name(),
		)

		return false
	}

	if commit.Date.After(now) {
		logger().Debug(
			"skipping commit with commit date in the future",
			"commit",
			commit.Name(),
		)

		return false
	}

	return true
}

// Turns an iterator over lines from git log into an iterator of commits
func ParseCommits(lines iter.Seq[string]) (iter.Seq[Commit], func() error) {
	var iterErr error

	seq := func(yield func(Commit) bool) {
		var commit Commit
		var diff *FileDiff
		now := time.Now()
		linesThisCommit := 0

		for line := range lines {
			done := linesThisCommit >= 6 && (len(line) == 0 || isRev(line))
			if done {
				if allowCommit(commit, now) {
					if !yield(commit) {
						return
					}
				}

				commit = Commit{}
				diff = nil
				linesThisCommit = 0

				if len(line) == 0 {
					continue
				}
			}

			switch {
			case linesThisCommit == 0:
				commit.Hash = line
			case linesThisCommit == 1:
				commit.ShortHash = line
			case linesThisCommit == 2:
				parts := strings.Split(line, " ")
				commit.IsMerge = len(parts) > 1
			case linesThisCommit == 3:
				commit.AuthorName = line
			case linesThisCommit == 4:
				commit.AuthorEmail = line
			case linesThisCommit == 5:
				i, err := strconv.Atoi(line)
				if err != nil {
					iterErr = fmt.Errorf(
						"error parsing date from commit %s: %w",
						commit.Name(),
						err,
					)
					return
				}

				commit.Date = time.Unix(int64(i), 0)
			default:
				var err error

				// Handle file diffs
				if diff == nil {
					diff = &FileDiff{}

					// Split to get non-empty tokens
					parts := strings.SplitN(line, "\t", 3)
					nonemptyParts := []string{}
					for _, p := range parts {
						if len(p) > 0 {
							nonemptyParts = append(nonemptyParts, p)
						}
					}

					if len(nonemptyParts) == 3 {
						if nonemptyParts[0] != "-" {
							diff.LinesAdded, err = parseLinesChanged(
								nonemptyParts[0],
								line,
							)
							if err != nil {
								goto handleError
							}
						}

						if nonemptyParts[1] != "-" {
							diff.LinesRemoved, err = parseLinesChanged(
								nonemptyParts[1],
								line,
							)
							if err != nil {
								goto handleError
							}
						}

						diff.Path = nonemptyParts[2]
						commit.FileDiffs = append(commit.FileDiffs, *diff)
						diff = nil
					} else if len(nonemptyParts) == 2 {
						if nonemptyParts[0] != "-" {
							diff.LinesAdded, err = parseLinesChanged(
								nonemptyParts[0],
								line,
							)
							if err != nil {
								goto handleError
							}
						}

						if nonemptyParts[1] != "-" {
							diff.LinesRemoved, err = parseLinesChanged(
								nonemptyParts[1],
								line,
							)
							if err != nil {
								goto handleError
							}
						}
					} else {
						err = fmt.Errorf(
							"wrong number of elements on line after split: %d",
							len(nonemptyParts),
						)
					}
				} else {
					if len(diff.Path) > 0 {
						diff.Path = line
						commit.FileDiffs = append(commit.FileDiffs, *diff)
						diff = nil
					} else {
						// Used to handle moved files specially. For now, just
						// mark as path until we overwrite it with next line
						diff.Path = line
					}
				}

			handleError:
				if err != nil {
					iterErr = fmt.Errorf(
						"error parsing file diffs from commit %s: %w",
						commit.Name(),
						err,
					)
					return
				}
			}

			linesThisCommit += 1
		}

		if linesThisCommit > 0 && allowCommit(commit, now) {
			yield(commit) // Last yield
		}
	}

	finish := func() error {
		return iterErr
	}

	return seq, finish
}

// Returns true if this is a (full-length) Git revision hash, false otherwise.
//
// We also need to handle a hash with "^" in front.
func isRev(s string) bool {
	matched := commitHashRegexp.MatchString(s)
	return matched && (len(s) == 40 || len(s) == 41)
}
