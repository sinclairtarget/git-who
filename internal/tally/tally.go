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
	Mode                 TallyMode
	Key                  func(c git.Commit) string // Unique ID for author
	AllowOutsideWorkTree bool                      // Count edits to paths outside work tree?
}

// Metrics tallied while walking git log
type Tally struct {
	AuthorName     string
	AuthorEmail    string
	Commits        int // Num commits editing paths in tree by this author
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
	treefiles map[string]bool,
	opts TallyOpts,
) ([]Tally, error) {
	// Map of author to tally
	authorTallies := map[string]Tally{}

	// Map of author to map of path to tally. Tally lines per path
	pathTallies := make(map[string]map[string]struct {
		added   int
		removed int
	})

	start := time.Now()

	// Tally over commits
	for commit, err := range commits {
		if err != nil {
			return nil, fmt.Errorf("error iterating commits: %w", err)
		}

		key := opts.Key(commit)
		authorTally := authorTallies[key]

		authorTally.AuthorName = commit.AuthorName
		authorTally.AuthorEmail = commit.AuthorEmail

		_, ok := pathTallies[key]
		if !ok {
			pathTallies[key] = map[string]struct {
				added   int
				removed int
			}{}
		}

		foundWTreePath := false
		for _, diff := range commit.FileDiffs {
			if exists := treefiles[diff.Path]; exists {
				foundWTreePath = true
			}

			pathTally := pathTallies[key][diff.Path]

			if !commit.IsMerge {
				pathTally.added += diff.LinesAdded
				pathTally.removed += diff.LinesRemoved
				pathTallies[key][diff.Path] = pathTally
			}

			// If file move would create a file in the working tree, move it
			// and its existing count of lines added/removed, potentially
			// overwriting.
			destInWTree := treefiles[diff.MoveDest]
			if destInWTree {
				foundWTreePath = true

				for key, _ := range pathTallies {
					pathTally := pathTallies[key][diff.Path]
					delete(pathTallies[key], diff.Path)
					pathTallies[key][diff.MoveDest] = pathTally
				}
			}
		}

		if !commit.IsMerge && (foundWTreePath || opts.AllowOutsideWorkTree) {
			authorTally.Commits += 1
			authorTally.LastCommitTime = commit.Date
			authorTallies[key] = authorTally
		}
	}

	// Handle lines added and file count
	for key, authorTally := range authorTallies {
		for path, pathTally := range pathTallies[key] {
			if exists := treefiles[path]; !exists && !opts.AllowOutsideWorkTree {
				continue
			}

			authorTally.LinesAdded += pathTally.added
			authorTally.LinesRemoved += pathTally.removed
			authorTally.FileCount += 1
		}

		authorTallies[key] = authorTally
	}

	// Sort list
	sorted := sortTallies(authorTallies, opts.Mode)

	elapsed := time.Now().Sub(start)
	logger().Debug("tallied commits", "duration_ms", elapsed.Milliseconds())

	return sorted, nil
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
