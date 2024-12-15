// Handles summations over commits.
package tally

import (
	"fmt"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/itererr"
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

func TallyCommits(commits *itererr.Iter[git.Commit]) (map[string]Tally, error) {
	tallies := make(map[string]Tally)
	filesets := make(map[string]map[string]bool) // Used to dedupe filepaths

	start := time.Now()

	for commit := range commits.Seq {
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

	if commits.Err != nil {
		return nil, fmt.Errorf("error iterating commits: %w", commits.Err)
	}

	// Get count of unique files touched
	for key, tally := range tallies {
		fileset := filesets[key]
		tally.FileCount = len(fileset)
		tallies[key] = tally
	}

	elapsed := time.Now().Sub(start)
	logger().Debug("tallied commits", "duration_ms", elapsed.Milliseconds())

	return tallies, nil
}
