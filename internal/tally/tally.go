// Handles summations over commits.
package tally

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/timeutils"
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
	Key  func(c git.Commit) string // Unique ID for author
}

// Whether we need --stat and --summary data from git log for this tally mode
func (opts TallyOpts) NeedsDiffs() bool {
	return opts.Mode == FilesMode || opts.Mode == LinesMode
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

// A tally that can be combined with other tallies
type intermediateTally struct {
	commitset      map[string]bool
	added          int
	removed        int
	lastCommitTime time.Time
	numTallied     int
}

func newTally(numTallied int) intermediateTally {
	return intermediateTally{
		commitset:  map[string]bool{},
		numTallied: numTallied,
	}
}

func (t intermediateTally) Commits() int {
	return len(t.commitset)
}

func (a intermediateTally) Add(b intermediateTally) intermediateTally {
	union := a.commitset
	for commit, _ := range b.commitset {
		union[commit] = true
	}

	return intermediateTally{
		commitset:      union,
		added:          a.added + b.added,
		removed:        a.removed + b.removed,
		lastCommitTime: timeutils.Max(a.lastCommitTime, b.lastCommitTime),
		numTallied:     a.numTallied + b.numTallied,
	}
}

// Returns a slice of tallies, each one for a different author, in descending
// order by most commits / files / lines (depending on the tally mode).
func TallyCommits(
	commits iter.Seq2[git.Commit, error],
	wtreefiles map[string]bool,
	allowOutsideWorktree bool,
	opts TallyOpts,
) ([]Tally, error) {
	// Map of author to final tally
	authorTallies := map[string]Tally{}

	start := time.Now()

	if !opts.NeedsDiffs() && allowOutsideWorktree {
		// Just sum over commits
		for commit, err := range commits {
			if err != nil {
				return nil, fmt.Errorf("error iterating commits: %w", err)
			}

			key := opts.Key(commit)

			authorTally := authorTallies[key]
			authorTally.AuthorName = commit.AuthorName
			authorTally.AuthorEmail = commit.AuthorEmail
			authorTally.Commits += 1
			authorTally.LastCommitTime = timeutils.Max(
				commit.Date,
				authorTally.LastCommitTime,
			)

			authorTallies[key] = authorTally
		}
	} else {
		pathTallies, err := tallyByPaths(commits, wtreefiles, opts)
		if err != nil {
			return nil, err
		}

		// Sum over paths
		for key, author := range pathTallies {
			authorTally := authorTallies[key]
			authorTally.AuthorName = author.name
			authorTally.AuthorEmail = author.email

			runningTally := newTally(0)
			for path, pathTally := range author.paths {
				if inWTree := wtreefiles[path]; inWTree || allowOutsideWorktree {
					runningTally = runningTally.Add(pathTally)
				}
			}

			authorTally.Commits = runningTally.Commits()
			authorTally.LinesAdded = runningTally.added
			authorTally.LinesRemoved = runningTally.removed
			authorTally.FileCount = runningTally.numTallied
			authorTally.LastCommitTime = runningTally.lastCommitTime

			authorTallies[key] = authorTally
		}
	}

	// Sort list
	sorted := sortTallies(authorTallies, opts.Mode)

	elapsed := time.Now().Sub(start)
	logger().Debug("tallied commits", "duration_ms", elapsed.Milliseconds())

	return sorted, nil
}

func sortTallies(tallies map[string]Tally, mode TallyMode) []Tally {
	sorted := slices.SortedFunc(maps.Values(tallies), func(a, b Tally) int {
		return -a.Compare(b, mode)
	})

	return sorted
}
