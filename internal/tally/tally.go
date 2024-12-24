// Handles summations over commits.
package tally

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

// Whether we rank authors by commit, lines, or files.
type TallyMode int

const (
	CommitMode TallyMode = iota
	LinesMode
	FilesMode
)

type Tally struct {
	AuthorName   string
	AuthorEmail  string
	Commits      int
	LinesAdded   int
	LinesRemoved int
	FileCount    int
}

func (t Tally) SortKey(mode TallyMode) int {
	switch mode {
	case CommitMode:
		return t.Commits
	case FilesMode:
		return t.FileCount
	case LinesMode:
		return t.LinesAdded + t.LinesRemoved
	default:
		panic("unrecognized mode in switch statement")
	}
}

// Returns a slice of tallies in descending order by most commits / files /
// lines.
func TallyCommits(
	commits iter.Seq2[git.Commit, error],
	mode TallyMode,
) ([]Tally, error) {
	tallies := make(map[string]Tally)
	filesets := make(map[string]map[string]bool) // Used to dedupe filepaths

	start := time.Now()

	for commit, err := range commits {
		if err != nil {
			return nil, fmt.Errorf("error iterating commits: %w", err)
		}

		key := commit.AuthorEmail
		tally := tallies[key]

		tally.AuthorName = commit.AuthorName
		tally.AuthorEmail = commit.AuthorEmail
		tally.Commits += 1

		_, ok := filesets[key]
		if !ok {
			filesets[key] = make(map[string]bool)
		}
		for _, diff := range commit.FileDiffs {
			// TODO: Is the total number of changes really just the sum of
			// changes per commit?
			tally.LinesAdded += diff.LinesAdded
			tally.LinesRemoved += diff.LinesRemoved

			filesets[key][diff.Path] = true
		}

		tallies[key] = tally
	}

	// Get count of unique files touched
	for key, tally := range tallies {
		fileset := filesets[key]
		tally.FileCount = len(fileset)
		tallies[key] = tally
	}

	// Sort list
	sorted := slices.SortedFunc(
		maps.Values(tallies),
		func(a, b Tally) int {
			aRank := a.SortKey(mode)
			bRank := b.SortKey(mode)

			if aRank > bRank {
				return -1
			} else if aRank == bRank {
				return 0
			} else {
				return 1
			}
		},
	)

	elapsed := time.Now().Sub(start)
	logger().Debug("tallied commits", "duration_ms", elapsed.Milliseconds())

	return sorted, nil
}
