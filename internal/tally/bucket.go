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

type bucketFunc func(time.Time) (string, time.Time)

func calcBucketSize(start time.Time, end time.Time) bucketFunc {
	duration := end.Sub(start)
	day := time.Hour * 24
	year := day * 365

	if duration > 5*year {
		// Yearly buckets
		return func(t time.Time) (string, time.Time) {
			year, _, _ := t.UTC().Date()
			bucketedTime := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
			name := bucketedTime.Format("2006")
			return name, bucketedTime
		}
	} else if duration > 60*day {
		// Monthly buckets
		return func(t time.Time) (string, time.Time) {
			year, month, _ := t.UTC().Date()
			bucketedTime := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
			name := bucketedTime.Format("Jan 2006")
			return name, bucketedTime
		}
	} else {
		// Daily buckets
		return func(t time.Time) (string, time.Time) {
			year, month, day := t.UTC().Date()
			bucketedTime := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
			name := bucketedTime.Format(time.DateOnly)
			return name, bucketedTime
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

	sorted := sortTallies(bucket.tallies, mode)
	bucket.Tally = sorted[0]

	return bucket
}

func TallyCommitsByDate(
	commits iter.Seq2[git.Commit, error],
	opts TallyOpts,
	now time.Time,
) ([]TimeBucket, error) {
	buckets := []TimeBucket{}
	var toBucket bucketFunc
	var bucket TimeBucket

	for commit, err := range commits {
		if err != nil {
			return buckets, fmt.Errorf("error iterating commits: %w", err)
		}

		if toBucket == nil {
			toBucket = calcBucketSize(commit.Date, now)
		}

		key := opts.Key(commit)
		name, bucketedTime := toBucket(commit.Date)

		if bucket.Time.IsZero() {
			bucket = newBucket(name, bucketedTime)
		} else if bucketedTime.Sub(bucket.Time) > 0 {
			bucket = finalizeBucket(bucket, opts.Mode)
			buckets = append(buckets, bucket)
			bucket = newBucket(name, bucketedTime)
		}

		tally := bucket.tallies[key]

		tally.AuthorName = commit.AuthorName
		tally.AuthorEmail = commit.AuthorEmail
		tally.Commits += 1
		tally.LastCommitTime = commit.Date

		_, ok := bucket.filesets[key]
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
	}

	bucket = finalizeBucket(bucket, opts.Mode)
	buckets = append(buckets, bucket)
	return buckets, nil
}
