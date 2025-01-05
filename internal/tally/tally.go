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
	LastModifiedMode
)

type TallyOpts struct {
	Mode TallyMode
	Key  func(c git.Commit) string
}

// Metrics tallied while walking git log
type Tally struct {
	AuthorName     string
	AuthorEmail    string
	Commits        int // Num reachable commits by this author
	LinesAdded     int // Num lines added to paths in tree by author
	LinesRemoved   int // Num lines deleted from paths in tree by author
	FileCount      int // Num of file paths in working dir touched by author
	LastCommitTime time.Time
}

func (t Tally) SortKey(mode TallyMode) int64 {
	switch mode {
	case CommitMode:
		return int64(t.Commits)
	case FilesMode:
		return int64(t.FileCount)
	case LinesMode:
		return int64(t.LinesAdded + t.LinesRemoved)
	case LastModifiedMode:
		return t.LastCommitTime.Unix()
	default:
		panic("unrecognized mode in switch statement")
	}
}

func (a Tally) Compare(b Tally, mode TallyMode) int {
	aRank := a.SortKey(mode)
	bRank := b.SortKey(mode)

	if aRank < bRank {
		return -1
	} else if bRank < aRank {
		return 1
	}

	// Break ties with last edited
	return a.LastCommitTime.Compare(b.LastCommitTime)
}

// Returns a slice of tallies, each one for a different author, in descending
// order by most commits / files / lines (depending on the tally mode).
func TallyCommits(
	commits iter.Seq2[git.Commit, error],
	opts TallyOpts,
) ([]Tally, error) {
	// Map of author to tally
	tallies := map[string]Tally{}

	// Map of author to fileset. Used to dedupe filepaths
	filesets := make(map[string]map[string]bool)

	start := time.Now()

	for commit, err := range commits {
		if err != nil {
			return nil, fmt.Errorf("error iterating commits: %w", err)
		}

		key := opts.Key(commit)
		tally := tallies[key]

		tally.AuthorName = commit.AuthorName
		tally.AuthorEmail = commit.AuthorEmail
		tally.Commits += 1
		tally.LastCommitTime = commit.Date

		_, ok := filesets[key]
		if !ok {
			filesets[key] = make(map[string]bool)
		}
		for _, diff := range commit.FileDiffs {
			tally.LinesAdded += diff.LinesAdded
			tally.LinesRemoved += diff.LinesRemoved
			if diff.MoveDest != "" {
				moveFile(filesets, diff)
			} else {
				filesets[key][diff.Path] = true
			}
		}

		tallies[key] = tally
	}

	// Get count of unique files touched
	for key, tally := range tallies {
		fileset := filesets[key]
		tally.FileCount = countFiles(fileset)
		tallies[key] = tally
	}

	// Sort list
	sorted := sortTallies(tallies, opts.Mode)

	elapsed := time.Now().Sub(start)
	logger().Debug("tallied commits", "duration_ms", elapsed.Milliseconds())

	return sorted, nil
}

func moveFile(filesets map[string]map[string]bool, diff git.FileDiff) {
	// File rename for everyone
	for author, _ := range filesets {
		filesets[author][diff.Path] = false
		filesets[author][diff.MoveDest] = true
	}
}

func countFiles(fileset map[string]bool) int {
	sum := 0
	for _, exists := range fileset {
		if exists {
			sum += 1
		}
	}

	return sum
}

func sortTallies(tallies map[string]Tally, mode TallyMode) []Tally {
	sorted := slices.SortedFunc(maps.Values(tallies), func(a, b Tally) int {
		return -a.Compare(b, mode)
	})

	return sorted
}
