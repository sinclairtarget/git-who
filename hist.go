package main

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/sinclairtarget/git-who/internal/ansi"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

const barWidth = 50

func hist(
	revs []string,
	paths []string,
	mode tally.TallyMode,
	since string,
	authors []string,
	nauthors []string,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running \"hist\": %w", err)
		}
	}()

	logger().Debug(
		"called hist()",
		"revs",
		revs,
		"paths",
		paths,
		"since",
		since,
		"authors",
		authors,
		"nauthors",
		nauthors,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	populateDiffs := mode == tally.FilesMode || mode == tally.LinesMode
	filters := git.LogFilters{
		Since:    since,
		Authors:  authors,
		Nauthors: nauthors,
	}
	commits, closer, err := git.CommitsWithOpts(
		ctx,
		revs,
		paths,
		filters,
		populateDiffs,
	)
	if err != nil {
		return err
	}

	tallyOpts := tally.TallyOpts{
		Mode: mode,
		Key:  func(c git.Commit) string { return c.AuthorName },
	}
	buckets, err := tally.TallyCommitsByDate(commits, tallyOpts, time.Now())

	err = closer()
	if err != nil {
		return err
	}

	maxVal := barWidth
	for _, bucket := range buckets {
		if bucket.Value(mode) > maxVal {
			maxVal = bucket.Value(mode)
		}
	}

	drawPlot(buckets, maxVal, mode)
	return nil
}

func drawPlot(buckets []tally.TimeBucket, maxVal int, mode tally.TallyMode) {
	var lastAuthor string
	for _, bucket := range buckets {
		value := bucket.Value(mode)
		clampedValue := int(math.Round(
			(float64(value) / float64(maxVal)) * float64(barWidth),
		))
		bar := strings.Repeat("#", clampedValue)

		tallyPart := fmtHistTally(
			bucket.Tally,
			mode,
			bucket.Tally.AuthorName == lastAuthor,
		)
		fmt.Printf("%s ┤ %-*s %s\n", bucket.Name, barWidth, bar, tallyPart)

		lastAuthor = bucket.Tally.AuthorName
	}
}

func fmtHistTally(t tally.Tally, mode tally.TallyMode, fade bool) string {
	var metric string
	switch mode {
	case tally.CommitMode:
		metric = fmt.Sprintf("(%d)", t.Commits)
	case tally.FilesMode:
		metric = fmt.Sprintf("(%d)", t.FileCount)
	case tally.LinesMode:
		metric = fmt.Sprintf(
			"(%s%d%s / %s%d%s)",
			ansi.Green,
			t.LinesAdded,
			ansi.DefaultColor,
			ansi.Red,
			t.LinesRemoved,
			ansi.DefaultColor,
		)
	default:
		panic("unrecognized tally mode in switch")
	}

	if fade {
		return fmt.Sprintf(
			"%s%s %s%s",
			ansi.Dim,
			t.AuthorName,
			metric,
			ansi.Reset,
		)
	} else {
		return fmt.Sprintf("%s %s", t.AuthorName, metric)
	}
}
