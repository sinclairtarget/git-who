package git

import (
	"fmt"
	"iter"
	"maps"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var fileRenameRegexp *regexp.Regexp
var commitHashRegexp *regexp.Regexp

func init() {
	fileRenameRegexp = regexp.MustCompile(`{(.*) => (.*)}`)
	commitHashRegexp = regexp.MustCompile(`^[\^a-f0-9]+$`)
}

// Splits a path from git log --numstat on "/", while ignoring "/" surrounded
// by "{" and "}".
func splitPath(path string) []string {
	parts := []string{}
	var b strings.Builder
	var inBrackets bool

	for _, c := range path {
		if c == os.PathSeparator && !inBrackets {
			parts = append(parts, b.String())
			b.Reset()
			continue
		}

		if c == '{' {
			inBrackets = true
		} else if c == '}' {
			inBrackets = false
		}

		b.WriteRune(c)
	}

	if b.Len() > 0 {
		parts = append(parts, b.String())
	}

	return parts
}

// Parse the path given by git log --numstat for a file diff.
//
// Sometimes this looks like /foo/{bar => bim}/baz.txt when a file is moved.
func parseDiffPath(path string) (outPath string, dst string, err error) {
	if strings.Contains(path, "=>") && !strings.Contains(path, "}") {
		// Simple case
		parts := strings.Split(path, " => ")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("error parsing diff path from \"%s\" path", path)
		}
		outPath = parts[0]
		dst = parts[1]
		return outPath, dst, nil
	}

	var pathBuilder strings.Builder
	var dstBuilder strings.Builder

	parts := splitPath(path)
	for i, part := range parts {
		if strings.Contains(part, "=>") {
			matches := fileRenameRegexp.FindStringSubmatch(part)
			if matches == nil || len(matches) != 3 {
				return "", "", fmt.Errorf(
					"error parsing rename from \"%s\" in path \"%s\"",
					part,
					path,
				)
			}

			pathBuilder.WriteString(matches[1])
			dstBuilder.WriteString(matches[2])

			if i < len(parts)-1 {
				if matches[1] != "" {
					pathBuilder.WriteString(string(os.PathSeparator))
				}
				if matches[2] != "" {
					dstBuilder.WriteString(string(os.PathSeparator))
				}
			}
		} else {
			pathBuilder.WriteString(part)
			dstBuilder.WriteString(part)

			if i < len(parts)-1 {
				pathBuilder.WriteString(string(os.PathSeparator))
				dstBuilder.WriteString(string(os.PathSeparator))
			}
		}
	}

	outPath = pathBuilder.String()
	dst = dstBuilder.String()
	if dst == outPath {
		dst = ""
	}

	return outPath, dst, nil
}

// e.g. 9       0       rename-across-dirs/foo/bar.txt
func parseFileDiff(line string) (diff FileDiff, err error) {
	parts := strings.Split(line, "\t")
	if len(parts) != 3 {
		return diff, fmt.Errorf("could not parse file diff: %s", line)
	}

	if parts[0] != "-" {
		added, err := strconv.Atoi(parts[0])
		if err != nil {
			return diff,
				fmt.Errorf("could not parse %s as int on line \"%s\": %w",
					parts[0],
					line,
					err,
				)
		}

		diff.LinesAdded = added
	}

	if parts[1] != "-" {
		removed, err := strconv.Atoi(parts[1])
		if err != nil {
			return diff,
				fmt.Errorf("could not parse %s as int on line \"%s\": %w",
					parts[1],
					line,
					err,
				)
		}
		diff.LinesRemoved = removed
	}

	path, moveDest, err := parseDiffPath(parts[2])
	if err != nil {
		return diff, fmt.Errorf(
			"could not parse path part of file diff on line \"%s\": %w",
			line,
			err,
		)
	}

	if len(moveDest) > 0 {
		// If file is moved, attribute diff to the new path
		diff.Path = moveDest
	} else {
		diff.Path = path
	}

	return diff, nil
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
		diffLookup := map[string]FileDiff{}
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

			done := linesThisCommit >= 7 && (len(line) == 0 || isRev(line))
			if done {
				commit.FileDiffs = slices.Collect(maps.Values(diffLookup))
				if allowCommit(commit, now) {
					if !yield(commit, nil) {
						return
					}
				}

				commit = Commit{}
				diffLookup = map[string]FileDiff{}
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
				diff, err := parseFileDiff(line)
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
				diffLookup[diff.Path] = diff
			}

			linesThisCommit += 1
		}

		if linesThisCommit > 0 && allowCommit(commit, now) {
			commit.FileDiffs = slices.Collect(maps.Values(diffLookup))
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
