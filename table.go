package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sinclairtarget/git-who/internal/ansi"
	"github.com/sinclairtarget/git-who/internal/format"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

const colWidth = 65 // Width in columns to use by default

// The "table" subcommand summarizes the authorship history of the given
// commits and paths in a table printed to stdout.
func table(
	revs []string,
	paths []string,
	mode tally.TallyMode,
	useCsv bool,
	showEmail bool,
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
	)

	opts := tally.TallyOpts{Mode: mode}
	if showEmail {
		opts.Key = func(c git.Commit) string { return c.AuthorEmail }
	} else {
		opts.Key = func(c git.Commit) string { return c.AuthorName }
	}

	tallies, err := func() (_ []tally.Tally, err error) {
		commits, closer, err := git.Commits(revs, paths)
		if err != nil {
			return nil, err
		}
		defer func() {
			errClose := closer()
			if err == nil {
				err = errClose
			}
		}()

		tallies, err := tally.TallyCommits(commits, opts)
		if err != nil {
			return nil, err
		}

		return tallies, nil
	}()
	if err != nil {
		return fmt.Errorf("failed to tally commits: %w", err)
	}

	if useCsv {
		err := writeCsv(tallies, showEmail)
		if err != nil {
			return err
		}
	} else {
		writeTable(tallies, showEmail)
	}

	return nil
}

func toRecord(t tally.Tally, showEmail bool) []string {
	record := []string{t.AuthorName}

	if showEmail {
		record = append(record, t.AuthorEmail)
	}

	return append(
		record,
		strconv.Itoa(t.Commits),
		strconv.Itoa(t.LinesAdded),
		strconv.Itoa(t.LinesRemoved),
		strconv.Itoa(t.FileCount),
		t.LastCommitTime.Format(time.RFC3339),
	)
}

func writeCsv(tallies []tally.Tally, showEmail bool) error {
	w := csv.NewWriter(os.Stdout)

	// Write header
	if showEmail {
		w.Write([]string{
			"name",
			"email",
			"commits",
			"lines added",
			"lines removed",
			"files",
			"last commit time",
		})
	} else {
		w.Write([]string{
			"name",
			"commits",
			"lines added",
			"lines removed",
			"files",
			"last commit time",
		})
	}

	for _, tally := range tallies {
		record := toRecord(tally, showEmail)
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

func writeTable(tallies []tally.Tally, showEmail bool) {
	if len(tallies) == 0 {
		return
	}

	var build strings.Builder
	for _ = range colWidth - 2 {
		build.WriteRune('─')
	}
	rule := build.String()

	// -- Write header --
	fmt.Printf("┌%s┐\n", rule)
	fmt.Printf(
		"│%-29s %7s %7s %17s│\n",
		"Author",
		"Commits",
		"Files",
		"Lines (+/-)",
	)
	fmt.Printf("├%s┤\n", rule)

	// -- Write table rows --
	for _, tally := range tallies {
		lines := fmt.Sprintf(
			"%s%7d%s / %s%7d%s",
			ansi.Green,
			tally.LinesAdded,
			ansi.Reset,
			ansi.Red,
			tally.LinesRemoved,
			ansi.Reset,
		)

		var author string
		if showEmail {
			author = fmt.Sprintf(
				"%s %s",
				tally.AuthorName,
				format.GitEmail(tally.AuthorEmail),
			)
		} else {
			author = tally.AuthorName
		}

		fmt.Printf(
			"│%-29s %7d %7d %17s│\n",
			format.Abbrev(author, 29),
			tally.Commits,
			tally.FileCount,
			lines,
		)
	}

	fmt.Printf("└%s┘\n", rule)
}
