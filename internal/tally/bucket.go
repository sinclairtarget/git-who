package tally

import (
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"time"

	"github.com/sinclairtarget/git-who/internal/git"
)

type TimeBucket struct {
	Name       string
	Time       time.Time
	Tally      FinalTally // Winning author's tally
	TotalTally FinalTally // Overall tally for all authors
	tallies    map[string]Tally
}

func newBucket(name string, t time.Time) TimeBucket {
	return TimeBucket{
		Name:    name,
		Time:    t,
		tallies: map[string]Tally{},
	}
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

func (b TimeBucket) TotalValue(mode TallyMode) int {
	switch mode {
	case CommitMode:
		return b.TotalTally.Commits
	case FilesMode:
		return b.TotalTally.FileCount
	case LinesMode:
		return b.TotalTally.LinesAdded + b.TotalTally.LinesRemoved
	default:
		panic("unrecognized tally mode in switch")
	}
}

func (a TimeBucket) Combine(b TimeBucket) TimeBucket {
	if a.Name != b.Name {
		panic("cannot combine buckets whose names do not match")
	}

	if a.Time != b.Time {
		panic("cannot combine buckets whose times do not match")
	}

	merged := a
	for key, tally := range b.tallies {
		existing, ok := a.tallies[key]
		if ok {
			merged.tallies[key] = existing.Combine(tally)
		} else {
			merged.tallies[key] = tally
		}
	}

	return merged
}

func (b TimeBucket) Rank(mode TallyMode) TimeBucket {
	if len(b.tallies) > 0 {
		b.Tally = Rank(b.tallies, mode)[0]

		var runningTally Tally
		for _, tally := range b.tallies {
			runningTally = runningTally.Combine(tally)
		}
		b.TotalTally = runningTally.Final()
	}

	return b
}

type TimeSeries []TimeBucket

func (a TimeSeries) Combine(b TimeSeries) TimeSeries {
	buckets := map[int64]TimeBucket{}
	for _, bucket := range a {
		buckets[bucket.Time.Unix()] = bucket
	}
	for _, bucket := range b {
		existing, ok := buckets[bucket.Time.Unix()]
		if ok {
			buckets[bucket.Time.Unix()] = existing.Combine(bucket)
		} else {
			buckets[bucket.Time.Unix()] = bucket
		}
	}

	sortedKeys := slices.Sorted(maps.Keys(buckets))

	outBuckets := []TimeBucket{}
	for _, key := range sortedKeys {
		outBuckets = append(outBuckets, buckets[key])
	}

	return outBuckets
}

// Resolution for a time series.
//
// apply - Truncate time to its time bucket
// label - Format the date to a label for the bucket
// next - Get next time in series, given a time
type Resolution struct {
	apply func(time.Time) time.Time
	label func(time.Time) string
	next  func(time.Time) time.Time
}

func applyDaily(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

var daily = Resolution{
	apply: applyDaily,
	next: func(t time.Time) time.Time {
		t = applyDaily(t)
		year, month, day := t.Date()
		return time.Date(year, month, day+1, 0, 0, 0, 0, time.Local)
	},
	label: func(t time.Time) string {
		return applyDaily(t).Format(time.DateOnly)
	},
}

func CalcResolution(start time.Time, end time.Time) Resolution {
	duration := end.Sub(start)
	day := time.Hour * 24
	year := day * 365

	if duration > year*5 {
		// Yearly buckets
		apply := func(t time.Time) time.Time {
			year, _, _ := t.Date()
			return time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
		}
		return Resolution{
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
		return Resolution{
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
		return daily
	}
}

// Returns tallies grouped by calendar date.
func TallyCommitsByDate(
	commits iter.Seq[git.Commit],
	opts TallyOpts,
) (_ []TimeBucket, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error while tallying commits by date: %w", err)
		}
	}()

	if opts.Mode == LastModifiedMode || opts.Mode == FirstModifiedMode {
		return nil, errors.New("mode not implemented")
	}

	var (
		minTime time.Time = time.Now()
		maxTime time.Time
	)

	resolution := daily
	buckets := map[int64]TimeBucket{} // Map of (unix) time to bucket

	// Tally
	for commit := range commits {
		bucketedCommitTime := resolution.apply(commit.Date)
		if bucketedCommitTime.Before(minTime) {
			minTime = bucketedCommitTime
		}
		if bucketedCommitTime.After(maxTime) {
			maxTime = bucketedCommitTime
		}

		bucket, ok := buckets[bucketedCommitTime.Unix()]
		if !ok {
			bucket = newBucket(
				resolution.label(bucketedCommitTime),
				resolution.apply(bucketedCommitTime),
			)
		}

		skipMerge := commit.IsMerge && !opts.CountMerges
		if !skipMerge {
			key := opts.Key(commit)

			tally, ok := bucket.tallies[key]
			if !ok {
				tally.name = commit.AuthorName
				tally.email = commit.AuthorEmail
				tally.fileset = map[string]bool{}
			}

			tally.numTallied += 1

			if !commit.IsMerge {
				for _, diff := range commit.FileDiffs {
					tally.added += diff.LinesAdded
					tally.removed += diff.LinesRemoved
					tally.fileset[diff.Path] = true
				}
			}

			bucket.tallies[key] = tally
			buckets[bucket.Time.Unix()] = bucket
		}
	}

	// Turn into slice representing *dense* timeseries
	t := minTime
	bucketSlice := []TimeBucket{}

	for t.Before(maxTime) || t.Equal(maxTime) {
		bucket, ok := buckets[t.Unix()]
		if !ok {
			bucket = newBucket(resolution.label(t), resolution.apply(t))
		}

		bucketSlice = append(bucketSlice, bucket)
		t = resolution.next(t)
	}

	return bucketSlice, nil
}

// Returns a list of "time buckets" with tallies for each date.
//
// The resolution / size of the buckets is determined based on the duration
// between the first commit and end time, if the end-time is non-zero. Otherwise
// the end time is the time of the last commit in chronological order.
func TallyCommitsTimeline(
	commits iter.Seq[git.Commit],
	opts TallyOpts,
	end time.Time,
) ([]TimeBucket, error) {
	buckets, err := TallyCommitsByDate(commits, opts)
	if err != nil {
		return buckets, err
	}

	if len(buckets) == 0 {
		return buckets, err
	}

	if end.IsZero() {
		end = buckets[len(buckets)-1].Time
	}

	resolution := CalcResolution(buckets[0].Time, end)
	rebuckets := Rebucket(buckets, resolution, end)

	return rebuckets, nil
}

func Rebucket(
	buckets []TimeBucket,
	resolution Resolution,
	end time.Time,
) []TimeBucket {
	if len(buckets) < 1 {
		return buckets
	}

	rebuckets := []TimeBucket{}

	// Re-bucket using new resolution
	t := resolution.apply(buckets[0].Time)
	for t.Before(end) || t.Equal(end) {
		bucket := newBucket(resolution.label(t), resolution.apply(t))
		rebuckets = append(rebuckets, bucket)
		t = resolution.next(t)
	}

	i := 0
	for _, bucket := range buckets {
		rebucketedTime := resolution.apply(bucket.Time)
		rebucket := rebuckets[i]
		if rebucketedTime.After(rebucket.Time) {
			// Next bucket, might have to skip some empty ones
			for !rebucketedTime.Equal(rebucket.Time) {
				i += 1
				rebucket = rebuckets[i]
			}
		}

		bucket.Time = rebucket.Time
		bucket.Name = rebucket.Name
		rebuckets[i] = rebuckets[i].Combine(bucket)
	}

	return rebuckets
}
