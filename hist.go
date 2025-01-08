package main

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/sinclairtarget/git-who/internal/ansi"
	"github.com/sinclairtarget/git-who/internal/format"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

const barWidth = 36

func hist(
	revs []string,
	paths []string,
	mode tally.TallyMode,
	showEmail bool,
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

	tallyOpts := tally.TallyOpts{Mode: mode}
	if showEmail {
		tallyOpts.Key = func(c git.Commit) string { return c.AuthorEmail }
	} else {
		tallyOpts.Key = func(c git.Commit) string { return c.AuthorName }
	}

	populateDiffs := tallyOpts.IsDiffMode()
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

	drawPlot(buckets, maxVal, mode, showEmail)
	return nil
}

func drawPlot(
	buckets []tally.TimeBucket,
	maxVal int,
	mode tally.TallyMode,
	showEmail bool,
) {
	var lastAuthor string
	for _, bucket := range buckets {
		value := bucket.Value(mode)
		clampedValue := int(math.Ceil(
			(float64(value) / float64(maxVal)) * float64(barWidth-1),
		))
		bar := strings.Repeat("#", clampedValue)

		if value > 0 {
			tallyPart := fmtHistTally(
				bucket.Tally,
				mode,
				showEmail,
				bucket.Tally.AuthorName == lastAuthor,
			)
			fmt.Printf("%s ┤ %-*s %s\n", bucket.Name, barWidth, bar, tallyPart)

			lastAuthor = bucket.Tally.AuthorName
		} else {
			fmt.Printf("%s ┤ \n", bucket.Name)
		}
	}
}

func fmtHistTally(
	t tally.Tally,
	mode tally.TallyMode,
	showEmail bool,
	fade bool,
) string {
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

	var author string
	if showEmail {
		author = format.Abbrev(format.GitEmail(t.AuthorEmail), 25)
	} else {
		author = format.Abbrev(t.AuthorName, 25)
	}

	if fade {
		return fmt.Sprintf(
			"%s%s %s%s",
			ansi.Dim,
			author,
			metric,
			ansi.Reset,
		)
	} else {
		return fmt.Sprintf("%s %s", author, metric)
	}
}
