package subcommands

import (
	"context"
	"fmt"
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/sinclairtarget/git-who/internal/cache"
	"github.com/sinclairtarget/git-who/internal/concurrent"
	"github.com/sinclairtarget/git-who/internal/format"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/git/cmd"
	"github.com/sinclairtarget/git-who/internal/git/config"
	"github.com/sinclairtarget/git-who/internal/pretty"
	"github.com/sinclairtarget/git-who/internal/tally"
)

const barWidth = 36

func Hist(
	revs []string,
	pathspecs []string,
	mode tally.TallyMode,
	showEmail bool,
	countMerges bool,
	since string,
	until string,
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
		"pathspecs",
		pathspecs,
		"mode",
		mode,
		"showEmail",
		showEmail,
		"countMerges",
		countMerges,
		"since",
		since,
		"until",
		until,
		"authors",
		authors,
		"nauthors",
		nauthors,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tallyOpts := tally.TallyOpts{Mode: mode, CountMerges: countMerges}
	if showEmail {
		tallyOpts.Key = func(c git.Commit) string { return c.AuthorEmail }
	} else {
		tallyOpts.Key = func(c git.Commit) string { return c.AuthorName }
	}

	populateDiffs := tallyOpts.IsDiffMode()
	filters := cmd.LogFilters{
		Since:    since,
		Until:    until,
		Authors:  authors,
		Nauthors: nauthors,
	}

	var end time.Time // Default is zero time, meaning use last commit
	if len(revs) == 1 && revs[0] == "HEAD" && len(until) == 0 {
		// If no revs or --until given, end timeline at current time
		end = time.Now()
	}

	gitRootPath, err := git.GetRoot()
	if err != nil {
		return err
	}

	configFiles, err := config.DetectSupplementalFiles(gitRootPath)
	if err != nil {
		return err
	}

	var buckets []tally.TimeBucket
	if populateDiffs && runtime.GOMAXPROCS(0) > 1 {
		buckets, err = concurrent.TallyCommitsTimeline(
			ctx,
			revs,
			pathspecs,
			filters,
			configFiles,
			tallyOpts,
			end,
			cache.GetCache(gitRootPath, configFiles),
			pretty.AllowDynamic(os.Stdout),
		)
		if err != nil {
			return err
		}
	} else {
		buckets, err = func() (_ []tally.TimeBucket, err error) {
			commits, finish := git.CommitsWithOpts(
				ctx,
				revs,
				pathspecs,
				filters,
				populateDiffs,
				configFiles,
			)
			defer func() { err = finish() }()

			buckets, err := tally.TallyCommitsTimeline(
				commits,
				tallyOpts,
				end,
			)
			return buckets, err
		}()

		if err != nil {
			return err
		}
	}

	// -- Pick winner in each bucket --
	for i, bucket := range buckets {
		buckets[i] = bucket.Rank(mode)
	}

	// -- Draw bar plot --
	maxVal := barWidth
	for _, bucket := range buckets {
		if bucket.TotalValue(mode) > maxVal {
			maxVal = bucket.TotalValue(mode)
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
			(float64(value) / float64(maxVal)) * float64(barWidth),
		))

		total := bucket.TotalValue(mode)
		clampedTotal := int(math.Ceil(
			(float64(total) / float64(maxVal)) * float64(barWidth),
		))

		valueBar := strings.Repeat("#", clampedValue)
		totalBar := strings.Repeat("-", clampedTotal-clampedValue)

		if value > 0 {
			tallyPart := fmtHistTally(
				bucket.Tally,
				mode,
				showEmail,
				bucket.Tally.AuthorName == lastAuthor,
			)
			fmt.Printf(
				"%s ┤ %s%s%-*s%s  %s\n",
				bucket.Name,
				valueBar,
				pretty.Dim(),
				barWidth-clampedValue,
				totalBar,
				pretty.Reset(),
				tallyPart,
			)

			lastAuthor = bucket.Tally.AuthorName
		} else {
			fmt.Printf("%s ┤ \n", bucket.Name)
		}
	}
}

func fmtHistTally(
	t tally.FinalTally,
	mode tally.TallyMode,
	showEmail bool,
	fade bool,
) string {
	var metric string
	switch mode {
	case tally.CommitMode:
		metric = fmt.Sprintf("(%s)", format.Number(t.Commits))
	case tally.FilesMode:
		metric = fmt.Sprintf("(%s)", format.Number(t.FileCount))
	case tally.LinesMode:
		metric = fmt.Sprintf(
			"(%s%s%s / %s%s%s)",
			pretty.Green(),
			format.Number(t.LinesAdded),
			pretty.DefaultColor(),
			pretty.Red(),
			format.Number(t.LinesRemoved),
			pretty.DefaultColor(),
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
			pretty.Dim(),
			author,
			metric,
			pretty.Reset(),
		)
	} else {
		return fmt.Sprintf("%s %s", author, metric)
	}
}
