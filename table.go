package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sinclairtarget/git-who/internal/cache"
	cacheBackends "github.com/sinclairtarget/git-who/internal/cache/backends"
	"github.com/sinclairtarget/git-who/internal/concurrent"
	"github.com/sinclairtarget/git-who/internal/format"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/pretty"
	"github.com/sinclairtarget/git-who/internal/tally"
)

const narrowWidth = 55
const wideWidth = 80

func pickWidth(mode tally.TallyMode, showEmail bool) int {
	wideMode := mode == tally.FilesMode || mode == tally.LinesMode
	if wideMode || showEmail {
		return wideWidth
	}

	return narrowWidth
}

// The "table" subcommand summarizes the authorship history of the given
// commits and paths in a table printed to stdout.
func table(
	revs []string,
	paths []string,
	mode tally.TallyMode,
	useCsv bool,
	showEmail bool,
	countMerges bool,
	limit int,
	since string,
	authors []string,
	nauthors []string,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running \"table\": %w", err)
		}
	}()

	logger().Debug(
		"called table()",
		"revs",
		revs,
		"paths",
		paths,
		"mode",
		mode,
		"useCsv",
		useCsv,
		"showEmail",
		showEmail,
		"countMerges",
		countMerges,
		"limit",
		limit,
		"since",
		since,
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
	filters := git.LogFilters{
		Since:    since,
		Authors:  authors,
		Nauthors: nauthors,
	}

	var tallies map[string]tally.Tally
	if populateDiffs && runtime.GOMAXPROCS(0) > 1 {
		tallies, err = concurrent.TallyCommits(
			ctx,
			revs,
			paths,
			filters,
			tallyOpts,
			cache.NewCache(cacheBackends.JSONBackend{Path: "commits.ndjson"}),
			pretty.AllowDynamic(os.Stdout),
		)
		if err != nil {
			return err
		}
	} else {
		// This is fast in the no-diff case even if we don't parallelize it
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

		tallies, err = tally.TallyCommits(commits, tallyOpts)
		if err != nil {
			return fmt.Errorf("failed to tally commits: %w", err)
		}

		err = closer()
		if err != nil {
			return err
		}
	}

	rankedTallies := tally.Rank(tallies, mode)

	numFilteredOut := 0
	if limit > 0 && limit < len(rankedTallies) {
		numFilteredOut = len(rankedTallies) - limit
		rankedTallies = rankedTallies[:limit]
	}

	if useCsv {
		err := writeCsv(rankedTallies, tallyOpts, showEmail)
		if err != nil {
			return err
		}
	} else {
		colwidth := pickWidth(mode, showEmail)
		writeTable(rankedTallies, colwidth, showEmail, mode, numFilteredOut)
	}

	return nil
}

func toRecord(
	t tally.FinalTally,
	opts tally.TallyOpts,
	showEmail bool,
) []string {
	record := []string{t.AuthorName}

	if showEmail {
		record = append(record, t.AuthorEmail)
	}

	record = append(record, strconv.Itoa(t.Commits))

	if opts.IsDiffMode() {
		record = append(
			record,
			strconv.Itoa(t.LinesAdded),
			strconv.Itoa(t.LinesRemoved),
			strconv.Itoa(t.FileCount),
		)
	}

	return append(record, t.LastCommitTime.Format(time.RFC3339))
}

func writeCsv(
	tallies []tally.FinalTally,
	opts tally.TallyOpts,
	showEmail bool,
) error {
	w := csv.NewWriter(os.Stdout)

	// Write header
	columnHeaders := []string{"name"}
	if showEmail {
		columnHeaders = append(columnHeaders, "email")
	}

	columnHeaders = append(columnHeaders, "commits")

	if opts.IsDiffMode() {
		columnHeaders = append(
			columnHeaders,
			"lines added",
			"lines removed",
			"files",
		)
	}

	columnHeaders = append(columnHeaders, "last commit time")
	w.Write(columnHeaders)

	for _, tally := range tallies {
		record := toRecord(tally, opts, showEmail)
		if err := w.Write(record); err != nil {
			return fmt.Errorf("error writing CSV record to stdout: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("error flushing CSV writer: %w", err)
	}

	return nil
}

func writeTable(
	tallies []tally.FinalTally,
	colwidth int,
	showEmail bool,
	mode tally.TallyMode,
	numFilteredOut int,
) {
	if len(tallies) == 0 {
		return
	}

	var build strings.Builder
	for _ = range colwidth - 2 {
		build.WriteRune('─')
	}
	rule := build.String()

	// -- Write header --
	fmt.Printf("┌%s┐\n", rule)

	if mode == tally.CommitMode || mode == tally.LastModifiedMode {
		fmt.Printf(
			"│%-*s %-11s %7s│\n",
			colwidth-22,
			"Author",
			"Last Edit",
			"Commits",
		)
	} else {
		fmt.Printf(
			"│%-*s %-11s %7s %7s  %17s│\n",
			colwidth-36-13,
			"Author",
			"Last Edit",
			"Commits",
			"Files",
			"Lines (+/-)",
		)
	}
	fmt.Printf("├%s┤\n", rule)

	// -- Write table rows --
	for _, t := range tallies {
		lines := fmt.Sprintf(
			"%s%7s%s / %s%7s%s",
			pretty.Green,
			format.Number(t.LinesAdded),
			pretty.Reset,
			pretty.Red,
			format.Number(t.LinesRemoved),
			pretty.Reset,
		)

		var author string
		if showEmail {
			author = fmt.Sprintf(
				"%s %s",
				t.AuthorName,
				format.GitEmail(t.AuthorEmail),
			)
		} else {
			author = t.AuthorName
		}

		if mode == tally.CommitMode || mode == tally.LastModifiedMode {
			fmt.Printf(
				"│%-*s %-11s %7s│\n",
				colwidth-22,
				format.Abbrev(author, colwidth-22),
				format.RelativeTime(progStart, t.LastCommitTime),
				format.Number(t.Commits),
			)
		} else {
			fmt.Printf(
				"│%-*s %-11s %7s %7s  %17s│\n",
				colwidth-36-13,
				format.Abbrev(author, colwidth-36-13),
				format.RelativeTime(progStart, t.LastCommitTime),
				format.Number(t.Commits),
				format.Number(t.FileCount),
				lines,
			)
		}
	}

	if numFilteredOut > 0 {
		msg := fmt.Sprintf("...%s more...", format.Number(numFilteredOut))
		fmt.Printf("│%-*s│\n", colwidth-2, msg)
	}

	fmt.Printf("└%s┘\n", rule)
}
