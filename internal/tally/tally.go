// Handles summations over commits.
package tally

import (
	"fmt"
	"iter"
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
func (opts TallyOpts) IsDiffMode() bool {
	return opts.Mode == FilesMode || opts.Mode == LinesMode
}

// Metrics tallied for a single author while walking git log.
//
// This kind of tally cannot be combined with others because intermediate
// information has been lost.
type FinalTally struct {
	AuthorName     string
	AuthorEmail    string
	Commits        int // Num commits editing paths in tree by this author
	LinesAdded     int // Num lines added to paths in tree by author
	LinesRemoved   int // Num lines deleted from paths in tree by author
	FileCount      int // Num of file paths in working dir touched by author
	LastCommitTime time.Time
}

func (t FinalTally) SortKey(mode TallyMode) int64 {
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

func (a FinalTally) Compare(b FinalTally, mode TallyMode) int {
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

// A non-final tally that can be combined with other tallies and then finalized
type Tally struct {
	name           string
	email          string
	commitset      map[string]bool
	added          int
	removed        int
	fileset        map[string]bool
	lastCommitTime time.Time
	// Can be used to count Tally objs when we don't need to disambiguate
	numTallied int
}

func or(a, b string) string {
	if a == "" {
		return b
	} else if b == "" {
		return a
	}

	return a
}

func (a Tally) Combine(b Tally) Tally {
	return Tally{
		name:           or(a.name, b.name),
		email:          or(a.email, b.email),
		commitset:      unionInPlace(a.commitset, b.commitset),
		added:          a.added + b.added,
		removed:        a.removed + b.removed,
		fileset:        unionInPlace(a.fileset, b.fileset),
		lastCommitTime: timeutils.Max(a.lastCommitTime, b.lastCommitTime),
		numTallied:     a.numTallied + b.numTallied,
	}
}

func (t Tally) Final() FinalTally {
	commits := t.numTallied // Not using commitset? Fallback to numTallied
	if len(t.commitset) > 0 {
		commits = len(t.commitset)
	}

	files := t.numTallied // Not using fileset? Fallback to numTallied
	if len(t.fileset) > 0 {
		files = len(t.fileset)
	}

	return FinalTally{
		AuthorName:     t.name,
		AuthorEmail:    t.email,
		Commits:        commits,
		LinesAdded:     t.added,
		LinesRemoved:   t.removed,
		FileCount:      files,
		LastCommitTime: t.lastCommitTime,
	}
}

func TallyCommits(
	commits iter.Seq2[git.Commit, error],
	wtreefiles map[string]bool,
	allowOutsideWorktree bool,
	opts TallyOpts,
) (map[string]Tally, error) {
	// Map of author to tally
	tallies := map[string]Tally{}

	start := time.Now()

	if !opts.IsDiffMode() && allowOutsideWorktree {
		// Don't need info about file paths, just count commits and commit time
		for commit, err := range commits {
			if err != nil {
				return nil, fmt.Errorf("error iterating commits: %w", err)
			}

			if commit.IsMerge {
				continue
			}

			key := opts.Key(commit)

			tally, ok := tallies[key]
			if !ok {
				tally.name = commit.AuthorName
				tally.email = commit.AuthorEmail
			}

			tally.numTallied += 1
			tally.lastCommitTime = timeutils.Max(
				commit.Date,
				tally.lastCommitTime,
			)

			tallies[key] = tally
		}
	} else {
		talliesByPath, err := tallyByPath(commits, wtreefiles, opts)
		if err != nil {
			return nil, err
		}

		// Reduce by-path tallies to a single tally for each author, skipping
		// paths outside the worktree if they aren't allowed
		for key, pathTallies := range talliesByPath {
			var runningTally Tally
			runningTally.commitset = map[string]bool{}

			for path, tally := range pathTallies {
				inWTree := wtreefiles[path]
				if inWTree || allowOutsideWorktree {
					runningTally = runningTally.Combine(tally)
				}
			}

			tallies[key] = runningTally
		}
	}

	elapsed := time.Now().Sub(start)
	logger().Debug("tallied commits", "duration_ms", elapsed.Milliseconds())

	return tallies, nil
}

func Rank(tallies map[string]Tally, mode TallyMode) []FinalTally {
	final := []FinalTally{}
	for _, t := range tallies {
		final = append(final, t.Final())
	}

	slices.SortFunc(final, func(a, b FinalTally) int {
		return -a.Compare(b, mode)
	})
	return final
}

// Tally metrics per author per path, regardless of whether in working tree or
// not.
//
// Only handle renames into working tree though.
func tallyByPath(
	commits iter.Seq2[git.Commit, error],
	wtreefiles map[string]bool,
	opts TallyOpts,
) (map[string]map[string]Tally, error) {
	tallies := map[string]map[string]Tally{}

	// Tally over commits
	for commit, err := range commits {
		if err != nil {
			return nil, fmt.Errorf("error iterating commits: %w", err)
		}

		key := opts.Key(commit)
		pathTallies, ok := tallies[key]
		if !ok {
			pathTallies = map[string]Tally{}
			tallies[key] = pathTallies
		}

		for _, diff := range commit.FileDiffs {
			if !commit.IsMerge {
				tally, ok := pathTallies[diff.Path]
				if !ok {
					tally.name = commit.AuthorName
					tally.email = commit.AuthorEmail
					tally.commitset = map[string]bool{}
					tally.numTallied = 1
				}

				tally.commitset[commit.ShortHash] = true
				tally.added += diff.LinesAdded
				tally.removed += diff.LinesRemoved
				tally.lastCommitTime = commit.Date

				pathTallies[diff.Path] = tally
			}

			// If file move would create a file in the working tree, move tally
			// to that path, potentially overwriting, for all authors.
			destInWTree := wtreefiles[diff.MoveDest]
			if destInWTree {
				for _, pathTallies := range tallies {
					oldTally, ok := pathTallies[diff.Path]
					if ok {
						delete(pathTallies, diff.Path)
						pathTallies[diff.MoveDest] = oldTally
					}
				}
			}
		}
	}

	return tallies, nil
}

func unionInPlace(a, b map[string]bool) map[string]bool {
	if a == nil {
		return b
	}

	union := a

	for k, _ := range b {
		union[k] = true
	}

	return union
}
