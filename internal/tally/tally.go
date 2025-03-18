// Handles summations over commits.
package tally

import (
	"fmt"
	"iter"
	"slices"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/utils/timeutils"
)

// Whether we rank authors by commit, lines, or files.
type TallyMode int

const (
	CommitMode TallyMode = iota
	LinesMode
	FilesMode
	LastModifiedMode
	FirstModifiedMode
)

const NoDiffPathname = ".git-who-no-diff-commits"

type TallyOpts struct {
	Mode        TallyMode
	Key         func(c git.Commit) string // Unique ID for author
	CountMerges bool
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
	AuthorName      string
	AuthorEmail     string
	Commits         int // Num commits editing paths in tree by this author
	LinesAdded      int // Num lines added to paths in tree by author
	LinesRemoved    int // Num lines deleted from paths in tree by author
	FileCount       int // Num of file paths in working dir touched by author
	FirstCommitTime time.Time
	LastCommitTime  time.Time
}

func (t FinalTally) SortKey(mode TallyMode) int64 {
	switch mode {
	case CommitMode:
		return int64(t.Commits)
	case FilesMode:
		return int64(t.FileCount)
	case LinesMode:
		return int64(t.LinesAdded + t.LinesRemoved)
	case FirstModifiedMode:
		return -t.FirstCommitTime.Unix()
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
	name            string
	email           string
	commitset       map[string]bool
	added           int
	removed         int
	fileset         map[string]bool
	firstCommitTime time.Time
	lastCommitTime  time.Time
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

func (a Tally) Combine(b Tally) Tally {
	return Tally{
		name:            or(a.name, b.name),
		email:           or(a.email, b.email),
		commitset:       unionInPlace(a.commitset, b.commitset),
		added:           a.added + b.added,
		removed:         a.removed + b.removed,
		fileset:         unionInPlace(a.fileset, b.fileset),
		firstCommitTime: timeutils.Min(a.firstCommitTime, b.firstCommitTime),
		lastCommitTime:  timeutils.Max(a.lastCommitTime, b.lastCommitTime),
		numTallied:      a.numTallied + b.numTallied,
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

	if t.name == "" && t.email == "" {
		panic("tally finalized but has no name and no email")
	}

	return FinalTally{
		AuthorName:      t.name,
		AuthorEmail:     t.email,
		Commits:         commits,
		LinesAdded:      t.added,
		LinesRemoved:    t.removed,
		FileCount:       files,
		FirstCommitTime: t.firstCommitTime,
		LastCommitTime:  t.lastCommitTime,
	}
}

// author -> path -> tally
type TalliesByPath map[string]map[string]Tally

func (left TalliesByPath) Combine(right TalliesByPath) TalliesByPath {
	for key, leftPathTallies := range left {
		rightPathTallies, ok := right[key]
		if !ok {
			rightPathTallies = map[string]Tally{}
		}

		for path, leftTally := range leftPathTallies {
			rightTally, ok := rightPathTallies[path]
			if !ok {
				rightTally.firstCommitTime = time.Unix(1<<62, 0)
			}

			t := leftTally.Combine(rightTally)
			t.numTallied = 1 // Same path
			rightPathTallies[path] = t
		}

		right[key] = rightPathTallies
	}

	return right
}

// Reduce by-path tallies to a single tally for each author.
func (byPath TalliesByPath) Reduce() map[string]Tally {
	tallies := map[string]Tally{}

	for key, pathTallies := range byPath {
		var runningTally Tally
		runningTally.commitset = map[string]bool{}

		for _, tally := range pathTallies {
			runningTally = runningTally.Combine(tally)
		}

		if len(runningTally.commitset) > 0 {
			tallies[key] = runningTally
		}
	}

	return tallies
}

func TallyCommits(
	commits iter.Seq2[git.Commit, error],
	opts TallyOpts,
) (map[string]Tally, error) {
	// Map of author to tally
	var tallies map[string]Tally

	start := time.Now()

	if !opts.IsDiffMode() {
		tallies = map[string]Tally{}

		// Don't need info about file paths, just count commits and commit time
		for commit, err := range commits {
			if err != nil {
				return nil, fmt.Errorf("error iterating commits: %w", err)
			}

			if commit.IsMerge && !opts.CountMerges {
				continue
			}

			key := opts.Key(commit)

			tally, ok := tallies[key]
			if !ok {
				tally.name = commit.AuthorName
				tally.email = commit.AuthorEmail
				tally.firstCommitTime = commit.Date
			}

			tally.numTallied += 1
			tally.firstCommitTime = timeutils.Min(
				commit.Date,
				tally.firstCommitTime,
			)
			tally.lastCommitTime = timeutils.Max(
				commit.Date,
				tally.lastCommitTime,
			)

			tallies[key] = tally
		}
	} else {
		talliesByPath, err := TallyCommitsByPath(commits, opts)
		if err != nil {
			return nil, err
		}

		tallies = talliesByPath.Reduce()
	}

	elapsed := time.Now().Sub(start)
	logger().Debug("tallied commits", "duration_ms", elapsed.Milliseconds())

	return tallies, nil
}

// Tally metrics per author per path.
func TallyCommitsByPath(
	commits iter.Seq2[git.Commit, error],
	opts TallyOpts,
) (TalliesByPath, error) {
	tallies := TalliesByPath{}

	// Tally over commits
	for commit, err := range commits {
		if err != nil {
			return nil, fmt.Errorf("error iterating commits: %w", err)
		}

		if commit.IsMerge && !opts.CountMerges {
			continue
		}

		key := opts.Key(commit)

		pathTallies, ok := tallies[key]
		if !ok {
			pathTallies = map[string]Tally{}
		}

		if len(commit.FileDiffs) == 0 {
			// We still want to count commits that introduce no diff.
			// This could happen with a merge commit that has no diff with its
			// first parent. Have also seen this happen with an SVN-imported
			// commit.
			//
			// We count these commits under a special pathname we hope never
			// collides.
			tally, ok := pathTallies[NoDiffPathname]
			if !ok {
				tally.name = commit.AuthorName
				tally.email = commit.AuthorEmail
				tally.firstCommitTime = commit.Date
				tally.commitset = map[string]bool{}
				tally.numTallied = 0 // Don't count toward files changed
			}

			tally.commitset[commit.ShortHash] = true
			tally.firstCommitTime = timeutils.Min(
				tally.firstCommitTime,
				commit.Date,
			)
			tally.lastCommitTime = timeutils.Max(
				tally.lastCommitTime,
				commit.Date,
			)

			pathTallies[NoDiffPathname] = tally
		} else {
			for _, diff := range commit.FileDiffs {
				tally, ok := pathTallies[diff.Path]
				if !ok {
					tally.name = commit.AuthorName
					tally.email = commit.AuthorEmail
					tally.firstCommitTime = commit.Date
					tally.commitset = map[string]bool{}
				}

				tally.commitset[commit.ShortHash] = true
				tally.firstCommitTime = timeutils.Min(
					tally.firstCommitTime,
					commit.Date,
				)
				tally.lastCommitTime = timeutils.Max(
					tally.lastCommitTime,
					commit.Date,
				)

				if !commit.IsMerge {
					// Only non-merge commits contribute to files / lines
					tally.numTallied = 1
					tally.added += diff.LinesAdded
					tally.removed += diff.LinesRemoved
				}

				pathTallies[diff.Path] = tally
			}
		}

		tallies[key] = pathTallies
	}

	return tallies, nil
}

// Sort tallies according to mode.
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
