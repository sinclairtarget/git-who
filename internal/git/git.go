/*
* Wraps access to data needed from Git.
*
* We invoke Git directly as a subprocess and parse the output rather than using
* git2go/libgit2.
*/
package git

import (
	"bufio"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sinclairtarget/git-who/internal/itererr"
)

// A file that was changed in a Commit.
type FileDiff struct {
	Path         string
	LinesAdded   int
	LinesRemoved int
}

type Commit struct {
	Hash         string
	ShortHash    string
	AuthorName   string
	AuthorEmail  string
	Date         time.Time
	Subject      string
	FileDiffs    []FileDiff
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

// Arguments used for `git log`
var baseArgs = []string {
	"log",
	"--pretty=format:%H%n%h%n%an%n%ae%n%ad%n%s",
	"--numstat",
	"--date=unix",
}

// Runs git log and returns an iterator over each line of the output
func LogLines(revs []string, path string) (*itererr.Iter[string], error) {
	args := slices.Concat(baseArgs, revs, []string { path })

	cmd := exec.Command("git", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start subprocess: %w", err)
	}

	scanner := bufio.NewScanner(stdout)

	lines := itererr.Iter[string] {}
	lines.Seq = func(yield func(string) bool) {
		for scanner.Scan() {
			if !yield(scanner.Text()) {
				break
			}
		}

		err := scanner.Err()
		if err != nil {
			lines.Err = fmt.Errorf("failed to scan stdout: %w", err)
		}

		err = cmd.Wait()
		if err != nil && lines.Err == nil {
			// TODO: Can we log stderr as well here to help diagnose?
			lines.Err = fmt.Errorf(
				"error after waiting for subprocess: %w",
				err,
			)
		}
	}

	return &lines, nil
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
func ParseCommits(lines *itererr.Iter[string]) *itererr.Iter[Commit] {
	commits := itererr.Iter[Commit] {}

	commits.Seq = func(yield func(Commit) bool) {
		commit := Commit { FileDiffs: make([]FileDiff, 0) }
		linesThisCommit := 0

		for line := range lines.Seq {
			if len(line) == 0 {
				if !yield(commit) {
					break
				}

				commit = Commit { FileDiffs: make([]FileDiff, 0) }
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
					commits.Err = fmt.Errorf("error parsing commits: %w", err)
					return
				}

				commit.Date = time.Unix(int64(i), 0)
			case linesThisCommit == 5:
				commit.Subject = line
			case linesThisCommit >= 6:
				diff, err := parseFileDiff(line)
				if err != nil {
					commits.Err = err
					return
				}
				commit.FileDiffs = append(commit.FileDiffs, diff)
			}

			linesThisCommit += 1
		}

		if linesThisCommit > 0 {
			yield(commit)
		}

		if lines.Err != nil {
			commits.Err = fmt.Errorf("error parsing commits: %w", lines.Err)
		}
	}

	return &commits
}
