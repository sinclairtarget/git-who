package main

import (
	"encoding/csv"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

// The "table" subcommand summarizes the authorship history of the given
// commits and path in a table printed to stdout.
func table(revs []string, path string, useCsv bool) error {
	logger().Debug(
		"called table()",
		"revs",
		revs,
		"path",
		path,
		"useCsv",
		useCsv,
	)

	if useCsv == false {
		return fmt.Errorf("generating non-csv table not yet implemented")
	}

	lines, err := git.LogLines(revs, path)
	if err != nil {
		return fmt.Errorf("failed to run git log: %w", err)
	}

	tallies, err := tally.TallyCommits(git.ParseCommits(lines))
	if err != nil {
		return fmt.Errorf("failed to tally commits: %w", err)
	}

	if useCsv {
		writeCsv(tallies)
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

func writeCsv(tallies map[string]tally.Tally) error {
	sorted := slices.SortedFunc(
		maps.Values(tallies),
		func(a, b tally.Tally) int {
			if a.Commits > b.Commits {
				return -1
			} else if a.Commits == b.Commits {
				return 0
			} else {
				return 1
			}
		},
	)

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

	for _, tally := range sorted {
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
