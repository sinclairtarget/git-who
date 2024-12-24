package main

import (
	"encoding/csv"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

const colWidth = 100

// The "table" subcommand summarizes the authorship history of the given
// commits and paths in a table printed to stdout.
func table(
	revs []string,
	paths []string,
	mode tally.TallyMode,
	useCsv bool,
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

	tallies, err := func() (_ map[string]tally.Tally, err error) {
		commits, closer, err := git.Commits(revs, paths)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err == nil {
				err = closer()
			}
		}()

		tallies, err := tally.TallyCommits(commits)
		if err != nil {
			return nil, err
		}

		return tallies, nil
	}()
	if err != nil {
		return fmt.Errorf("failed to tally commits: %w", err)
	}

	sorted := slices.SortedFunc(
		maps.Values(tallies),
		func(a, b tally.Tally) int {
			aRank := a.SortKey(mode)
			bRank := b.SortKey(mode)

			if aRank > bRank {
				return -1
			} else if aRank == bRank {
				return 0
			} else {
				return 1
			}
		},
	)

	if useCsv {
		err := writeCsv(sorted)
		if err != nil {
			return err
		}
	} else {
		writeTable(sorted)
	}

	return nil
}

func toRecord(t tally.Tally) []string {
	record := make([]string, 0)
	return append(
		record,
		t.AuthorName,
		t.AuthorEmail,
		strconv.Itoa(t.Commits),
		strconv.Itoa(t.LinesAdded),
		strconv.Itoa(t.LinesRemoved),
		strconv.Itoa(t.FileCount),
	)
}

func writeCsv(tallies []tally.Tally) error {
	w := csv.NewWriter(os.Stdout)

	// Write header
	w.Write([]string{
		"name",
		"email",
		"commits",
		"lines added",
		"lines removed",
		"files",
	})

	for _, tally := range tallies {
		record := toRecord(tally)
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

func writeTable(tallies []tally.Tally) {
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
		"│%-27s %-29s %9s %9s %20s│\n",
		"Author",
		"Email",
		"Commits",
		"Files",
		"Lines (+/-)",
	)
	fmt.Printf("├%s┤\n", rule)

	// -- Write table rows --
	for _, tally := range tallies {
		lines := fmt.Sprintf(
			"[32m%9d[0m / [31m%8d[0m",
			tally.LinesAdded,
			tally.LinesRemoved,
		)

		fmt.Printf(
			"│%-27s %-29s %9d %9d %20s│\n",
			abbrev(tally.AuthorName, 27),
			abbrev(tally.AuthorEmail, 29),
			tally.Commits,
			tally.FileCount,
			lines,
		)
	}

	fmt.Printf("└%s┘\n", rule)
}

func abbrev(s string, max int) string {
	if len(s) <= max {
		return s
	}

	return s[:max-3] + "..."
}
