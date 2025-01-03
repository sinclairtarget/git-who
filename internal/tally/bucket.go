package tally

import (
	"fmt"
	"iter"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

type TimeBucket struct {
	Name     string
	Time     time.Time
	Tally    Tally
	tallies  map[string]Tally
	filesets map[string]map[string]bool
}

func (b TimeBucket) Value(mode TallyMode) int {
	switch mode {
	case CommitMode:
		return b.Tally.Commits
	case FilesMode:
		return b.Tally.FileCount
	case LinesMode:
		return b.Tally.LinesAdded + b.Tally.LinesRemoved
	default:
		panic("unrecognized tally mode in switch")
	}
}

func newBucket(name string, t time.Time) TimeBucket {
	return TimeBucket{
		Name:     name,
		Time:     t,
		tallies:  map[string]Tally{},
		filesets: map[string]map[string]bool{},
	}
}

// Resolution for a time series.
//
// apply - Truncate time to its time bucket
// label - Format the date to a label for the bucket
// next - Get next time in series, given a time
type resolution struct {
	apply func(time.Time) time.Time
	label func(time.Time) string
	next  func(time.Time) time.Time
}

func calcResolution(start time.Time, end time.Time) resolution {
	duration := end.Sub(start)
	day := time.Hour * 24
	year := day * 365

	if duration > year*5 {
		// Yearly buckets
		apply := func(t time.Time) time.Time {
			year, _, _ := t.Date()
			return time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
		}
		return resolution{
			apply: apply,
			next: func(t time.Time) time.Time {
				t = apply(t)
				year, _, _ := t.Date()
				return time.Date(year+1, 1, 1, 0, 0, 0, 0, time.Local)
			},
			label: func(t time.Time) string {
				return apply(t).Format("2006")
			},
		}
	} else if duration > day*60 {
		// Monthly buckets
		apply := func(t time.Time) time.Time {
			year, month, _ := t.Date()
			return time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
		}
		return resolution{
			apply: apply,
			next: func(t time.Time) time.Time {
				t = apply(t)
				year, month, _ := t.Date()
				return time.Date(year, month+1, 1, 0, 0, 0, 0, time.Local)
			},
			label: func(t time.Time) string {
				return apply(t).Format("Jan 2006")
			},
		}
	} else {
		// Daily buckets
		apply := func(t time.Time) time.Time {
			year, month, day := t.Date()
			return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
		}
		return resolution{
			apply: apply,
			next: func(t time.Time) time.Time {
				t = apply(t)
				year, month, day := t.Date()
				return time.Date(year, month, day+1, 0, 0, 0, 0, time.Local)
			},
			label: func(t time.Time) string {
				return apply(t).Format(time.DateOnly)
			},
		}
	}
}

func finalizeBucket(bucket TimeBucket, mode TallyMode) TimeBucket {
	// Get count of unique files touched
	for key, tally := range bucket.tallies {
		fileset := bucket.filesets[key]
		tally.FileCount = countFiles(fileset)
		bucket.tallies[key] = tally
	}

	if len(bucket.tallies) > 0 {
		sorted := sortTallies(bucket.tallies, mode)
		bucket.Tally = sorted[0]
	}

	return bucket
}

// Returns a list of "time buckets," with a winning tally for each date.
//
// The resolution / size of the buckets is determined based on the duration
// between the first commit and now.
func TallyCommitsByDate(
	commits iter.Seq2[git.Commit, error],
	opts TallyOpts,
	now time.Time,
) (_ []TimeBucket, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error while tallying commits by date: %w", err)
		}
	}()

	buckets := []TimeBucket{}

	next, stop := iter.Pull2(commits)
	defer stop()

	// Use first commit to calculate resolution
	firstCommit, err, ok := next()
	if err != nil {
		return buckets, err
	}
	if !ok {
		return buckets, nil // Iterator is empty
	}

	resolution := calcResolution(firstCommit.Date, now)

	// Init buckets/timeseries
	t := resolution.apply(firstCommit.Date)
	for now.Sub(t) > 0 {
		bucket := newBucket(resolution.label(t), resolution.apply(t))
		buckets = append(buckets, bucket)
		t = resolution.next(t)
	}

	// Tally
	i := 0
	for {
		commit, err, ok := next()
		if err != nil {
			return buckets, fmt.Errorf("error iterating commits: %w", err)
		}
		if !ok {
			break
		}

		bucket := buckets[i]
		bucketedCommitTime := resolution.apply(commit.Date)

		if bucketedCommitTime.Sub(bucket.Time) > 0 {
			// Next bucket
			buckets[i] = finalizeBucket(bucket, opts.Mode)
			for !bucketedCommitTime.Equal(bucket.Time) {
				i += 1
				bucket = buckets[i]
			}
		}

		key := opts.Key(commit)
		tally := bucket.tallies[key]

		tally.AuthorName = commit.AuthorName
		tally.AuthorEmail = commit.AuthorEmail
		tally.Commits += 1
		tally.LastCommitTime = commit.Date

		_, ok = bucket.filesets[key]
		if !ok {
			bucket.filesets[key] = map[string]bool{}
		}
		for _, diff := range commit.FileDiffs {
			tally.LinesAdded += diff.LinesAdded
			tally.LinesRemoved += diff.LinesRemoved

			if diff.MoveDest != "" {
				moveFile(bucket.filesets, diff)
			} else {
				bucket.filesets[key][diff.Path] = true
			}
		}

		bucket.tallies[key] = tally
		buckets[i] = bucket
	}

	buckets[i] = finalizeBucket(buckets[i], opts.Mode)

	return buckets, nil
}
