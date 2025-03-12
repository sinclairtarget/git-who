package git

import (
	"errors"
	"fmt"
	"iter"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var commitHashRegexp *regexp.Regexp

func init() {
	commitHashRegexp = regexp.MustCompile(`^[\^a-f0-9]+$`)
}

func parseLinesChanged(s string, seg string) (int, error) {
	changed, err := strconv.Atoi(s)
	if err != nil {
		return 0,
			fmt.Errorf("could not parse %s as int from \"%s\": %w",
				s,
				seg,
				err,
			)
	}

	return changed, nil
}

func parseFileDiffs(line string) (_ []FileDiff, err error) {
	diffs := []FileDiff{}

	segments := strings.Split(line, "\x00")
	if len(segments) < 1 {
		return diffs, errors.New("not enough file diff segments")
	}

	var diff FileDiff
	for _, seg := range segments {
		if len(seg) == 0 {
			break
		}

		parts := strings.Split(strings.Trim(seg, "\t"), "\t")
		switch len(parts) {
		case 1:
			diff.Path = parts[0]
		case 2:
			if parts[0] != "-" {
				diff.LinesAdded, err = parseLinesChanged(parts[0], seg)
				if err != nil {
					return diffs, err
				}
			}
			if parts[1] != "-" {
				diff.LinesRemoved, err = parseLinesChanged(parts[1], seg)
				if err != nil {
					return diffs, err
				}
			}
		case 3:
			if parts[0] != "-" {
				diff.LinesAdded, err = parseLinesChanged(parts[0], seg)
				if err != nil {
					return diffs, err
				}
			}
			if parts[1] != "-" {
				diff.LinesRemoved, err = parseLinesChanged(parts[1], seg)
				if err != nil {
					return diffs, err
				}
			}
			diff.Path = parts[2]
			diffs = append(diffs, diff)
			diff = FileDiff{}
		default:
			return diffs, fmt.Errorf("could not parse file diff: %s", seg)
		}
	}

	if len(diff.Path) > 0 {
		diffs = append(diffs, diff)
	}

	return diffs, nil
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
func ParseCommits(lines iter.Seq2[string, error]) iter.Seq2[Commit, error] {
	return func(yield func(Commit, error) bool) {
		var commit Commit
		now := time.Now()
		linesThisCommit := 0

		for line, err := range lines {
			if err != nil {
				yield(
					commit,
					fmt.Errorf(
						"error reading commit %s: %w",
						commit.Name(),
						err,
					),
				)
				return
			}

			if linesThisCommit > 6 && len(line) == 0 {
				continue
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
					yield(
						commit,
						fmt.Errorf(
							"error parsing date from commit %s: %w",
							commit.Name(),
							err,
						),
					)
					return
				}

				commit.Date = time.Unix(int64(i), 0)
			case linesThisCommit == 6:
				break // Used to parse subject here; no longer
			default:
				nextHash := ""
				if line[0] == '\x00' {
					nextHash = line[1:]
				} else {
					var err error
					commit.FileDiffs, err = parseFileDiffs(line)
					if err != nil {
						yield(
							commit,
							fmt.Errorf(
								"error parsing file diffs from commit %s: %w",
								commit.Name(),
								err,
							),
						)
						return
					}

					i := strings.Index(line, "\x00\x00")
					if i > 0 {
						nextHash = line[i+2:]
					}
				}

				if allowCommit(commit, now) {
					if !yield(commit, nil) {
						return
					}
				}

				commit = Commit{Hash: nextHash}
				linesThisCommit = 1
				continue
			}

			linesThisCommit += 1
		}

		if linesThisCommit > 0 && allowCommit(commit, now) {
			yield(commit, nil)
		}
	}
}

// Returns true if this is a (full-length) Git revision hash, false otherwise.
//
// We also need to handle a hash with "^" in front.
func isRev(s string) bool {
	matched := commitHashRegexp.MatchString(s)
	return matched && (len(s) == 40 || len(s) == 41)
}
